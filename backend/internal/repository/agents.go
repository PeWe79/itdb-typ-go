package repository

import (
	"context"
	"database/sql"
)

const agentListQuery = `SELECT agents.*, TRIM(
        CASE WHEN (agents.type & 1) = 1 THEN '采购方 Buyer ' ELSE '' END ||
        CASE WHEN (agents.type & 2) = 2 THEN '软件厂商 S/W Manufacturer ' ELSE '' END ||
        CASE WHEN (agents.type & 8) = 8 THEN '硬件厂商 H/W Manufacturer ' ELSE '' END ||
        CASE WHEN (agents.type & 4) = 4 THEN '供应商 Vendor ' ELSE '' END ||
        CASE WHEN (agents.type & 16) = 16 THEN '承包方 Contractor ' ELSE '' END
    ) AS typetext FROM agents`

func (r *DomainRepository) ListAgents(ctx context.Context, search string) (*sql.Rows, error) {
	query := `SELECT * FROM (` + agentListQuery + `) AS agent_view`
	args := []interface{}{}
	if search != "" {
		query += ` WHERE (CAST(agent_view.id AS TEXT) LIKE ? OR agent_view.typetext LIKE ? OR agent_view.title LIKE ? OR agent_view.contactinfo LIKE ? OR agent_view.contacts LIKE ?)`
		q := "%" + search + "%"
		args = append(args, q, q, q, q, q)
	}
	query += ` ORDER BY agent_view.title, agent_view.type, agent_view.id`
	return r.QueryContext(ctx, query, args...)
}

func (r *DomainRepository) GetAgent(ctx context.Context, id int64) (*sql.Rows, error) {
	return r.QueryContext(ctx, `SELECT * FROM agents WHERE id = ?`, id)
}

func (r *DomainRepository) CreateAgent(ctx context.Context, typeMask int64, title, contactInfo, contacts, urls string) (sql.Result, error) {
	return r.ExecContext(ctx, `INSERT INTO agents (type, title, contactinfo, contacts, urls) VALUES (?, ?, ?, ?, ?)`, typeMask, title, contactInfo, contacts, urls)
}

func (r *DomainRepository) UpdateAgent(ctx context.Context, id, typeMask int64, title, contactInfo, contacts, urls string) (sql.Result, error) {
	return r.ExecContext(ctx, `UPDATE agents SET type = ?, title = ?, contactinfo = ?, contacts = ?, urls = ? WHERE id = ?`, typeMask, title, contactInfo, contacts, urls, id)
}

func (r *DomainRepository) DeleteAgentReferences(ctx context.Context, tx *sql.Tx, id int64) error {
	for _, query := range []string{
		`DELETE FROM agents WHERE id = ?`,
		`UPDATE items SET manufacturerid = '' WHERE manufacturerid = ?`,
		`UPDATE invoices SET vendorid = '' WHERE vendorid = ?`,
		`UPDATE invoices SET buyerid = '' WHERE buyerid = ?`,
		`UPDATE software SET manufacturerid = '' WHERE manufacturerid = ?`,
	} {
		if _, err := tx.ExecContext(ctx, query, id); err != nil {
			return err
		}
	}
	return nil
}
