//go:build integration

// Package testutil provides shared test helpers for the backend modules.
// postgres.go contains the testcontainers-based Postgres harness.
// Build tag: integration — only compiled when -tags=integration is passed.
package testutil

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SetupPostgres starts a postgres:16-alpine container, applies the initial
// migration, seeds the role table, and returns a ready *gorm.DB.
// The returned teardown function MUST be called (defer teardown()) to
// terminate the container after the test.
func SetupPostgres(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("SetupPostgres: failed to start container: %v", err)
	}

	teardown := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("SetupPostgres: failed to terminate container: %v", err)
		}
	}

	// Build the DSN for both GORM and golang-migrate.
	// ConnectionString returns postgres://user:pass@host:port/db?sslmode=disable
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		teardown()
		t.Fatalf("SetupPostgres: failed to get connection string: %v", err)
	}

	// Open GORM connection.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		teardown()
		t.Fatalf("SetupPostgres: failed to open GORM connection: %v", err)
	}

	// Ensure pgcrypto extension is available (migration also does this, but
	// doing it here first makes the sequence idempotent).
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
		teardown()
		t.Fatalf("SetupPostgres: failed to create pgcrypto extension: %v", err)
	}

	// Resolve path to migrations directory relative to this file.
	// At runtime the test binary's cwd is the package dir, but using
	// runtime.Caller gives us the source file path which is always reliable.
	_, thisFile, _, _ := runtime.Caller(0)
	// postgres.go is at backend/internal/testutil/postgres.go
	// migrations are at backend/migrations/
	// relative path: ../../migrations (up 2 levels from testutil/ to backend/)
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
	migrationsDir = filepath.Clean(migrationsDir)
	migrationsURL := fmt.Sprintf("file://%s", migrationsDir)

	m, err := migrate.New(migrationsURL, dsn)
	if err != nil {
		teardown()
		t.Fatalf("SetupPostgres: failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		teardown()
		t.Fatalf("SetupPostgres: migration failed: %v", err)
	}

	// Seed the role table — the migration creates the table but does NOT seed rows.
	seedSQL := `INSERT INTO role (nombre)
		VALUES ('alumno'), ('creador'), ('supervisor'), ('administrador')
		ON CONFLICT DO NOTHING`
	if err := db.Exec(seedSQL).Error; err != nil {
		teardown()
		t.Fatalf("SetupPostgres: failed to seed roles: %v", err)
	}

	return db, teardown
}
