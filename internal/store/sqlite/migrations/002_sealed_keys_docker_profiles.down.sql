-- Migration: 002_sealed_keys_docker_profiles (rollback)
-- Description: Remove sealed keys and docker profiles tables

DROP INDEX IF EXISTS idx_docker_profiles_registry;
DROP INDEX IF EXISTS idx_docker_profiles_name;
DROP TABLE IF EXISTS docker_profiles;
DROP TABLE IF EXISTS sealed_keys;

-- Note: SQLite doesn't support DROP COLUMN in older versions
-- key_rotation_days column will remain but be unused

-- Remove migration record
DELETE FROM schema_migrations WHERE version = 2;
