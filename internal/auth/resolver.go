// Package auth provides a generic token resolution framework.
// It supports multiple token sources with configurable priority order.
package auth

import (
	"fmt"
	"os"
	"strings"
)

// Source indicates where a token was found
type Source string

const (
	SourceFlag    Source = "flag"
	SourceEnv     Source = "env"
	SourceConfig  Source = "config"
	SourceProfile Source = "profile"
	SourceCLI     Source = "cli"
	SourceNone    Source = "none"
)

// Result contains the resolved token and its source
type Result struct {
	Token  string
	Source Source
	Name   string // The specific source name (e.g., "GITHUB_TOKEN", "profile:default")
}

// TokenProvider is a function that attempts to provide a token.
// Returns the token and source name if found, or empty string if not available.
// Returns an error only for unexpected failures (not for missing token).
type TokenProvider func() (token string, sourceName string, err error)

// Resolver resolves tokens from multiple sources in priority order
type Resolver struct {
	providers   []TokenProvider
	serviceName string
	helpMessage string
}

// NewResolver creates a new token resolver for a service
func NewResolver(serviceName string) *Resolver {
	return &Resolver{
		serviceName: serviceName,
		providers:   make([]TokenProvider, 0),
	}
}

// WithFlag adds a flag-provided token as a source (highest priority).
// The flag value is evaluated at resolution time.
func (r *Resolver) WithFlag(flagValue *string) *Resolver {
	r.providers = append(r.providers, func() (string, string, error) {
		if flagValue != nil && *flagValue != "" {
			return *flagValue, "flag", nil
		}
		return "", "", nil
	})
	return r
}

// WithFlagValue adds a flag value directly (for when value is already known)
func (r *Resolver) WithFlagValue(value string) *Resolver {
	r.providers = append(r.providers, func() (string, string, error) {
		if value != "" {
			return value, "flag", nil
		}
		return "", "", nil
	})
	return r
}

// WithEnv adds an environment variable as a token source
func (r *Resolver) WithEnv(envVar string) *Resolver {
	r.providers = append(r.providers, func() (string, string, error) {
		if token := os.Getenv(envVar); token != "" {
			return token, envVar, nil
		}
		return "", "", nil
	})
	return r
}

// WithEnvs adds multiple environment variables as token sources (checked in order)
func (r *Resolver) WithEnvs(envVars ...string) *Resolver {
	for _, envVar := range envVars {
		r.WithEnv(envVar)
	}
	return r
}

// WithProvider adds a custom token provider
func (r *Resolver) WithProvider(provider TokenProvider) *Resolver {
	r.providers = append(r.providers, provider)
	return r
}

// WithHelpMessage sets the help message shown when no token is found
func (r *Resolver) WithHelpMessage(msg string) *Resolver {
	r.helpMessage = msg
	return r
}

// Resolve attempts to find a token from all configured sources in order.
// Returns the first successful token found, or an error if no token is available.
func (r *Resolver) Resolve() (*Result, error) {
	for _, provider := range r.providers {
		token, sourceName, err := provider()
		if err != nil {
			return nil, fmt.Errorf("token provider error: %w", err)
		}
		if token != "" {
			return &Result{
				Token:  token,
				Source: categorizeSource(sourceName),
				Name:   sourceName,
			}, nil
		}
	}

	// No token found
	if r.helpMessage != "" {
		return nil, fmt.Errorf("%s token required\n\n%s", r.serviceName, r.helpMessage)
	}
	return nil, fmt.Errorf("%s token required", r.serviceName)
}

// MustResolve is like Resolve but panics on error
func (r *Resolver) MustResolve() *Result {
	result, err := r.Resolve()
	if err != nil {
		panic(err)
	}
	return result
}

// categorizeSource determines the Source category from a source name
func categorizeSource(name string) Source {
	switch {
	case name == "flag":
		return SourceFlag
	case strings.HasPrefix(name, "profile"):
		return SourceProfile
	case strings.HasPrefix(name, "cli"):
		return SourceCLI
	case name == "config" || strings.HasSuffix(name, ".json"):
		return SourceConfig
	case strings.Contains(name, "_") || strings.Contains(name, "TOKEN"):
		return SourceEnv
	default:
		return SourceNone
	}
}

// EnvOrDefault returns the value of an environment variable or a default
func EnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// EnvOrFile tries to get a value from env var, falling back to reading a file
func EnvOrFile(envVar, filePath string) (string, error) {
	if token := os.Getenv(envVar); token != "" {
		return token, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return strings.TrimSpace(string(data)), nil
}
