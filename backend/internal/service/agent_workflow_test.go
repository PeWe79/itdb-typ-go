package service

import (
	"context"
	"database/sql"
	"testing"

	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"

	_ "modernc.org/sqlite"
)

func TestAgentWorkflowCreateUpdateAndEncoding(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, query := range []string{
		`CREATE TABLE agents (id INTEGER PRIMARY KEY AUTOINCREMENT, type INTEGER, title TEXT, contactinfo TEXT, contacts TEXT, urls TEXT)`,
		`CREATE TABLE items (id INTEGER PRIMARY KEY, manufacturerid INTEGER)`,
		`CREATE TABLE software (id INTEGER PRIMARY KEY, manufacturerid INTEGER)`,
		`CREATE TABLE invoices (id INTEGER PRIMARY KEY, vendorid INTEGER, buyerid INTEGER)`,
		`CREATE TABLE contracts (id INTEGER PRIMARY KEY, contractorid INTEGER)`,
		auditHistoryTestDDL,
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	store := repository.NewStore(db)
	workflow := NewAgentWorkflow(repository.NewAgentRepository(store), NewAuditService(store, 100))
	id, err := workflow.Create(context.Background(), UserActor{Username: "admin", IP: "127.0.0.1"}, domain.AgentPayload{
		Types: []int64{1, 8}, Title: " Acme ", ContactInfo: "primary",
		Contacts: []domain.AgentContact{{Name: "张三|采购", Phones: "123", Email: "a@b", Role: "销售", Comments: "备注#"}},
		URLs:     []domain.AgentURL{{Description: "官网", URL: "https://example.test"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var mask int64
	var title, contacts, urls string
	if err := db.QueryRow(`SELECT type, title, contacts, urls FROM agents WHERE id = ?`, id).Scan(&mask, &title, &contacts, &urls); err != nil {
		t.Fatal(err)
	}
	if mask != 9 || title != " Acme " || contacts != "张三 采购#123#a@b#销售#备注" || urls != "官网#https://example.test" {
		t.Fatalf("unexpected agent data: %d %q %q %q", mask, title, contacts, urls)
	}
	if err := workflow.Update(context.Background(), UserActor{Username: "admin"}, id, domain.AgentPayload{Type: ptrInt64(4), Title: "Acme 2"}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT type, title FROM agents WHERE id = ?`, id).Scan(&mask, &title); err != nil {
		t.Fatal(err)
	}
	if mask != 4 || title != "Acme 2" {
		t.Fatalf("update failed: %d %q", mask, title)
	}
}

func ptrInt64(value int64) *int64 { return &value }
