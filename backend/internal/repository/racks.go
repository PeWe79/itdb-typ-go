package repository

import (
	"context"
	"database/sql"
)

func (r *DomainRepository) ListRacks(ctx context.Context, search string) (*sql.Rows, error) {
	q := `SELECT * FROM (SELECT racks.id,racks.label,racks.usize,racks.depth,racks.locationid,racks.locareaid,racks.model,racks.comments,racks.revnums,locations.name AS location,locareas.areaname AS area,COUNT(items.id) AS population,COALESCE(SUM(items.usize),0) AS occupation FROM racks LEFT JOIN items ON items.rackid=racks.id LEFT JOIN locations ON locations.id=racks.locationid LEFT JOIN locareas ON locareas.id=racks.locareaid GROUP BY racks.id) AS rack_view`
	a := []interface{}{}
	if search != "" {
		q += ` WHERE CAST(rack_view.id AS TEXT) LIKE ? OR CAST(rack_view.occupation AS TEXT) LIKE ? OR CAST(rack_view.population AS TEXT) LIKE ? OR CAST(rack_view.usize AS TEXT) LIKE ? OR CAST(rack_view.depth AS TEXT) LIKE ? OR rack_view.location LIKE ? OR rack_view.area LIKE ? OR rack_view.label LIKE ?`
		x := "%" + search + "%"
		for i := 0; i < 8; i++ {
			a = append(a, x)
		}
	}
	q += ` ORDER BY rack_view.id DESC`
	return r.QueryContext(ctx, q, a...)
}
func (r *DomainRepository) GetRack(ctx context.Context, id int64) (*sql.Rows, error) {
	return r.QueryContext(ctx, `SELECT * FROM racks WHERE id=?`, id)
}
func (r *DomainRepository) InsertRack(ctx context.Context, args ...interface{}) (sql.Result, error) {
	return r.ExecContext(ctx, `INSERT INTO racks (locationid,usize,depth,comments,model,label,revnums,locareaid) VALUES (?,?,?,?,?,?,?,?)`, args...)
}
func (r *DomainRepository) UpdateRack(ctx context.Context, tx *sql.Tx, id int64, args ...interface{}) (sql.Result, error) {
	args = append(args, id)
	return tx.ExecContext(ctx, `UPDATE racks SET locationid=?,usize=?,depth=?,comments=?,model=?,label=?,revnums=?,locareaid=? WHERE id=?`, args...)
}
func (r *DomainRepository) RackItemCount(ctx context.Context, id int64) (int64, error) {
	var n int64
	e := r.QueryRowContext(ctx, `SELECT COUNT(id) FROM items WHERE rackid=?`, id).Scan(&n)
	return n, e
}
func (r *DomainRepository) UpdateRackItems(ctx context.Context, tx *sql.Tx, id, location int64, area interface{}) error {
	_, e := tx.ExecContext(ctx, `UPDATE items SET locationid=?,locareaid=? WHERE rackid=?`, location, area, id)
	return e
}
func (r *DomainRepository) DeleteRack(ctx context.Context, tx *sql.Tx, id int64) error {
	if _, e := tx.ExecContext(ctx, `UPDATE items SET rackid='' WHERE rackid=?`, id); e != nil {
		return e
	}
	_, e := tx.ExecContext(ctx, `DELETE FROM racks WHERE id=?`, id)
	return e
}
