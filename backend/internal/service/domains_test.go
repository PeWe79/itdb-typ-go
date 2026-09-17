package service

import (
	"database/sql"
	"testing"

	"itdb-backend/internal/repository"

	_ "modernc.org/sqlite"
)

func TestNewDomainServicesWiresEveryBusinessBoundary(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	services := NewDomainServices(repository.NewStore(db))
	boundaries := []RepositoryService{services.Auth, services.Items, services.Software, services.Invoices, services.Contracts, services.Files, services.Agents, services.Users, services.Infrastructure, services.Dictionaries, services.Tags, services.System, services.Tools}
	for i, boundary := range boundaries {
		if boundary == nil {
			t.Fatalf("domain service %d is nil", i)
		}
	}
}
