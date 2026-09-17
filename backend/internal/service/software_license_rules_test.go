package service

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEnforceSoftwareLicenseCountRequiresLicensedWhenLinking(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, cpuno INTEGER, coresperscpu INTEGER)`); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	if err := enforceSoftwareLicenseCount(context.Background(), tx, nil, 0, []int64{5}); err == nil {
		t.Fatalf("linking without licensed qty should fail")
	}
	if err := enforceSoftwareLicenseCount(context.Background(), tx, nil, 0, nil); err != nil {
		t.Fatalf("saving without links and without licensed qty should pass, got %v", err)
	}
	zero := int64(0)
	if err := enforceSoftwareLicenseCount(context.Background(), tx, &zero, 0, []int64{5}); err == nil {
		t.Fatalf("zero licensed qty with links should fail with license limit message")
	}
}
