package standalone

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/ice/v3"
	"github.com/pion/stun/v2"
)

// ICEConfig contains configuration for ICE connectivity
type ICEConfig struct {
	STUNServers    []string
	TURNServers    []TURNServer
	LocalUfrag     string
	LocalPwd       string
	IsControlling  bool
	Timeout        time.Duration
	NetworkTypes   []ice.NetworkType
}

// TURNServer represents a TURN server configuration
type TURNServer struct {
	URL        string
	Username   string
	Credential string
}

// ICECandidate represents an ICE candidate for connectivity
type ICECandidate struct {
	Type       string `json:"type"`       // "host", "srflx", "relay"
	Foundation string `json:"foundation"`
	Component  int    `json:"component"`
	Protocol   string `json:"protocol"`   // "udp", "tcp"
	Priority   uint32 `json:"priority"`
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	RelatedIP  string `json:"related_ip,omitempty"`
	RelatedPort int   `json:"related_port,omitempty"`
}

// ICECredentials contains the local ICE credentials to share with peer
type ICECredentials struct {
	Ufrag      string         `json:"ufrag"`
	Pwd        string         `json:"pwd"`
	Candidates []ICECandidate `json:"candidates"`
}

// ICEAgent manages ICE connectivity for a standalone instance
type ICEAgent struct {
	agent      *ice.Agent
	config     ICEConfig
	conn       net.Conn
	mu         sync.RWMutex
	candidates []ICECandidate
	onCandidate func(ICECandidate)
	connected   chan struct{}
}

// DefaultICEConfig returns a default ICE configuration
func DefaultICEConfig() ICEConfig {
	return ICEConfig{
		STUNServers: DefaultSTUNServers,
		Timeout:     30 * time.Second,
		NetworkTypes: []ice.NetworkType{
			ice.NetworkTypeUDP4,
			ice.NetworkTypeUDP6,
		},
	}
}

// NewICEAgent creates a new ICE agent for connectivity
func NewICEAgent(config ICEConfig) (*ICEAgent, error) {
	// Build URL list for ICE agent
	var urls []*stun.URI
	for _, server := range config.STUNServers {
		u, err := stun.ParseURI(fmt.Sprintf("stun:%s", server))
		if err != nil {
			continue
		}
		urls = append(urls, u)
	}

	// Add TURN servers
	for _, turn := range config.TURNServers {
		u, err := stun.ParseURI(turn.URL)
		if err != nil {
			continue
		}
		u.Username = turn.Username
		u.Password = turn.Credential
		urls = append(urls, u)
	}

	// Create ICE agent configuration
	agentConfig := &ice.AgentConfig{
		Urls:         urls,
		NetworkTypes: config.NetworkTypes,
	}

	// Set local credentials if provided
	if config.LocalUfrag != "" && config.LocalPwd != "" {
		agentConfig.LocalUfrag = config.LocalUfrag
		agentConfig.LocalPwd = config.LocalPwd
	}

	// Create the ICE agent
	agent, err := ice.NewAgent(agentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ICE agent: %w", err)
	}

	return &ICEAgent{
		agent:     agent,
		config:    config,
		connected: make(chan struct{}),
	}, nil
}

// GatherCandidates gathers local ICE candidates
func (a *ICEAgent) GatherCandidates(ctx context.Context) (*ICECredentials, error) {
	// Set up candidate handler
	candidateChan := make(chan *ice.Candidate, 10)

	if err := a.agent.OnCandidate(func(c ice.Candidate) {
		if c != nil {
			candidateChan <- &c
		} else {
			close(candidateChan)
		}
	}); err != nil {
		return nil, fmt.Errorf("failed to set candidate handler: %w", err)
	}

	// Start gathering
	if err := a.agent.GatherCandidates(); err != nil {
		return nil, fmt.Errorf("failed to start gathering: %w", err)
	}

	// Collect candidates with timeout
	var candidates []ICECandidate
	timeoutCtx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	for {
		select {
		case c, ok := <-candidateChan:
			if !ok {
				// Gathering complete
				goto done
			}
			if c != nil {
				candidate := a.candidateToICECandidate(*c)
				candidates = append(candidates, candidate)

				a.mu.Lock()
				a.candidates = append(a.candidates, candidate)
				if a.onCandidate != nil {
					a.onCandidate(candidate)
				}
				a.mu.Unlock()
			}
		case <-timeoutCtx.Done():
			goto done
		}
	}

done:
	// Get local credentials
	ufrag, pwd, err := a.agent.GetLocalUserCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get local credentials: %w", err)
	}

	return &ICECredentials{
		Ufrag:      ufrag,
		Pwd:        pwd,
		Candidates: candidates,
	}, nil
}

// candidateToICECandidate converts pion ice.Candidate to our ICECandidate
func (a *ICEAgent) candidateToICECandidate(c ice.Candidate) ICECandidate {
	candidate := ICECandidate{
		Type:       c.Type().String(),
		Foundation: c.Foundation(),
		Component:  int(c.Component()),
		Protocol:   c.NetworkType().NetworkShort(),
		Priority:   c.Priority(),
		IP:         c.Address(),
		Port:       c.Port(),
	}

	if c.RelatedAddress() != nil {
		candidate.RelatedIP = c.RelatedAddress().Address
		candidate.RelatedPort = c.RelatedAddress().Port
	}

	return candidate
}

// Connect establishes a connection with a remote peer using their ICE credentials
func (a *ICEAgent) Connect(ctx context.Context, remote *ICECredentials) (net.Conn, error) {
	// Set remote credentials
	if err := a.agent.SetRemoteCredentials(remote.Ufrag, remote.Pwd); err != nil {
		return nil, fmt.Errorf("failed to set remote credentials: %w", err)
	}

	// Add remote candidates
	for _, c := range remote.Candidates {
		candidate, err := a.iceCandidateToCandidate(c)
		if err != nil {
			continue // Skip invalid candidates
		}
		if err := a.agent.AddRemoteCandidate(candidate); err != nil {
			continue
		}
	}

	// Start connectivity checks
	conn, err := a.agent.Dial(ctx, remote.Ufrag, remote.Pwd)
	if err != nil {
		return nil, fmt.Errorf("ICE connectivity failed: %w", err)
	}

	a.mu.Lock()
	a.conn = conn
	close(a.connected)
	a.mu.Unlock()

	return conn, nil
}

// Accept waits for an incoming connection from a remote peer
func (a *ICEAgent) Accept(ctx context.Context, remote *ICECredentials) (net.Conn, error) {
	// Set remote credentials
	if err := a.agent.SetRemoteCredentials(remote.Ufrag, remote.Pwd); err != nil {
		return nil, fmt.Errorf("failed to set remote credentials: %w", err)
	}

	// Add remote candidates
	for _, c := range remote.Candidates {
		candidate, err := a.iceCandidateToCandidate(c)
		if err != nil {
			continue
		}
		if err := a.agent.AddRemoteCandidate(candidate); err != nil {
			continue
		}
	}

	// Accept connection
	conn, err := a.agent.Accept(ctx, remote.Ufrag, remote.Pwd)
	if err != nil {
		return nil, fmt.Errorf("ICE accept failed: %w", err)
	}

	a.mu.Lock()
	a.conn = conn
	close(a.connected)
	a.mu.Unlock()

	return conn, nil
}

// iceCandidateToCandidate converts our ICECandidate to pion ice.Candidate
func (a *ICEAgent) iceCandidateToCandidate(c ICECandidate) (ice.Candidate, error) {
	candidateType := ice.CandidateTypeHost
	switch c.Type {
	case "host":
		candidateType = ice.CandidateTypeHost
	case "srflx":
		candidateType = ice.CandidateTypeServerReflexive
	case "prflx":
		candidateType = ice.CandidateTypePeerReflexive
	case "relay":
		candidateType = ice.CandidateTypeRelay
	}

	config := ice.CandidateConfig{
		CandidateID: "",
		NetworkType: ice.NetworkTypeUDP4,
		Address:     c.IP,
		Port:        c.Port,
		Component:   uint16(c.Component),
		Priority:    c.Priority,
		Foundation:  c.Foundation,
	}

	if c.RelatedIP != "" {
		config.RelatedAddress = &ice.CandidateRelatedAddress{
			Address: c.RelatedIP,
			Port:    c.RelatedPort,
		}
	}

	switch candidateType {
	case ice.CandidateTypeHost:
		return ice.NewCandidateHost(&config)
	case ice.CandidateTypeServerReflexive:
		return ice.NewCandidateServerReflexive(&config)
	case ice.CandidateTypePeerReflexive:
		return ice.NewCandidatePeerReflexive(&config)
	case ice.CandidateTypeRelay:
		return ice.NewCandidateRelay(&config)
	default:
		return nil, fmt.Errorf("unknown candidate type: %s", c.Type)
	}
}

// Close closes the ICE agent and any established connection
func (a *ICEAgent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	var err error
	if a.conn != nil {
		err = a.conn.Close()
	}
	if a.agent != nil {
		if closeErr := a.agent.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

// GetLocalCandidates returns the gathered local candidates
func (a *ICEAgent) GetLocalCandidates() []ICECandidate {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]ICECandidate, len(a.candidates))
	copy(result, a.candidates)
	return result
}

// OnCandidate sets a callback for when new candidates are discovered
func (a *ICEAgent) OnCandidate(fn func(ICECandidate)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onCandidate = fn
}

// WaitConnected waits for the connection to be established
func (a *ICEAgent) WaitConnected(ctx context.Context) error {
	select {
	case <-a.connected:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SerializeCredentials serializes ICE credentials to JSON
func (creds *ICECredentials) Serialize() ([]byte, error) {
	return json.Marshal(creds)
}

// ParseCredentials parses ICE credentials from JSON
func ParseCredentials(data []byte) (*ICECredentials, error) {
	var creds ICECredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse ICE credentials: %w", err)
	}
	return &creds, nil
}
