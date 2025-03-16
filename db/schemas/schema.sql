-- =============================================================================
-- Provisioning API Database Schema
--
-- This schema supports a multi-tenant provisioning system with:
-- - Tenant lifecycle management
-- - Resource tracking across GCP services
-- - Database node management for Citus
-- - Tenant isolation capabilities
-- - Operation logging and audit trail
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Isolation Groups and Database Nodes
-- -----------------------------------------------------------------------------

-- Define region types across our deployment regions
CREATE TYPE region_type AS ENUM ('us1', 'us2', 'us3', 'us4', 'eu1', 'eu2', 'eu3', 'eu4');

-- Isolation groups table - For grouping tenants that should be isolated together
CREATE TABLE isolation_groups (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,              -- Human-readable group name
    region region_type NOT NULL,                   -- Region this group is deployed in

    citus_colocation_id INTEGER,                   -- Link to Citus colocation ID

    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(64) NOT NULL                -- User who created this group
);

-- Define database node types and statuses
CREATE TYPE node_type AS ENUM ('standard', 'high-memory', 'isolated', 'coordinator');
CREATE TYPE database_node_status AS ENUM ('active', 'draining', 'maintenance', 'offline', 'provisioning', 'decommissioned');

-- Database nodes table - Tracks all database nodes in our Citus cluster
CREATE TABLE database_nodes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    hostname VARCHAR(128) NOT NULL UNIQUE,         -- Full hostname of the node
    port INTEGER NOT NULL DEFAULT 5432,            -- PostgreSQL port
    region region_type NOT NULL,                   -- Region where node is deployed

    -- Categorization and status
    node_type node_type NOT NULL DEFAULT 'standard',           -- Type of node (standard, high-memory, etc.)
    status database_node_status NOT NULL DEFAULT 'active',     -- Current operational status

    -- Capacity metrics
    tenant_count INTEGER DEFAULT 0,                -- Current number of tenants on this node
    max_tenants INTEGER DEFAULT 10000,             -- Maximum recommended tenants (Citus limit)
    current_utilization_percent INTEGER DEFAULT 0, -- Calculated utilization percentage

    -- Citus-specific metadata
    citus_metadata JSONB DEFAULT '{}'::jsonb,      -- Flexible storage for Citus metadata

    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(64) NOT NULL                -- User who created this node
);

CREATE INDEX idx_database_nodes_region ON database_nodes(region);
CREATE INDEX idx_database_nodes_status ON database_nodes(status);
CREATE INDEX idx_database_nodes_utilization ON database_nodes(current_utilization_percent); -- For load balancing queries

-- -----------------------------------------------------------------------------
-- Tenants
-- -----------------------------------------------------------------------------

-- Define tenant status enum
CREATE TYPE tenant_status AS ENUM ('provisioning', 'active', 'suspended', 'error', 'deleting', 'deleted', 'isolated');

-- Tenants table - Core table tracking all provisioned tenant instances
CREATE TABLE tenants (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(64) NOT NULL UNIQUE,              -- Unique tenant identifier/name
    region region_type NOT NULL,                   -- Deployment region
    status tenant_status NOT NULL DEFAULT 'provisioning', -- Current lifecycle status
    tier VARCHAR(16) NOT NULL DEFAULT 'free',      -- Subscription tier (free, pro, enterprise, etc.)

    -- Database details
    database_schema VARCHAR(64) UNIQUE,            -- Tenant's database schema name
    is_isolated BOOLEAN DEFAULT FALSE,             -- Quick flag for isolated tenants

    -- GCP/K8s details
    gke_cluster_name VARCHAR(64),                  -- GKE cluster hosting this tenant
    kubernetes_namespace VARCHAR(64) UNIQUE,       -- K8s namespace for this tenant

    -- Relationships
    isolation_group_id BIGINT REFERENCES isolation_groups(id) ON DELETE SET NULL, -- Optional isolation group
    primary_node_id BIGINT REFERENCES database_nodes(id) ON DELETE SET NULL,      -- Primary DB node

    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(64) NOT NULL                -- User who created this tenant
);

CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_region ON tenants(region);
CREATE INDEX idx_tenants_isolation_group ON tenants(isolation_group_id);
CREATE INDEX idx_tenants_primary_node ON tenants(primary_node_id);

-- -----------------------------------------------------------------------------
-- Operations
-- -----------------------------------------------------------------------------

-- Define operation status enum
CREATE TYPE operation_status AS ENUM ('pending', 'in_progress', 'completed', 'failed', 'cancelled');

-- Operations table - Tracks all async operations on tenants and resources
CREATE TABLE operations (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id BIGINT REFERENCES tenants(id) ON DELETE SET NULL, -- Associated tenant (if any)

    operation_type VARCHAR(32) NOT NULL,           -- Type of operation (create, delete, isolate, etc.)
    status operation_status NOT NULL DEFAULT 'pending', -- Current status of operation

    -- Parameters and results
    parameters JSONB DEFAULT '{}'::jsonb,          -- Input parameters for the operation
    result JSONB DEFAULT '{}'::jsonb,              -- Result data from completed operation
    error_message TEXT,                            -- Error details if operation failed

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,                        -- When operation execution began
    completed_at TIMESTAMPTZ,                      -- When operation finished (success or failure)

    -- Attribution
    created_by VARCHAR(64) NOT NULL                -- User who initiated this operation
);

CREATE INDEX idx_operations_tenant ON operations(tenant_id);
CREATE INDEX idx_operations_status ON operations(status);
CREATE INDEX idx_operations_type ON operations(operation_type);

-- -----------------------------------------------------------------------------
-- Resources
-- -----------------------------------------------------------------------------

-- Define resource status enum
CREATE TYPE resource_status AS ENUM ('provisioning', 'active', 'error', 'deleting', 'suspended');

-- Resources table - Tracks all GCP resources provisioned for tenants
CREATE TABLE resources (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE, -- Owning tenant

    -- Resource identification
    resource_type VARCHAR(32) NOT NULL,            -- Type of resource (gke_deployment, pubsub_topic, etc.)
    resource_name VARCHAR(128) NOT NULL,           -- Name of the resource
    resource_id VARCHAR(128),                      -- External ID if applicable

    -- Location information
    region region_type NOT NULL,                   -- Region where resource is deployed
    project_id VARCHAR(64) NOT NULL,               -- GCP project ID

    -- Status and metadata
    status resource_status NOT NULL DEFAULT 'provisioning', -- Current status
    metadata JSONB DEFAULT '{}'::jsonb,            -- Flexible storage for resource-specific attributes

    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Link to creating operation (for attribution)
    created_by_operation_id BIGINT REFERENCES operations(id) ON DELETE SET NULL
);

CREATE INDEX idx_resources_tenant ON resources(tenant_id);
CREATE INDEX idx_resources_type_name ON resources(resource_type, resource_name);
CREATE INDEX idx_resources_project ON resources(project_id);
CREATE INDEX idx_resources_type_count ON resources(resource_type, tenant_id); -- For counting resources by type per tenant

-- Resource counts table - Aggregated counts of resources by type and project
-- Helps enforce GCP quotas like 10,000 Pub/Sub topics per project
CREATE TABLE resource_counts (
    resource_type VARCHAR(32) NOT NULL,            -- Type of resource (pubsub_topic, etc.)
    project_id VARCHAR(64) NOT NULL,               -- GCP project ID
    region region_type NOT NULL,                   -- Region for this count
    count INTEGER NOT NULL DEFAULT 0,              -- Current count of resources
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Last time count was updated
    PRIMARY KEY (resource_type, project_id, region)
);

-- -----------------------------------------------------------------------------
-- Audit Logs
-- -----------------------------------------------------------------------------

-- Define audit status enum
CREATE TYPE audit_status AS ENUM ('success', 'failure');

-- Audit logs table - Comprehensive audit trail of all actions
CREATE TABLE audit_logs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    -- What happened
    action VARCHAR(64) NOT NULL,                   -- Action performed (create_tenant, etc.)
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- When the action occurred
    status audit_status NOT NULL,                  -- Outcome status

    -- Who did it
    actor VARCHAR(64) NOT NULL,                    -- User who performed the action
    actor_ip INET,                                 -- IP address of actor

    -- What it affected
    tenant_id BIGINT REFERENCES tenants(id) ON DELETE SET NULL,       -- Affected tenant (if any)
    resource_id BIGINT REFERENCES resources(id) ON DELETE SET NULL,   -- Affected resource (if any)
    operation_id BIGINT REFERENCES operations(id) ON DELETE SET NULL, -- Related operation (if any)

    -- Details
    details JSONB NOT NULL DEFAULT '{}'::jsonb,    -- Action-specific details
    error_details TEXT,                            -- Error information if action failed

    -- Performance tracking
    duration_ms INTEGER                            -- How long the action took
);

-- Consider time-based partitioning for this table in production
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor);
