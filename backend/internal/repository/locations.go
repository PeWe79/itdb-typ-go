package repository

import (
	"context"
	"database/sql"
)

func (r *DomainRepository) ListLocations(ctx context.Context, search string) (*sql.Rows, error) {
	q := `SELECT * FROM (SELECT locations.id,locations.name,locations.floor,locations.floorplanfn,COALESCE((SELECT GROUP_CONCAT(area_rows.areaname,', ') FROM (SELECT TRIM(COALESCE(locareas.areaname,'')) AS areaname FROM locareas WHERE locareas.locationid=locations.id ORDER BY locareas.id) AS area_rows WHERE area_rows.areaname<>''),'') AS areaname FROM locations) AS location_view`
	a := []interface{}{}
	if search != "" {
		q += ` WHERE CAST(location_view.id AS TEXT) LIKE ? OR location_view.name LIKE ? OR location_view.floor LIKE ? OR location_view.areaname LIKE ? OR location_view.floorplanfn LIKE ?`
		x := "%" + search + "%"
		for i := 0; i < 5; i++ {
			a = append(a, x)
		}
	}
	q += ` ORDER BY location_view.id DESC`
	return r.QueryContext(ctx, q, a...)
}
func (r *DomainRepository) GetLocation(ctx context.Context, id int64) (*sql.Rows, error) {
	return r.QueryContext(ctx, `SELECT * FROM locations WHERE id=?`, id)
}
func (r *DomainRepository) ListAreas(ctx context.Context, id int64) (*sql.Rows, error) {
	return r.QueryContext(ctx, `SELECT * FROM locareas WHERE locationid=? ORDER BY areaname`, id)
}

// LocationItemCount counts hardware referencing the location directly or via its areas.
func (r *DomainRepository) LocationItemCount(ctx context.Context, id int64) (int64, error) {
	var n int64
	e := r.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE locationid=? OR locareaid IN (SELECT id FROM locareas WHERE locationid=?)`, id, id).Scan(&n)
	return n, e
}

// AreaRackCount counts racks referencing the area; used to report rack-specific delete blocks.
func (r *DomainRepository) AreaRackCount(ctx context.Context, id int64) (int64, error) {
	var n int64
	e := r.QueryRowContext(ctx, `SELECT COUNT(*) FROM racks WHERE locareaid=?`, id).Scan(&n)
	return n, e
}

// AreaItemCount counts hardware items referencing the area; used for aggregated delete blocks.
func (r *DomainRepository) AreaItemCount(ctx context.Context, id int64) (int64, error) {
	var n int64
	e := r.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE locareaid=?`, id).Scan(&n)
	return n, e
}

// LocationRackCount counts racks referencing the location; used to block location deletion.
func (r *DomainRepository) LocationRackCount(ctx context.Context, id int64) (int64, error) {
	var n int64
	e := r.QueryRowContext(ctx, `SELECT COUNT(*) FROM racks WHERE locationid=?`, id).Scan(&n)
	return n, e
}

func (r *DomainRepository) CreateLocation(ctx context.Context, name, floor, floorplan string) (sql.Result, error) {
	return r.ExecContext(ctx, `INSERT INTO locations (name,floor,floorplanfn) VALUES (?,?,?)`, name, floor, floorplan)
}
func (r *DomainRepository) UpdateLocation(ctx context.Context, tx *sql.Tx, id int64, name, floor string, floorplan *string) (sql.Result, error) {
	if floorplan != nil {
		return tx.ExecContext(ctx, `UPDATE locations SET name=?,floor=?,floorplanfn=? WHERE id=?`, name, floor, *floorplan, id)
	}
	return tx.ExecContext(ctx, `UPDATE locations SET name=?,floor=? WHERE id=?`, name, floor, id)
}
func (r *DomainRepository) LocationFloorplan(ctx context.Context, id int64) (string, string, error) {
	var file, name sql.NullString
	e := r.QueryRowContext(ctx, `SELECT floorplanfn,name FROM locations WHERE id=?`, id).Scan(&file, &name)
	return file.String, name.String, e
}
func (r *DomainRepository) DeleteLocation(ctx context.Context, tx *sql.Tx, id int64) error {
	if _, e := tx.ExecContext(ctx, `DELETE FROM locations WHERE id=?`, id); e != nil {
		return e
	}
	_, e := tx.ExecContext(ctx, `UPDATE items SET locationid=0 WHERE locationid=?`, id)
	return e
}
