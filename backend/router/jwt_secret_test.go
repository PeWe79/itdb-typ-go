package router

import (
	"database/sql"
	"itdb-backend/router/settings"
	"testing"

	_ "modernc.org/sqlite"
)

func newJWTSecretTestDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := settings.EnsureResourceSchema(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestResolveJWTSecretPersistsPerDatabase(t *testing.T) {
	db := newJWTSecretTestDB(t, "jwt-secret-stable")
	first, err := resolveJWTSecret(db, "")
	if err != nil {
		t.Fatal(err)
	}
	if first == "" {
		t.Fatal("empty generated secret")
	}
	second, err := resolveJWTSecret(db, "")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("secret changed across restarts: %s vs %s", first, second)
	}
}

func TestResolveJWTSecretRotatesAfterDatabaseReset(t *testing.T) {
	oldDB := newJWTSecretTestDB(t, "jwt-secret-old")
	oldSecret, err := resolveJWTSecret(oldDB, "")
	if err != nil {
		t.Fatal(err)
	}
	newDB := newJWTSecretTestDB(t, "jwt-secret-new")
	newSecret, err := resolveJWTSecret(newDB, "")
	if err != nil {
		t.Fatal(err)
	}
	if oldSecret == newSecret {
		t.Fatalf("secret reused after database reset")
	}
}

func TestResolveJWTSecretRespectsConfiguredValue(t *testing.T) {
	db := newJWTSecretTestDB(t, "jwt-secret-configured")
	secret, err := resolveJWTSecret(db, "explicit-secret")
	if err != nil {
		t.Fatal(err)
	}
	if secret != "explicit-secret" {
		t.Fatalf("configured secret not respected: %s", secret)
	}
	var stored string
	if err := db.QueryRow("SELECT jwt_secret FROM system_secrets WHERE id=1").Scan(&stored); err != sql.ErrNoRows {
		t.Fatalf("configured secret should not be persisted, got %q", stored)
	}
}
