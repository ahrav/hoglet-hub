package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ahrav/hoglet-hub/internal/db"
	"github.com/ahrav/hoglet-hub/internal/domain/tenant"
	"github.com/ahrav/hoglet-hub/internal/infra/storage"
)

var _ tenant.Repository = (*tenantStore)(nil)

// Package postgres provides PostgreSQL implementations of the domain repositories.
// It handles the persistence and retrieval of tenant data using the pgx driver.
type tenantStore struct {
	q      *db.Queries
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

// NewTenantStore creates a tenant.Repository backed by PostgreSQL.
// It provides CRUD operations for tenant entities with proper tracing.
func NewTenantStore(pool *pgxpool.Pool, tracer trace.Tracer) tenant.Repository {
	return &tenantStore{q: db.New(pool), pool: pool, tracer: tracer}
}

// defaultDBAttributes defines standard OpenTelemetry attributes for database operations.
var defaultDBAttributes = []attribute.KeyValue{attribute.String("db.system", "postgresql")}

// Create persists a new tenant and returns its ID.
// It handles the conversion between domain and database models,
// setting appropriate default values where needed.
func (s *tenantStore) Create(ctx context.Context, t *tenant.Tenant) (int64, error) {
	dbAttrs := append(defaultDBAttributes,
		attribute.String("tenant.name", t.Name),
		attribute.String("tenant.region", string(t.Region)),
		attribute.String("tenant.tier", string(t.Tier)),
	)

	var id int64
	err := storage.ExecuteAndTrace(ctx, s.tracer, "postgres.create_job", dbAttrs, func(ctx context.Context) error {
		var isolationGroupID pgtype.Int8
		if t.IsolationGroupID != nil {
			isolationGroupID.Int64 = *t.IsolationGroupID
			isolationGroupID.Valid = true
		}

		// TODO: This should never be the system, at least not for now.
		createdBy := "system" // Default creator when not explicitly provided

		isIsolated := pgtype.Bool{
			Bool:  t.IsolationGroupID != nil,
			Valid: true,
		}

		var err error
		id, err = s.q.CreateTenant(ctx, db.CreateTenantParams{
			Name:             t.Name,
			Region:           db.RegionType(t.Region),
			Status:           db.TenantStatus(t.Status),
			Tier:             string(t.Tier),
			IsIsolated:       isIsolated,
			IsolationGroupID: isolationGroupID,
			CreatedBy:        createdBy,
		})
		if err != nil {
			return err
		}

		return nil
	})

	return id, err
}

// Update modifies an existing tenant with new information.
// It handles nullable fields appropriately to avoid unintended overwrites.
func (s *tenantStore) Update(ctx context.Context, t *tenant.Tenant) error {
	dbAttrs := append(defaultDBAttributes,
		attribute.Int64("tenant.id", t.ID),
		attribute.String("tenant.status", string(t.Status)),
		attribute.String("tenant.tier", string(t.Tier)),
	)

	return storage.ExecuteAndTrace(ctx, s.tracer, "tenantStore.Update", dbAttrs, func(ctx context.Context) error {
		var isolationGroupID pgtype.Int8
		if t.IsolationGroupID != nil {
			isolationGroupID.Int64 = *t.IsolationGroupID
			isolationGroupID.Valid = true
		}

		// These fields are intentionally left as NULL since they're managed separately
		var dbSchema, k8sNamespace pgtype.Text
		var primaryNodeID pgtype.Int8

		isIsolated := pgtype.Bool{
			Bool:  t.IsolationGroupID != nil,
			Valid: true,
		}

		return s.q.UpdateTenant(ctx, db.UpdateTenantParams{
			ID:                  t.ID,
			Status:              db.TenantStatus(t.Status),
			Tier:                string(t.Tier),
			IsIsolated:          isIsolated,
			IsolationGroupID:    isolationGroupID,
			DatabaseSchema:      dbSchema,
			KubernetesNamespace: k8sNamespace,
			PrimaryNodeID:       primaryNodeID,
		})
	})
}

// FindByName retrieves a tenant by name.
// Returns ErrTenantNotFound if the tenant doesn't exist.
func (s *tenantStore) FindByName(ctx context.Context, name string) (*tenant.Tenant, error) {
	dbAttrs := append(defaultDBAttributes, attribute.String("tenant.name", name))

	var dbTenant db.Tenant
	err := storage.ExecuteAndTrace(ctx, s.tracer, "tenantStore.FindByName", dbAttrs, func(ctx context.Context) error {
		var err error
		dbTenant, err = s.q.FindTenantByName(ctx, name)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return tenant.ErrTenantNotFound
			}
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return mapDBTenantToDomain(dbTenant), nil
}

// FindByID retrieves a tenant by ID.
// Returns ErrTenantNotFound if the tenant doesn't exist.
func (s *tenantStore) FindByID(ctx context.Context, id int64) (*tenant.Tenant, error) {
	dbAttrs := append(defaultDBAttributes,
		attribute.Int64("tenant.id", id),
	)

	var dbTenant db.Tenant
	err := storage.ExecuteAndTrace(ctx, s.tracer, "tenantStore.FindByID", dbAttrs, func(ctx context.Context) error {
		var err error
		dbTenant, err = s.q.FindTenantByID(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return tenant.ErrTenantNotFound
			}
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return mapDBTenantToDomain(dbTenant), nil
}

// Delete marks a tenant for deletion.
// This is a soft delete that changes the tenant's status rather than removing the record.
func (s *tenantStore) Delete(ctx context.Context, id int64) error {
	dbAttrs := append(defaultDBAttributes, attribute.Int64("tenant.id", id))

	return storage.ExecuteAndTrace(ctx, s.tracer, "tenantStore.Delete", dbAttrs, func(ctx context.Context) error {
		return s.q.DeleteTenant(ctx, id)
	})
}

// mapDBTenantToDomain converts a database tenant record to a domain tenant entity.
// It handles nullable fields and time conversions appropriately.
func mapDBTenantToDomain(dbTenant db.Tenant) *tenant.Tenant {
	var isolationGroupID *int64
	if dbTenant.IsolationGroupID.Valid {
		val := dbTenant.IsolationGroupID.Int64
		isolationGroupID = &val
	}

	var updatedAt *time.Time
	if !dbTenant.UpdatedAt.Time.Equal(dbTenant.CreatedAt.Time) {
		val := dbTenant.UpdatedAt.Time
		updatedAt = &val
	}

	return &tenant.Tenant{
		ID:               dbTenant.ID,
		Name:             dbTenant.Name,
		Region:           tenant.Region(dbTenant.Region),
		Tier:             tenant.Tier(dbTenant.Tier),
		Status:           tenant.Status(dbTenant.Status),
		IsolationGroupID: isolationGroupID,
		CreatedAt:        dbTenant.CreatedAt.Time,
		UpdatedAt:        updatedAt,
	}
}
