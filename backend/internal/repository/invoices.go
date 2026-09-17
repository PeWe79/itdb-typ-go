package repository

import (
	"context"
	"database/sql"
)

const invoiceListQuery = `SELECT
        invoices.id,
        invoices.vendorid,
        invoices.buyerid,
        invoices.number,
        invoices.date,
        invoices.description,
        vendor.title AS vendor,
        buyer.title AS buyer,
        COALESCE((
            SELECT GROUP_CONCAT(
                '(' || CAST(files.id AS TEXT) || ') ' ||
                CASE
                    WHEN TRIM(COALESCE(files.title, '')) = '' THEN COALESCE(files.fname, '')
                    WHEN TRIM(COALESCE(files.fname, '')) = '' THEN COALESCE(files.title, '')
                    ELSE COALESCE(files.title, '') || ' / ' || COALESCE(files.fname, '')
                END,
                ' | '
            )
            FROM files
            JOIN invoice2file ON files.id = invoice2file.fileid
            WHERE invoice2file.invoiceid = invoices.id
            ORDER BY files.id
        ), '') AS files
    FROM invoices
    LEFT JOIN agents AS vendor ON vendor.id = invoices.vendorid
    LEFT JOIN agents AS buyer ON buyer.id = invoices.buyerid`

func (r *DomainRepository) ListInvoices(ctx context.Context, search string, limit, offset int64) (*sql.Rows, error) {
	q := `SELECT * FROM (` + invoiceListQuery + `) AS invoice_view`
	a := []interface{}{}
	if search != "" {
		q += ` WHERE CAST(invoice_view.id AS TEXT) LIKE ? OR invoice_view.vendor LIKE ? OR invoice_view.buyer LIKE ? OR (CASE WHEN COALESCE(invoice_view.date,0)>0 THEN date(invoice_view.date,'unixepoch') ELSE '' END) LIKE ? OR invoice_view.number LIKE ? OR invoice_view.description LIKE ? OR invoice_view.files LIKE ?`
		x := "%" + search + "%"
		for i := 0; i < 7; i++ {
			a = append(a, x)
		}
	}
	q += ` ORDER BY invoice_view.id DESC`
	if limit >= 0 {
		q += ` LIMIT ? OFFSET ?`
		a = append(a, limit, offset)
	}
	return r.QueryContext(ctx, q, a...)
}
func (r *DomainRepository) GetInvoice(ctx context.Context, id int64) (*sql.Rows, error) {
	return r.QueryContext(ctx, `SELECT * FROM invoices WHERE id = ?`, id)
}
func (r *DomainRepository) InsertInvoice(ctx context.Context, tx *sql.Tx, args ...interface{}) (sql.Result, error) {
	return tx.ExecContext(ctx, `INSERT INTO invoices (vendorid,buyerid,number,description,date) VALUES (?,?,?,?,?)`, args...)
}
func (r *DomainRepository) UpdateInvoice(ctx context.Context, tx *sql.Tx, id int64, args ...interface{}) (sql.Result, error) {
	args = append(args, id)
	return tx.ExecContext(ctx, `UPDATE invoices SET vendorid=?,buyerid=?,number=?,description=?,date=? WHERE id=?`, args...)
}
func (r *DomainRepository) InvoiceFileIDs(ctx context.Context, tx *sql.Tx, id int64) ([]int64, error) {
	rows, e := tx.QueryContext(ctx, `SELECT fileid FROM invoice2file WHERE invoiceid=?`, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var v int64
		if e := rows.Scan(&v); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *DomainRepository) DeleteInvoiceReferences(ctx context.Context, tx *sql.Tx, id int64) error {
	for _, q := range []string{`DELETE FROM invoice2file WHERE invoiceid=?`, `DELETE FROM invoices WHERE id=?`, `DELETE FROM item2inv WHERE invid=?`, `UPDATE software SET invoiceid='' WHERE invoiceid=?`} {
		if _, e := tx.ExecContext(ctx, q, id); e != nil {
			return e
		}
	}
	return nil
}
