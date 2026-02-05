package standalone

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/stun"
)

// Default STUN servers for public IP discovery
var DefaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun2.l.google.com:19302",
	"stun.cloudflare.com:3478",
	"stun.stunprotocol.org:3478",
}

// STUNResult contains the result of a STUN query
type STUNResult struct {
	PublicIP   net.IP
	PublicPort int
	LocalIP    net.IP
	LocalPort  int
	NATType    NATType
	Server     string
	Latency    time.Duration
}

// NATType represents the type of NAT detected
type NATType string

const (
	NATTypeUnknown           NATType = "unknown"
	NATTypeNone              NATType = "none"            // Direct public IP
	NATTypeFullCone          NATType = "full_cone"       // Endpoint-independent mapping
	NATTypeRestrictedCone    NATType = "restricted_cone" // Address-restricted
	NATTypePortRestricted    NATType = "port_restricted" // Port-restricted
	NATTypeSymmetric         NATType = "symmetric"       // Different mapping per destination
	NATTypeSymmetricFirewall NATType = "symmetric_firewall"
)

// STUNClient provides STUN-based network discovery
type STUNClient struct {
	servers  []string
	timeout  time.Duration
	mu       sync.RWMutex
	cache    *STUNResult
	cacheAt  time.Time
	cacheTTL time.Duration
}

// NewSTUNClient creates a new STUN client
func NewSTUNClient(servers ...string) *STUNClient {
	if len(servers) == 0 {
		servers = DefaultSTUNServers
	}

	return &STUNClient{
		servers:  servers,
		timeout:  5 * time.Second,
		cacheTTL: 5 * time.Minute,
	}
}

// WithTimeout sets the timeout for STUN queries
func (c *STUNClient) WithTimeout(d time.Duration) *STUNClient {
	c.timeout = d
	return c
}

// WithCacheTTL sets the cache TTL for STUN results
func (c *STUNClient) WithCacheTTL(d time.Duration) *STUNClient {
	c.cacheTTL = d
	return c
}

// DiscoverPublicIP discovers the public IP using STUN
func (c *STUNClient) DiscoverPublicIP(ctx context.Context) (*STUNResult, error) {
	// Check cache first
	c.mu.RLock()

	if c.cache != nil && time.Since(c.cacheAt) < c.cacheTTL {
		result := *c.cache
		c.mu.RUnlock()

		return &result, nil
	}

	c.mu.RUnlock()

	// Try each STUN server
	var lastErr error

	for _, server := range c.servers {
		result, err := c.querySTUNServer(ctx, server)
		if err != nil {
			lastErr = err
			continue
		}

		// Cache the result
		c.mu.Lock()
		c.cache = result
		c.cacheAt = time.Now()
		c.mu.Unlock()

		return result, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all STUN servers failed, last error: %w", lastErr)
	}

	return nil, fmt.Errorf("no STUN servers available")
}

// querySTUNServer queries a single STUN server
func (c *STUNClient) querySTUNServer(ctx context.Context, server string) (*STUNResult, error) {
	// Create a deadline context
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Resolve the STUN server address
	addr, err := net.ResolveUDPAddr("udp4", server)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve STUN server %s: %w", server, err)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to STUN server %s: %w", server, err)
	}

	defer func() { _ = conn.Close() }()

	// Get local address
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	// Build STUN binding request
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	start := time.Now()

	// Set deadline
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	// Send request
	if _, err := conn.Write(message.Raw); err != nil {
		return nil, fmt.Errorf("failed to send STUN request: %w", err)
	}

	// Read response
	buf := make([]byte, 1500)

	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read STUN response: %w", err)
	}

	latency := time.Since(start)

	// Parse response
	response := new(stun.Message)

	response.Raw = buf[:n]
	if err := response.Decode(); err != nil {
		return nil, fmt.Errorf("failed to decode STUN response: %w", err)
	}

	// Extract XOR-MAPPED-ADDRESS
	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(response); err != nil {
		// Try MAPPED-ADDRESS as fallback
		var mappedAddr stun.MappedAddress
		if err := mappedAddr.GetFrom(response); err != nil {
			return nil, fmt.Errorf("no mapped address in STUN response: %w", err)
		}

		return &STUNResult{
			PublicIP:   mappedAddr.IP,
			PublicPort: mappedAddr.Port,
			LocalIP:    localAddr.IP,
			LocalPort:  localAddr.Port,
			NATType:    c.determineNATType(localAddr.IP, mappedAddr.IP),
			Server:     server,
			Latency:    latency,
		}, nil
	}

	return &STUNResult{
		PublicIP:   xorAddr.IP,
		PublicPort: xorAddr.Port,
		LocalIP:    localAddr.IP,
		LocalPort:  localAddr.Port,
		NATType:    c.determineNATType(localAddr.IP, xorAddr.IP),
		Server:     server,
		Latency:    latency,
	}, nil
}

// determineNATType provides a basic NAT type determination
func (c *STUNClient) determineNATType(localIP, publicIP net.IP) NATType {
	if localIP.Equal(publicIP) {
		return NATTypeNone
	}
	// More sophisticated NAT type detection would require multiple STUN queries
	// to different servers and comparing the results
	return NATTypeUnknown
}

// DetectNATType performs comprehensive NAT type detection
// This requires querying multiple STUN servers and comparing results
func (c *STUNClient) DetectNATType(ctx context.Context) (NATType, error) {
	if len(c.servers) < 2 {
		return NATTypeUnknown, fmt.Errorf("NAT type detection requires at least 2 STUN servers")
	}

	// Query first server
	result1, err := c.querySTUNServer(ctx, c.servers[0])
	if err != nil {
		return NATTypeUnknown, fmt.Errorf("first STUN query failed: %w", err)
	}

	// Check if we have a direct public IP
	if result1.LocalIP.Equal(result1.PublicIP) {
		return NATTypeNone, nil
	}

	// Query second server
	result2, err := c.querySTUNServer(ctx, c.servers[1])
	if err != nil {
		return NATTypeUnknown, fmt.Errorf("second STUN query failed: %w", err)
	}

	// Compare results
	if result1.PublicIP.Equal(result2.PublicIP) && result1.PublicPort == result2.PublicPort {
		// Same external endpoint - could be full cone, restricted cone, or port restricted
		// Would need additional tests to distinguish
		return NATTypeFullCone, nil
	}

	// Different external endpoints - symmetric NAT
	return NATTypeSymmetric, nil
}

// GetPublicAddress returns a formatted public address string
func (r *STUNResult) GetPublicAddress() string {
	return fmt.Sprintf("%s:%d", r.PublicIP.String(), r.PublicPort)
}

// IsNATted returns true if behind NAT
func (r *STUNResult) IsNATted() bool {
	return !r.LocalIP.Equal(r.PublicIP)
}

// CanAcceptDirectConnections returns true if the NAT type allows incoming connections
func (r *STUNResult) CanAcceptDirectConnections() bool {
	switch r.NATType {
	case NATTypeNone, NATTypeFullCone:
		return true
	default:
		return false
	}
}
