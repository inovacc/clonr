// Package model defines the data structures used throughout Clonr.
//
// This package contains the core domain models that represent the application's
// data. These models are used by both the database layer and the gRPC layer,
// with appropriate conversions handled by each package.
//
// # Repository
//
// The [Repository] struct represents a Git repository tracked by Clonr:
//
//	type Repository struct {
//	    UID         string    // Unique identifier (UUID)
//	    URL         string    // Remote repository URL
//	    Path        string    // Local filesystem path
//	    Favorite    bool      // Whether marked as favorite
//	    ClonedAt    time.Time // When the repository was cloned
//	    UpdatedAt   time.Time // Last update timestamp
//	    LastChecked time.Time // Last time checked for updates
//	}
//
// # Config
//
// The [Config] struct holds application configuration:
//
//	type Config struct {
//	    DefaultCloneDir string // Base directory for cloning repositories
//	    Editor          string // Preferred editor command
//	    Terminal        string // Preferred terminal emulator
//	    MonitorInterval int    // Interval for checking updates (seconds)
//	    ServerPort      int    // gRPC server port
//	}
package model
