// Package database provides the storage abstraction layer for Clonr.
//
// The package defines the [Store] interface which abstracts all database
// operations, allowing different storage backends to be used interchangeably.
// Currently supported backends are BoltDB (default) and SQLite.
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
//	db := database.GetDB()
//	repos, err := db.GetAllRepos()
//
// The database backend is selected at build time using build tags:
//   - Default: BoltDB
//   - With -tags sqlite: SQLite via GORM
//
// # Server-Side Only
//
// This package should only be used by server-side code (internal/grpcserver).
// Client-side code should use the grpcclient package instead, which forwards
// operations to the server via gRPC.
package database
