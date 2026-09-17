package repository

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestStoreDelegatesQueriesAndTransactions(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE sample (id INTEGER PRIMARY KEY, name TEXT); INSERT INTO sample (name) VALUES ('ok')`); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db)
	var name string
	if err := store.QueryRow("SELECT name FROM sample WHERE id = ?", 1).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "ok" {
		t.Fatalf("name = %q", name)
	}
	tx, err := store.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`UPDATE sample SET name = 'updated' WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
