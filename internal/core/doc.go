// Package core provides the business logic layer for Clonr.
//
// This package contains all core functionality separated from UI concerns.
// Functions in this package handle validation, data transformation, and
// orchestration of operations across multiple subsystems.
//
// # Design Principles
//
//   - Functions return errors instead of printing to stdout/stderr
//   - All database operations go through grpc.GetClient()
//   - UI-specific logic belongs in the cli package, not here
//
// # Clone Operations
//
// Clone operations are split into two phases:
//
//  1. [PrepareClonePath] - Validates URL and determines the local path
//  2. [SaveClonedRepo] - Persists the repository to the database after git clone
//
// This split allows the Bubbletea UI to handle the git clone progress display
// while core handles the validation and persistence logic.
//
// # Organization Operations
//
// The package provides functions for managing GitHub organization repositories:
//   - Fetching organization repository lists
//   - Mirroring repositories locally
//   - Tracking cloned organization repositories
package core
