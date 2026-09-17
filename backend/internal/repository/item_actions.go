package repository

import (
	"context"
	"database/sql"
)

func (r *DomainRepository) ListItemActions(ctx context.Context, itemID int64) (*sql.Rows, error) {
	return r.QueryContext(ctx, `SELECT * FROM actions WHERE itemid = ? ORDER BY actiondate, id`, itemID)
}
