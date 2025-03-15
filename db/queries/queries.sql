-- Tenant Queries

-- name: CreateTenant :one
INSERT INTO tenants (
    name,
    region,
    status,
    tier,
    is_isolated,
    isolation_group_id,
    created_by
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: UpdateTenant :exec
UPDATE tenants
SET
    status = $2,
    tier = $3,
    is_isolated = $4,
    isolation_group_id = $5,
    database_schema = $6,
    kubernetes_namespace = $7,
    primary_node_id = $8,
    updated_at = NOW()
WHERE id = $1;

-- name: FindTenantByID :one
SELECT * FROM tenants
WHERE id = $1 AND status != 'deleting'
LIMIT 1;

-- name: FindTenantByName :one
SELECT * FROM tenants
WHERE name = $1 AND status != 'deleting'
LIMIT 1;

-- name: DeleteTenant :exec
UPDATE tenants
SET
    status = 'deleting',
    updated_at = NOW()
WHERE id = $1;

-- Operation Queries

-- name: CreateOperation :one
INSERT INTO operations (
    tenant_id,
    operation_type,
    status,
    parameters,
    created_by
) VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: UpdateOperation :exec
UPDATE operations
SET
    status = $2,
    result = $3,
    error_message = $4,
    started_at = $5,
    completed_at = $6,
    updated_at = NOW()
WHERE id = $1;

-- name: FindOperationByID :one
SELECT * FROM operations
WHERE id = $1
LIMIT 1;

-- name: FindOperationsByTenantID :many
SELECT * FROM operations
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: FindOperationsByStatus :many
SELECT * FROM operations
WHERE status = $1
ORDER BY created_at ASC;

-- name: FindIncompleteOperations :many
SELECT * FROM operations
WHERE status IN ('pending', 'in_progress')
ORDER BY created_at ASC;
