package repository

import (
	"context"
	"database/sql"
)

func (r *DomainRepository) ListContracts(ctx context.Context, search string) (*sql.Rows, error) {
	q := `SELECT contracts.id,contracts.parentid,contracts.contractorid,contractor.title AS contractor,contracttypes.name AS type,contracts.number,contracts.title,contracts.startdate,contracts.currentenddate FROM contracts JOIN contracttypes ON contracttypes.id=contracts.type LEFT JOIN agents AS contractor ON contractor.id=contracts.contractorid`
	a := []interface{}{}
	if search != "" {
		q += ` WHERE CAST(contracts.id AS TEXT) LIKE ? OR CAST(COALESCE(contracts.parentid,'') AS TEXT) LIKE ? OR contracttypes.name LIKE ? OR contractor.title LIKE ? OR contracts.number LIKE ? OR contracts.title LIKE ? OR (CASE WHEN COALESCE(contracts.startdate,0)>0 THEN date(contracts.startdate,'unixepoch') ELSE '' END) LIKE ? OR (CASE WHEN COALESCE(contracts.currentenddate,0)>0 THEN date(contracts.currentenddate,'unixepoch') ELSE '' END) LIKE ?`
		x := "%" + search + "%"
		for i := 0; i < 8; i++ {
			a = append(a, x)
		}
	}
	q += ` ORDER BY contracts.id DESC,contracts.parentid DESC`
	return r.QueryContext(ctx, q, a...)
}
func (r *DomainRepository) GetContract(ctx context.Context, id int64) (*sql.Rows, error) {
	return r.QueryContext(ctx, `SELECT * FROM contracts WHERE id=?`, id)
}
func (r *DomainRepository) InsertContract(ctx context.Context, tx *sql.Tx, args ...interface{}) (sql.Result, error) {
	return tx.ExecContext(ctx, `INSERT INTO contracts (type,subtype,parentid,title,number,description,comments,totalcost,contractorid,startdate,currentenddate,renewals) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, args...)
}
func (r *DomainRepository) UpdateContract(ctx context.Context, tx *sql.Tx, id int64, args ...interface{}) (sql.Result, error) {
	args = append(args, id)
	return tx.ExecContext(ctx, `UPDATE contracts SET type=?,subtype=?,parentid=?,title=?,number=?,description=?,comments=?,totalcost=?,contractorid=?,startdate=?,currentenddate=?,renewals=? WHERE id=?`, args...)
}
func (r *DomainRepository) ContractFileIDs(ctx context.Context, tx *sql.Tx, id int64) ([]int64, error) {
	rows, e := tx.QueryContext(ctx, `SELECT fileid FROM contract2file WHERE contractid=?`, id)
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
func (r *DomainRepository) DeleteContractReferences(ctx context.Context, tx *sql.Tx, id int64) error {
	for _, q := range []string{`DELETE FROM contract2item WHERE contractid=?`, `DELETE FROM contract2soft WHERE contractid=?`, `DELETE FROM contract2inv WHERE contractid=?`, `DELETE FROM contract2file WHERE contractid=?`, `DELETE FROM contracts WHERE id=?`} {
		if _, e := tx.ExecContext(ctx, q, id); e != nil {
			return e
		}
	}
	return nil
}

func (r *DomainRepository) ListContractEvents(ctx context.Context, id int64) (*sql.Rows, error) {
	return r.QueryContext(ctx, `SELECT * FROM contractevents WHERE contractid=? ORDER BY startdate,id DESC`, id)
}
func (r *DomainRepository) CreateContractEvent(ctx context.Context, contractID, siblingID, startDate, endDate int64, description string) (sql.Result, error) {
	return r.ExecContext(ctx, `INSERT INTO contractevents (contractid,siblingid,startdate,enddate,description) VALUES (?,?,?,?,?)`, contractID, siblingID, startDate, endDate, description)
}
func (r *DomainRepository) UpdateContractEvent(ctx context.Context, eventID, siblingID, startDate, endDate int64, description string) (sql.Result, error) {
	return r.ExecContext(ctx, `UPDATE contractevents SET siblingid=?,startdate=?,enddate=?,description=? WHERE id=?`, siblingID, startDate, endDate, description, eventID)
}
func (r *DomainRepository) DeleteContractEvent(ctx context.Context, eventID int64) (sql.Result, error) {
	return r.ExecContext(ctx, `DELETE FROM contractevents WHERE id=?`, eventID)
}
