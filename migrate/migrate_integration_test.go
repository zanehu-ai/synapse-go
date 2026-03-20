package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

const testDSN = "root:root@tcp(localhost:3306)/test_818_shared?parseTime=true&charset=utf8mb4"

// TC-HAPPY-MIGRATE-INT-001: Run with empty migrations directory succeeds (no change)
func TestRun_NoMigrations(t *testing.T) {
	// Create temp dir with no migration files — Run should return an error
	dir := t.TempDir()
	log, _ := zap.NewDevelopment()

	err := Run(testDSN, "file://"+dir, 0, log)
	if err == nil {
		t.Error("expected error for empty migrations directory")
	}
}

// TC-HAPPY-MIGRATE-INT-002: Run with a real migration
func TestRun_WithMigration(t *testing.T) {
	dir := t.TempDir()
	log, _ := zap.NewDevelopment()

	// Create a simple migration
	up := filepath.Join(dir, "000001_test.up.sql")
	down := filepath.Join(dir, "000001_test.down.sql")
	if err := os.WriteFile(up, []byte("CREATE TABLE IF NOT EXISTS _test_818_shared (id INT PRIMARY KEY);"), 0644); err != nil {
		t.Fatalf("write up migration: %v", err)
	}
	if err := os.WriteFile(down, []byte("DROP TABLE IF EXISTS _test_818_shared;"), 0644); err != nil {
		t.Fatalf("write down migration: %v", err)
	}

	err := Run(testDSN, "file://"+dir, 0, log)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// Run again — should succeed with no change
	err = Run(testDSN, "file://"+dir, 0, log)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	// Cleanup
	t.Cleanup(func() {
		dbURL, _ := ToMigrateURL(testDSN)
		m, err := migrate.New("file://"+dir, dbURL)
		if err == nil {
			m.Down()
			m.Close()
		}
	})
}

// TC-EXCEPTION-MIGRATE-INT-001: Run with invalid DSN
func TestRun_InvalidDSN(t *testing.T) {
	log, _ := zap.NewDevelopment()
	err := Run("invalid:invalid@tcp(localhost:9999)/nope", "file:///tmp", 0, log)
	if err == nil {
		t.Error("expected error for invalid DSN")
	}
}

// TC-EXCEPTION-MIGRATE-INT-002: Run with empty DSN
func TestRun_EmptyDSN(t *testing.T) {
	log, _ := zap.NewDevelopment()
	err := Run("", "file:///tmp", 0, log)
	if err == nil {
		t.Error("expected error for empty DSN")
	}
}
