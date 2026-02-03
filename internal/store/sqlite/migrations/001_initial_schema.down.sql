-- Migration: 001_initial_schema (rollback)
-- Description: Remove initial database schema

DROP TABLE IF EXISTS registered_clients;
DROP TABLE IF EXISTS pending_registrations;
DROP TABLE IF EXISTS synced_data;
DROP TABLE IF EXISTS server_encryption_config;
DROP TABLE IF EXISTS standalone_connections;
DROP TABLE IF EXISTS standalone_clients;
DROP TABLE IF EXISTS standalone_config;
DROP TABLE IF EXISTS workspaces;
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS config;
DROP TABLE IF EXISTS repositories;
DROP TABLE IF EXISTS schema_migrations;
