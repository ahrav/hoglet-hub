-- 0001_init_schemas.down.sql

-- =============================================================================
-- Down Migration: Drop Provisioning API Database Schema
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Drop Audit Logs (must be dropped first because of its dependencies)
-- -----------------------------------------------------------------------------
DROP TABLE IF EXISTS audit_logs;

-- -----------------------------------------------------------------------------
-- Drop Resource Counts
-- -----------------------------------------------------------------------------
DROP TABLE IF EXISTS resource_counts;

-- -----------------------------------------------------------------------------
-- Drop Resources (depends on tenants and operations)
-- -----------------------------------------------------------------------------
DROP TABLE IF EXISTS resources;

-- -----------------------------------------------------------------------------
-- Drop Operations (depends on tenants)
-- -----------------------------------------------------------------------------
DROP TABLE IF EXISTS operations;

-- -----------------------------------------------------------------------------
-- Drop Tenants (depends on isolation_groups and database_nodes)
-- -----------------------------------------------------------------------------
DROP TABLE IF EXISTS tenants;

-- -----------------------------------------------------------------------------
-- Drop Database Nodes
-- -----------------------------------------------------------------------------
DROP TABLE IF EXISTS database_nodes;

-- -----------------------------------------------------------------------------
-- Drop Isolation Groups
-- -----------------------------------------------------------------------------
DROP TABLE IF EXISTS isolation_groups;

-- -----------------------------------------------------------------------------
-- Drop ENUM types (drop in reverse order of dependency)
-- -----------------------------------------------------------------------------
DROP TYPE IF EXISTS audit_status;
DROP TYPE IF EXISTS resource_status;
DROP TYPE IF EXISTS operation_status;
DROP TYPE IF EXISTS tenant_status;
DROP TYPE IF EXISTS database_node_status;
DROP TYPE IF EXISTS node_type;
DROP TYPE IF EXISTS region_type;
