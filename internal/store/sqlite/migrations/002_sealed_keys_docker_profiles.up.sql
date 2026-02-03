-- Migration: 002_sealed_keys_docker_profiles
-- Description: Add sealed keys and docker profiles tables
-- Created: 2026-02-03

-- Sealed keys table (TPM-sealed encryption keys stored in database)
CREATE TABLE IF NOT EXISTS sealed_keys (
    id INTEGER PRIMARY KEY CHECK (id = 1),  -- Single sealed key per database
    sealed_data BLOB NOT NULL,               -- TPM-sealed data blob
    version INTEGER DEFAULT 1,               -- Version for future format changes
    key_type TEXT DEFAULT 'tpm',             -- 'tpm', 'password', 'software'
    metadata TEXT DEFAULT '{}',              -- JSON: PCR values, hints, etc.
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    rotated_at DATETIME,
    last_accessed DATETIME
);

-- Docker profiles table (container registry credentials)
CREATE TABLE IF NOT EXISTS docker_profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    registry TEXT DEFAULT 'docker.io',       -- docker.io, ghcr.io, gcr.io, etc.
    username TEXT NOT NULL,
    encrypted_token BLOB,                    -- Encrypted with sealed key
    token_storage TEXT DEFAULT 'encrypted',  -- 'encrypted' or 'open'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_docker_profiles_name ON docker_profiles(name);
CREATE INDEX IF NOT EXISTS idx_docker_profiles_registry ON docker_profiles(registry);

-- Add key_rotation_days to config table
ALTER TABLE config ADD COLUMN key_rotation_days INTEGER DEFAULT 30;

-- Record this migration
INSERT INTO schema_migrations (version, description) VALUES (2, 'Sealed keys and docker profiles');
