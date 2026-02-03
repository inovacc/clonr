-- Migration: 001_initial_schema
-- Description: Initial database schema for clonr
-- Created: 2026-02-03

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    description TEXT
);

-- Repositories table
CREATE TABLE IF NOT EXISTS repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uid TEXT UNIQUE NOT NULL,
    url TEXT UNIQUE NOT NULL,
    path TEXT UNIQUE NOT NULL,
    workspace TEXT DEFAULT '',
    favorite INTEGER DEFAULT 0,
    cloned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_checked DATETIME
);

CREATE INDEX IF NOT EXISTS idx_repositories_url ON repositories(url);
CREATE INDEX IF NOT EXISTS idx_repositories_path ON repositories(path);
CREATE INDEX IF NOT EXISTS idx_repositories_workspace ON repositories(workspace);
CREATE INDEX IF NOT EXISTS idx_repositories_favorite ON repositories(favorite);

-- Configuration table (single row)
CREATE TABLE IF NOT EXISTS config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    default_clone_dir TEXT DEFAULT '',
    editor TEXT DEFAULT '',
    terminal TEXT DEFAULT '',
    monitor_interval INTEGER DEFAULT 300,
    server_port INTEGER DEFAULT 4000,
    custom_editors TEXT DEFAULT '[]',  -- JSON array
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default config row
INSERT OR IGNORE INTO config (id) VALUES (1);

-- Profiles table (sensitive data stored encrypted)
CREATE TABLE IF NOT EXISTS profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    host TEXT DEFAULT 'github.com',
    username TEXT DEFAULT '',
    token_storage TEXT DEFAULT 'encrypted',  -- 'encrypted' or 'open'
    scopes TEXT DEFAULT '[]',  -- JSON array
    is_default INTEGER DEFAULT 0,
    encrypted_token BLOB,  -- Binary encrypted token
    workspace TEXT DEFAULT '',
    notify_channels TEXT DEFAULT '[]',  -- JSON array of NotifyChannel
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_profiles_name ON profiles(name);
CREATE INDEX IF NOT EXISTS idx_profiles_default ON profiles(is_default);

-- Workspaces table
CREATE TABLE IF NOT EXISTS workspaces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT DEFAULT '',
    path TEXT DEFAULT '',
    is_active INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workspaces_name ON workspaces(name);
CREATE INDEX IF NOT EXISTS idx_workspaces_active ON workspaces(is_active);

-- Standalone configuration table (single row for server mode)
CREATE TABLE IF NOT EXISTS standalone_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER DEFAULT 0,
    is_server INTEGER DEFAULT 0,
    instance_id TEXT DEFAULT '',
    port INTEGER DEFAULT 50052,
    api_key_hash BLOB,
    refresh_token BLOB,
    salt BLOB,
    capabilities TEXT DEFAULT '[]',  -- JSON array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME
);

-- Standalone clients table (for server mode - connected clients)
CREATE TABLE IF NOT EXISTS standalone_clients (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    machine_info TEXT DEFAULT '{}',  -- JSON
    connected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_sync DATETIME
);

-- Standalone connections table (for client mode - connections to servers)
CREATE TABLE IF NOT EXISTS standalone_connections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    instance_id TEXT DEFAULT '',
    host TEXT NOT NULL,
    port INTEGER DEFAULT 50052,
    api_key_encrypted BLOB,
    refresh_token_encrypted BLOB,
    local_password_hash BLOB,
    local_salt BLOB,
    sync_status TEXT DEFAULT 'disconnected',  -- 'connected', 'disconnected', 'error'
    synced_items TEXT DEFAULT '{}',  -- JSON
    last_sync DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_standalone_connections_name ON standalone_connections(name);

-- Server encryption configuration (single row)
CREATE TABLE IF NOT EXISTS server_encryption_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER DEFAULT 0,
    key_hash BLOB,
    salt BLOB,
    key_hint TEXT DEFAULT '',
    configured_at DATETIME
);

-- Synced data table (encrypted data from remote instances)
CREATE TABLE IF NOT EXISTS synced_data (
    id TEXT PRIMARY KEY,
    connection_name TEXT NOT NULL,
    instance_id TEXT DEFAULT '',
    data_type TEXT NOT NULL,  -- 'profile', 'workspace', 'repo', 'config'
    name TEXT NOT NULL,
    encrypted_data BLOB,
    nonce BLOB,
    state TEXT DEFAULT 'encrypted',  -- 'encrypted', 'decrypted', 'pending'
    checksum TEXT DEFAULT '',
    synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    decrypted_at DATETIME,
    UNIQUE(connection_name, data_type, name)
);

CREATE INDEX IF NOT EXISTS idx_synced_data_connection ON synced_data(connection_name);
CREATE INDEX IF NOT EXISTS idx_synced_data_state ON synced_data(state);
CREATE INDEX IF NOT EXISTS idx_synced_data_type ON synced_data(data_type);

-- Pending registrations table (server side - clients awaiting approval)
CREATE TABLE IF NOT EXISTS pending_registrations (
    client_id TEXT PRIMARY KEY,
    client_name TEXT NOT NULL,
    machine_info TEXT DEFAULT '{}',  -- JSON
    state TEXT DEFAULT 'initiated',  -- 'initiated', 'challenged', 'key_generated', 'key_pending', 'completed', 'rejected'
    challenge_token TEXT DEFAULT '',
    challenge_at DATETIME,
    initiated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_pending_registrations_state ON pending_registrations(state);

-- Registered clients table (server side - approved clients)
CREATE TABLE IF NOT EXISTS registered_clients (
    client_id TEXT PRIMARY KEY,
    client_name TEXT NOT NULL,
    machine_info TEXT DEFAULT '{}',  -- JSON
    encryption_key_hash BLOB,
    encryption_salt BLOB,
    key_hint TEXT DEFAULT '',
    status TEXT DEFAULT 'active',  -- 'active', 'suspended', 'revoked'
    sync_count INTEGER DEFAULT 0,
    last_ip TEXT DEFAULT '',
    registered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_registered_clients_status ON registered_clients(status);

-- Record this migration
INSERT INTO schema_migrations (version, description) VALUES (1, 'Initial schema');
