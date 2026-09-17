package service

import (
	"context"
	"database/sql"
	"testing"

	"itdb-backend/internal/repository"

	_ "modernc.org/sqlite"
)

func TestDetailRelationsWorkflowLoadsBidirectionalAndFileRelations(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{
		`CREATE TABLE itemlink (itemid1 INTEGER, itemid2 INTEGER)`,
		`CREATE TABLE item2inv (itemid INTEGER, invid INTEGER)`,
		`CREATE TABLE item2soft (itemid INTEGER, softid INTEGER)`,
		`CREATE TABLE contract2item (itemid INTEGER, contractid INTEGER)`,
		`CREATE TABLE item2file (itemid INTEGER, fileid INTEGER)`,
		`CREATE TABLE tags (id INTEGER, name TEXT)`,
		`CREATE TABLE tag2item (tagid INTEGER, itemid INTEGER)`,
		`CREATE TABLE actions (id INTEGER, itemid INTEGER, description TEXT, actiondate INTEGER)`,
		`CREATE TABLE software2file (softwareid INTEGER, fileid INTEGER)`,
		`CREATE TABLE contract2file (contractid INTEGER, fileid INTEGER)`,
		`CREATE TABLE invoice2file (invoiceid INTEGER, fileid INTEGER)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO itemlink VALUES (1,2),(3,1),(1,2); INSERT INTO item2inv VALUES (1,10); INSERT INTO item2soft VALUES (1,20); INSERT INTO contract2item VALUES (1,30); INSERT INTO item2file VALUES (1,40); INSERT INTO tags VALUES (5,'prod'); INSERT INTO tag2item VALUES (5,1); INSERT INTO actions VALUES (7,1,'created',1); INSERT INTO software2file VALUES (20,40);`); err != nil {
		t.Fatal(err)
	}
	workflow := NewDetailRelationsWorkflow(repository.NewStore(db))
	relations, err := workflow.Load(context.Background(), "items", 1)
	if err != nil {
		t.Fatal(err)
	}
	links := relations["itemLinks"].([]int64)
	if len(links) != 2 || links[0] != 2 || links[1] != 3 {
		t.Fatalf("links=%v", links)
	}
	if len(relations["tags"].([]string)) != 1 || len(relations["actions"].([]map[string]interface{})) != 1 {
		t.Fatalf("relations=%#v", relations)
	}
	fileRelations, err := workflow.Load(context.Background(), "files", 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(fileRelations["itemLinks"].([]int64)) != 1 || len(fileRelations["softwareLinks"].([]int64)) != 1 {
		t.Fatalf("file relations=%#v", fileRelations)
	}
}
