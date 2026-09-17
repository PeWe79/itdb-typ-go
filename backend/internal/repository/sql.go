package repository

import (
	"context"

	"itdb-backend/pkg/database"
)

// SQL provides the shared read-side database operations used by API handlers
// during the compatibility migration to domain repositories.
type SQL struct{ db Executor }

func NewSQL(db Executor) *SQL { return &SQL{db: db} }

func (q *SQL) QueryRows(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return database.RowsToMaps(rows)
}
