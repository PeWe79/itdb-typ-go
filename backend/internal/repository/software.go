package repository

import (
	"context"
	"database/sql"
)

const softwareListQuery = `SELECT
        software.id,
        software.manufacturerid,
        software.stitle AS title,
        software.sversion AS version,
        software.slicenseinfo,
        software.sinfo,
        software.purchdate,
        COALESCE((
            SELECT MAX(contracts.currentenddate)
            FROM contract2soft
            JOIN contracts ON contracts.id = contract2soft.contractid
            WHERE contract2soft.softid = software.id
        ), 0) AS maintend,
        software.licqty,
        software.lictype,
        agents.title AS manufacturer,
        COALESCE((
            SELECT CASE software.lictype
                WHEN 1 THEN SUM(COALESCE(items.cpuno, 0))
                WHEN 2 THEN SUM(COALESCE(items.cpuno, 0) * COALESCE(items.corespercpu, 0))
                ELSE COUNT(items.id)
            END
            FROM item2soft
            JOIN items ON items.id = item2soft.itemid
            WHERE item2soft.softid = software.id
        ), 0) AS usedqty,
        COALESCE((
            SELECT GROUP_CONCAT(vendors.title, ', ')
            FROM soft2inv
            JOIN invoices ON invoices.id = soft2inv.invid
            LEFT JOIN agents AS vendors ON vendors.id = invoices.vendorid
            WHERE soft2inv.softid = software.id
        ), '') AS vendor,
        COALESCE((
            SELECT GROUP_CONCAT(vendors.id, ',')
            FROM soft2inv
            JOIN invoices ON invoices.id = soft2inv.invid
            LEFT JOIN agents AS vendors ON vendors.id = invoices.vendorid
            WHERE soft2inv.softid = software.id
        ), '') AS vendorids,
        COALESCE((
            SELECT GROUP_CONCAT(CAST(invoices.id AS TEXT) || ':' || COALESCE(invoices.number, ''), ', ')
            FROM soft2inv
            JOIN invoices ON invoices.id = soft2inv.invid
            WHERE soft2inv.softid = software.id
        ), '') AS invoice,
        COALESCE((
            SELECT GROUP_CONCAT(inst.entry, ' | ')
            FROM (
                SELECT
                    '(' || items.id || ') ' ||
                    COALESCE(itemman.title, '') || ' ' ||
                    COALESCE(items.model, '') ||
                    CASE WHEN TRIM(COALESCE(items.dnsname, '')) = '' THEN '' ELSE ' ' || TRIM(items.dnsname) END AS entry
                FROM item2soft
                JOIN items ON items.id = item2soft.itemid
                LEFT JOIN agents AS itemman ON itemman.id = items.manufacturerid
                WHERE item2soft.softid = software.id
                ORDER BY items.id
            ) AS inst
        ), '') AS installedon,
        COALESCE((SELECT GROUP_CONCAT(tags.name, ', ') FROM tags JOIN tag2software ON tags.id = tag2software.tagid WHERE tag2software.softwareid = software.id), '') AS tags,
        COALESCE((SELECT GROUP_CONCAT(tags.id, ',') FROM tags JOIN tag2software ON tags.id = tag2software.tagid WHERE tag2software.softwareid = software.id), '') AS tagids
    FROM software
    JOIN agents ON agents.id = software.manufacturerid`

func (r *DomainRepository) ListSoftware(ctx context.Context, search string, limit, offset int64) (*sql.Rows, error) {
	q := `SELECT * FROM (` + softwareListQuery + `) AS software_view`
	args := []interface{}{}
	if search != "" {
		q += ` WHERE CAST(software_view.id AS TEXT) LIKE ? OR software_view.manufacturer LIKE ? OR software_view.title LIKE ? OR software_view.version LIKE ? OR software_view.slicenseinfo LIKE ? OR software_view.sinfo LIKE ? OR software_view.tags LIKE ? OR software_view.vendor LIKE ? OR software_view.invoice LIKE ? OR software_view.installedon LIKE ?`
		x := "%" + search + "%"
		for i := 0; i < 10; i++ {
			args = append(args, x)
		}
	}
	q += ` ORDER BY software_view.id DESC`
	if limit >= 0 {
		q += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	return r.QueryContext(ctx, q, args...)
}
func (r *DomainRepository) GetSoftware(ctx context.Context, id int64) (*sql.Rows, error) {
	return r.QueryContext(ctx, `SELECT * FROM software WHERE id = ?`, id)
}

func (r *DomainRepository) InsertSoftware(ctx context.Context, tx *sql.Tx, args ...interface{}) (sql.Result, error) {
	return tx.ExecContext(ctx, `INSERT INTO software (invoiceid, slicenseinfo, manufacturerid, stitle, sversion, sinfo, purchdate, licqty, lictype) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, args...)
}
func (r *DomainRepository) UpdateSoftware(ctx context.Context, tx *sql.Tx, id int64, args ...interface{}) (sql.Result, error) {
	args = append(args, id)
	return tx.ExecContext(ctx, `UPDATE software SET invoiceid = ?, slicenseinfo = ?, manufacturerid = ?, stitle = ?, sversion = ?, sinfo = ?, purchdate = ?, licqty = ?, lictype = ? WHERE id = ?`, args...)
}
func (r *DomainRepository) SoftwareFileIDs(ctx context.Context, tx *sql.Tx, id int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT fileid FROM software2file WHERE softwareid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func (r *DomainRepository) DeleteSoftwareReferences(ctx context.Context, tx *sql.Tx, id int64) error {
	for _, q := range []string{`DELETE FROM software2file WHERE softwareid = ?`, `DELETE FROM tag2software WHERE softwareid = ?`, `DELETE FROM software WHERE id = ?`, `DELETE FROM item2soft WHERE softid = ?`} {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return err
		}
	}
	return nil
}
