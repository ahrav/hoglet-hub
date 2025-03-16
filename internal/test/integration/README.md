# Integration Tests

This package contains end-to-end integration tests for the Hoglet Hub application. These tests verify the application functions correctly with real infrastructure dependencies.

## What's Covered

- **Tenant Service Integration Tests**: Tests for the tenant creation and deletion workflows
- Full database interaction with real Postgres
- Asynchronous workflow execution and verification
- Database migrations for test setup

## Requirements

To run these tests, you need one of:

1. A running Postgres database (can be provided via Docker, see below)
2. Nothing! The tests can create their own isolated test containers (see "Using TestContainer" below)

## Running Tests

### Using TestContainer (easiest, no setup needed)

The simplest way to run the tests is to use the built-in TestContainer support:

```bash
# Run tests with automatic test container setup
make integration-test-direct
```

This approach:
1. Automatically creates a test container for each test suite
2. Runs migrations in the container
3. Executes tests against the containerized database
4. Cleans up all containers when done

This is the simplest approach, requiring no external dependencies or setup.

### Using Docker Setup (shared database)

If you prefer a shared test database for all tests:

```bash
# Start test Postgres, run migrations, and run all integration tests
make integration-test-docker

# Run a subset of tests (use -short flag)
make integration-test-short
```

This approach:
1. Starts a dedicated Postgres container on port 5433
2. Copies migration files to a temporary directory
3. Runs migrations using the migrate/migrate Docker image
4. Runs tests against the migrated database
5. Stops the container when done

### Using an Existing Database

If you want to use an existing database:

```bash
# Set environment variable for test database
export TEST_DATABASE_URL="postgres://username:password@localhost:5432/hoglet_test?sslmode=disable"

# Run the tests (migrations must be applied separately)
make integration-test

# Or directly with Go
go test -tags=integration ./internal/test/integration/... -v
```

## Test Organization

- `tenant_service_test.go`: Tests for tenant creation and deletion through the service layer
- `testutil/`: Helper functions and utilities for integration testing
  - `db.go`: Database connection utilities that leverage testcontainer.go
  - `workflow.go`: Utilities for testing async workflows
  - `assertions.go`: Test assertion helpers

## Test Configuration

You can configure how tests connect to the database in the code:

```go
// Use the existing database specified by TEST_DATABASE_URL (default)
opts := testutil.DefaultTestDatabaseOption()

// Or use an isolated test container for this test
opts.UseTestContainer = true

// Setup the database and get connection pool + cleanup function
pool, cleanup := testutil.SetupTestDatabase(t, opts)
defer cleanup() // Don't forget to call cleanup!
```

## Test Structure

Each integration test typically follows this pattern:

1. **Setup**: Initialize database connections and services
2. **Execution**: Perform the primary operation (e.g., creating a tenant)
3. **Verification**: Wait for async operations to complete and verify results
4. **Cleanup**: Delete resources to leave a clean state

## Adding New Tests

When adding new integration tests:

1. Create a new test file in this directory
2. Use the utilities in `testutil/` for common operations
3. Ensure tests clean up after themselves
4. Follow the existing patterns for waiting on async operations
5. Use unique resource names (with timestamps) to avoid conflicts in parallel runs

## Test Data

Tests create their own data with unique names to avoid conflicts. Each test is responsible for cleaning up its own data.

## Configuration

The tests can be configured using environment variables:

- `TEST_DATABASE_URL`: Connection string for test database (default: `postgres://postgres:postgres@localhost:5432/hoglet_test?sslmode=disable`)
