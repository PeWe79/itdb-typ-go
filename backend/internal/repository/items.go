package repository

import (
	"context"
	"database/sql"
)

const itemListQuery = `SELECT items.id, items.label, items.itemtypeid, items.manufacturerid, items.locationid, items.locareaid, items.rackid, items.userid,
 itemtypes.typedesc AS itemType, agents.title AS manufacturer, items.model, items.sn, items.sn2, items.sn3, items.purchasedate, items.warrantymonths,
 items.ipv4, items.dnsname, items.principal, users.username, statustypes.statusdesc AS status, statustypes.color AS statuscolor, dpttypes.dptname AS dpt,
 CASE
   WHEN TRIM(COALESCE(locations.name,'')) = '' THEN '-'
   WHEN TRIM(COALESCE(locareas.areaname,'')) = '' THEN locations.name
   ELSE locations.name || '-' || locareas.areaname
 END AS location,
 COALESCE(locareas.areaname,'') AS area,
 CASE WHEN TRIM(COALESCE(racks.label,'') || COALESCE(items.switchport,'')) = '' THEN '-' ELSE TRIM(COALESCE(racks.label,'') || ' ' || COALESCE(items.switchport,'')) END AS rack,
 items.function, items.remadmip, items.usize, items.rackposition, items.rackposdepth,
 COALESCE((SELECT GROUP_CONCAT(tags.name, ', ') FROM tags JOIN tag2item ON tags.id = tag2item.tagid WHERE tag2item.itemid = items.id), '') AS tags,
        COALESCE((SELECT GROUP_CONCAT(tags.id, ',') FROM tags JOIN tag2item ON tags.id = tag2item.tagid WHERE tag2item.itemid = items.id), '') AS tagids,
 COALESCE((SELECT GROUP_CONCAT(sw.id || '#' || TRIM(COALESCE(swman.title, '') || ' ' || COALESCE(sw.stitle, '') || ' ' || COALESCE(sw.sversion, '')), ', ') FROM software AS sw JOIN item2soft AS i2s ON sw.id = i2s.softid LEFT JOIN agents AS swman ON swman.id = sw.manufacturerid WHERE i2s.itemid = items.id), '') AS software
 FROM items JOIN itemtypes ON itemtypes.id = items.itemtypeid JOIN agents ON agents.id = items.manufacturerid
 LEFT JOIN statustypes ON statustypes.id = items.status LEFT JOIN dpttypes ON dpttypes.id = items.dptid LEFT JOIN users ON users.id = items.userid
 LEFT JOIN locations ON locations.id = items.locationid LEFT JOIN locareas ON locareas.id = items.locareaid LEFT JOIN racks ON racks.id = items.rackid`

func (r *DomainRepository) ListItems(ctx context.Context, search string, limit, offset int64) (*sql.Rows, error) {
	query := `SELECT * FROM (` + itemListQuery + `) AS item_view`
	args := []interface{}{}
	if search != "" {
		query += ` WHERE CAST(item_view.id AS TEXT) LIKE ? OR item_view.label LIKE ? OR item_view.itemType LIKE ? OR item_view.manufacturer LIKE ? OR item_view.model LIKE ? OR item_view.sn LIKE ? OR (CASE WHEN COALESCE(item_view.purchasedate, 0) > 0 THEN date(item_view.purchasedate, 'unixepoch') ELSE '' END) LIKE ? OR item_view.ipv4 LIKE ? OR item_view.dpt LIKE ? OR item_view.principal LIKE ? OR item_view.status LIKE ? OR item_view.location LIKE ? OR item_view.rack LIKE ? OR item_view.function LIKE ? OR item_view.remadmip LIKE ? OR item_view.tags LIKE ? OR item_view.software LIKE ?`
		q := "%" + search + "%"
		for i := 0; i < 17; i++ {
			args = append(args, q)
		}
	}
	query += ` ORDER BY item_view.id DESC`
	if limit >= 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	return r.QueryContext(ctx, query, args...)
}

func (r *DomainRepository) GetItem(ctx context.Context, id int64) (*sql.Rows, error) {
	return r.QueryContext(ctx, `SELECT * FROM items WHERE id = ?`, id)
}
