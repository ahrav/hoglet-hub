package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ahrav/hoglet-hub/internal/db"
	"github.com/ahrav/hoglet-hub/internal/domain/operation"
	"github.com/ahrav/hoglet-hub/internal/infra/storage"
)

// Package postgres provides PostgreSQL implementations of the domain repositories.
var _ operation.Repository = (*operationStore)(nil)

// operationStore implements operation.Repository using Postgres and sqlc-generated queries.
type operationStore struct {
	q      *db.Queries
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

// defaultDBAttributes defines standard OpenTelemetry attributes for database operations.
var defaultDBAttributes = []attribute.KeyValue{attribute.String("db.system", "postgresql")}

// NewOperationStore creates an operation.Repository backed by PostgreSQL.
// It provides persistence for operation entities and their lifecycle management.
func NewOperationStore(pool *pgxpool.Pool, tracer trace.Tracer) operation.Repository {
	return &operationStore{q: db.New(pool), pool: pool, tracer: tracer}
}

// Create persists a new operation and returns its ID.
// It handles serialization of operation parameters and sets default values where needed.
func (s *operationStore) Create(ctx context.Context, op *operation.Operation) (int64, error) {
	dbAttrs := append(defaultDBAttributes,
		attribute.String("operation.type", string(op.Type)),
		attribute.String("operation.status", string(op.Status)),
	)

	if op.TenantID != nil {
		dbAttrs = append(dbAttrs, attribute.Int64("tenant.id", *op.TenantID))
	}

	var id int64
	err := storage.ExecuteAndTrace(ctx, s.tracer, "operationStore.Create", dbAttrs, func(ctx context.Context) error {
		var tenantID pgtype.Int8
		if op.TenantID != nil {
			tenantID.Int64 = *op.TenantID
			tenantID.Valid = true
		}

		paramsJSON, err := json.Marshal(op.Parameters)
		if err != nil {
			return err
		}

		createdBy := "system@hoglet-hub.com"
		if op.CreatedBy != nil {
			createdBy = *op.CreatedBy
		}

		var createErr error
		id, createErr = s.q.CreateOperation(ctx, db.CreateOperationParams{
			TenantID:      tenantID,
			OperationType: string(op.Type),
			Status:        db.OperationStatus(op.Status),
			Parameters:    paramsJSON,
			CreatedBy:     createdBy,
		})
		return createErr
	})

	return id, err
}

// Update modifies an existing operation with new state information.
// This is used to track operation progress, results, and completion status.
func (s *operationStore) Update(ctx context.Context, op *operation.Operation) error {
	dbAttrs := append(defaultDBAttributes,
		attribute.Int64("operation.id", op.ID),
		attribute.String("operation.status", string(op.Status)),
	)

	if op.TenantID != nil {
		dbAttrs = append(dbAttrs, attribute.Int64("tenant.id", *op.TenantID))
	}

	return storage.ExecuteAndTrace(ctx, s.tracer, "operationStore.Update", dbAttrs, func(ctx context.Context) error {
		resultJSON, err := json.Marshal(op.Result)
		if err != nil {
			return err
		}

		var errorMsg pgtype.Text
		if op.ErrorMessage != nil {
			errorMsg.String = *op.ErrorMessage
			errorMsg.Valid = true
		}

		var startedAt, completedAt pgtype.Timestamptz
		if op.StartedAt != nil {
			startedAt.Time = *op.StartedAt
			startedAt.Valid = true
		}
		if op.CompletedAt != nil {
			completedAt.Time = *op.CompletedAt
			completedAt.Valid = true
		}

		return s.q.UpdateOperation(ctx, db.UpdateOperationParams{
			ID:           op.ID,
			Status:       db.OperationStatus(op.Status),
			Result:       resultJSON,
			ErrorMessage: errorMsg,
			StartedAt:    startedAt,
			CompletedAt:  completedAt,
		})
	})
}

// FindByID retrieves an operation by ID.
// Returns ErrOperationNotFound if the operation doesn't exist.
func (s *operationStore) FindByID(ctx context.Context, id int64) (*operation.Operation, error) {
	dbAttrs := append(defaultDBAttributes, attribute.Int64("operation.id", id))

	var dbOp db.Operation
	err := storage.ExecuteAndTrace(ctx, s.tracer, "operationStore.FindByID", dbAttrs, func(ctx context.Context) error {
		var err error
		dbOp, err = s.q.FindOperationByID(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return operation.ErrOperationNotFound
			}
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return mapDBOperationToDomain(dbOp)
}

// FindByTenantID retrieves all operations associated with a tenant.
// This allows tracking all operations for a specific tenant.
func (s *operationStore) FindByTenantID(ctx context.Context, tenantID int64) ([]*operation.Operation, error) {
	dbAttrs := append(defaultDBAttributes, attribute.Int64("tenant.id", tenantID))

	var dbOps []db.Operation
	err := storage.ExecuteAndTrace(ctx, s.tracer, "operationStore.FindByTenantID", dbAttrs, func(ctx context.Context) error {
		tenantIDPg := pgtype.Int8{Int64: tenantID, Valid: true}
		var err error
		dbOps, err = s.q.FindOperationsByTenantID(ctx, tenantIDPg)
		return err
	})

	if err != nil {
		return nil, err
	}

	return mapDBOperationsToDomain(dbOps)
}

// FindByStatus retrieves operations with a specific status.
// Useful for finding operations in particular states (pending, running, etc.).
func (s *operationStore) FindByStatus(ctx context.Context, status operation.Status) ([]*operation.Operation, error) {
	dbAttrs := append(defaultDBAttributes, attribute.String("operation.status", string(status)))

	var dbOps []db.Operation
	err := storage.ExecuteAndTrace(ctx, s.tracer, "operationStore.FindByStatus", dbAttrs, func(ctx context.Context) error {
		var err error
		dbOps, err = s.q.FindOperationsByStatus(ctx, db.OperationStatus(status))
		return err
	})

	if err != nil {
		return nil, err
	}

	return mapDBOperationsToDomain(dbOps)
}

// FindIncomplete retrieves all non-terminal operations.
// This is primarily used by background workers to find operations that need processing.
func (s *operationStore) FindIncomplete(ctx context.Context) ([]*operation.Operation, error) {
	dbAttrs := append(defaultDBAttributes, attribute.String("operation.filter", "incomplete"))

	var dbOps []db.Operation
	err := storage.ExecuteAndTrace(ctx, s.tracer, "operationStore.FindIncomplete", dbAttrs, func(ctx context.Context) error {
		var err error
		dbOps, err = s.q.FindIncompleteOperations(ctx)
		return err
	})

	if err != nil {
		return nil, err
	}

	return mapDBOperationsToDomain(dbOps)
}

// mapDBOperationToDomain converts a database operation record to a domain operation entity.
// It handles nullable fields and JSON deserialization of parameters and results.
func mapDBOperationToDomain(dbOp db.Operation) (*operation.Operation, error) {
	var tenantID *int64
	if dbOp.TenantID.Valid {
		val := dbOp.TenantID.Int64
		tenantID = &val
	}

	var startedAt *time.Time
	if dbOp.StartedAt.Valid {
		val := dbOp.StartedAt.Time
		startedAt = &val
	}

	var completedAt *time.Time
	if dbOp.CompletedAt.Valid {
		val := dbOp.CompletedAt.Time
		completedAt = &val
	}

	var updatedAt *time.Time
	if !dbOp.UpdatedAt.Time.Equal(dbOp.CreatedAt.Time) {
		val := dbOp.UpdatedAt.Time
		updatedAt = &val
	}

	var errorMessage *string
	if dbOp.ErrorMessage.Valid {
		val := dbOp.ErrorMessage.String
		errorMessage = &val
	}

	createdBy := "system"
	if dbOp.CreatedBy != "" {
		createdBy = dbOp.CreatedBy
	}
	createdByPtr := &createdBy

	params := map[string]any{}
	if len(dbOp.Parameters) > 0 {
		if err := json.Unmarshal(dbOp.Parameters, &params); err != nil {
			return nil, err
		}
	}

	result := map[string]any{}
	if len(dbOp.Result) > 0 {
		if err := json.Unmarshal(dbOp.Result, &result); err != nil {
			return nil, err
		}
	}

	opType, err := operation.ParseType(dbOp.OperationType)
	if err != nil {
		return nil, err
	}

	return &operation.Operation{
		ID:           dbOp.ID,
		Type:         opType,
		Status:       operation.Status(dbOp.Status),
		TenantID:     tenantID,
		CreatedAt:    dbOp.CreatedAt.Time,
		StartedAt:    startedAt,
		CompletedAt:  completedAt,
		UpdatedAt:    updatedAt,
		CreatedBy:    createdByPtr,
		ErrorMessage: errorMessage,
		Parameters:   params,
		Result:       result,
	}, nil
}

// mapDBOperationsToDomain converts multiple database operation records to domain entities.
func mapDBOperationsToDomain(dbOps []db.Operation) ([]*operation.Operation, error) {
	ops := make([]*operation.Operation, 0, len(dbOps))
	for _, dbOp := range dbOps {
		op, err := mapDBOperationToDomain(dbOp)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, nil
}
