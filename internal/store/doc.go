// Package database provides the storage abstraction layer for Clonr.
//
// The package defines the [Store] interface which abstracts all database
// operations. The storage backend is BoltDB, an embedded key-value store.
//
// # Store Interface
//
// The [Store] interface defines methods for:
//   - Repository CRUD operations (SaveRepo, GetAllRepos, RemoveRepoByURL, etc.)
//   - Repository queries (RepoExistsByURL, RepoExistsByPath)
//   - Configuration management (GetConfig, SaveConfig)
//
// # Singleton Pattern
//
// Use [GetDB] to obtain the singleton database instance:
//
//	storage := database.GetDB()
//	repos, err := storage.GetAllRepos()
//
// # Server-Side Only
//
// This package should only be used by server-side code (internal/grpcserver).
// Client-side code should use the grpcclient package instead, which forwards
// operations to the server via gRPC.
package store
