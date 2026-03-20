package migrate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

// Run executes all pending up migrations.
// If forceVersion > 0, it forces the schema_migrations table to that version
// first (useful for bootstrapping an existing database).
func Run(dsn, migrationsPath string, forceVersion int, log *zap.Logger) error {
	dbURL, err := ToMigrateURL(dsn)
	if err != nil {
		return fmt.Errorf("build migrate URL: %w", err)
	}

	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer m.Close()

	if forceVersion > 0 {
		log.Info("forcing migration version", zap.Int("version", forceVersion))
		if err := m.Force(forceVersion); err != nil {
			return fmt.Errorf("force version %d: %w", forceVersion, err)
		}
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	ver, dirty, _ := m.Version()
	log.Info("migrations applied", zap.Uint("version", ver), zap.Bool("dirty", dirty))
	return nil
}

// ToMigrateURL converts a GORM-style MySQL DSN to the URL format required by
// golang-migrate: mysql://user:pass@tcp(host:port)/dbname?params&multiStatements=true
func ToMigrateURL(dsn string) (string, error) {
	if dsn == "" {
		return "", errors.New("empty DSN")
	}

	// Already has scheme — return as-is with multiStatements
	if strings.HasPrefix(dsn, "mysql://") {
		return ensureMultiStatements(dsn), nil
	}

	return ensureMultiStatements("mysql://" + dsn), nil
}

func ensureMultiStatements(url string) string {
	if strings.Contains(url, "multiStatements=true") {
		return url
	}
	if strings.Contains(url, "?") {
		return url + "&multiStatements=true"
	}
	return url + "?multiStatements=true"
}
