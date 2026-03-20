package migrate

import (
	"testing"
)

func TestToMigrateURL(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		want    string
		wantErr bool
	}{
		// TC-HAPPY-MIGRATE-001: plain DSN gets mysql:// prefix
		{
			"plain DSN",
			"root:pass@tcp(localhost:3306)/mydb?parseTime=true",
			"mysql://root:pass@tcp(localhost:3306)/mydb?parseTime=true&multiStatements=true",
			false,
		},
		// TC-HAPPY-MIGRATE-002: already has mysql:// prefix
		{
			"with scheme",
			"mysql://root:pass@tcp(localhost:3306)/mydb",
			"mysql://root:pass@tcp(localhost:3306)/mydb?multiStatements=true",
			false,
		},
		// TC-HAPPY-MIGRATE-003: already has multiStatements
		{
			"already has multiStatements",
			"mysql://root:pass@tcp(localhost:3306)/mydb?multiStatements=true",
			"mysql://root:pass@tcp(localhost:3306)/mydb?multiStatements=true",
			false,
		},
		// TC-EXCEPTION-MIGRATE-001: empty DSN
		{
			"empty DSN",
			"",
			"",
			true,
		},
		// TC-BOUNDARY-MIGRATE-001: DSN without query params
		{
			"no query params",
			"root:pass@tcp(localhost:3306)/mydb",
			"mysql://root:pass@tcp(localhost:3306)/mydb?multiStatements=true",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToMigrateURL(tt.dsn)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ToMigrateURL(%q) error = %v, wantErr %v", tt.dsn, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ToMigrateURL(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}
