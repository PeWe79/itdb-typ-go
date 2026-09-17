package repository

import (
	"context"
	"database/sql"
	"time"
)

func (r *DomainRepository) InsertItem(ctx context.Context, tx *sql.Tx, args ...interface{}) (sql.Result, error) {
	return tx.ExecContext(ctx, `INSERT INTO items (
		label, itemtypeid, function, manufacturerid, warrinfo, model, sn, sn2, sn3, origin,
		warrantymonths, purchasedate, purchprice, dnsname, dptid, principal, locationid, locareaid, userid,
		maintenanceinfo, comments, ispart, rackid, rackposition, rackposdepth, rackmountable, usize, status,
		macs, ipv4, ipv6, remadmip, hd, cpu, cpuno, corespercpu, ram, raid, raidconfig, panelport,
		switchid, switchport, ports
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, args...)
}

func (r *DomainRepository) UpdateItem(ctx context.Context, tx *sql.Tx, id int64, args ...interface{}) (sql.Result, error) {
	args = append(args, id)
	return tx.ExecContext(ctx, `UPDATE items SET
		itemtypeid = ?, function = ?, manufacturerid = ?, label = ?, warrinfo = ?, model = ?, sn = ?, sn2 = ?, sn3 = ?,
		locationid = ?, locareaid = ?, origin = ?, warrantymonths = ?, purchasedate = ?, purchprice = ?, dnsname = ?,
		userid = ?, dptid = ?, principal = ?, comments = ?, maintenanceinfo = ?, ispart = ?, hd = ?, cpu = ?, cpuno = ?,
		corespercpu = ?, ram = ?, raid = ?, raidconfig = ?, rackmountable = ?, rackid = ?, rackposition = ?, rackposdepth = ?,
		usize = ?, status = ?, macs = ?, ipv4 = ?, ipv6 = ?, remadmip = ?, panelport = ?, switchid = ?, switchport = ?, ports = ?
		WHERE id = ?`, args...)
}

func (r *DomainRepository) ItemUserID(ctx context.Context, tx *sql.Tx, id int64) (sql.NullInt64, error) {
	var value sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT userid FROM items WHERE id = ?`, id).Scan(&value)
	return value, err
}

func (r *DomainRepository) ItemFileIDs(ctx context.Context, tx *sql.Tx, id int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT fileid FROM item2file WHERE itemid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []int64{}
	for rows.Next() {
		var value int64
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func (r *DomainRepository) DeleteItemReferences(ctx context.Context, tx *sql.Tx, id int64) error {
	for _, statement := range []struct {
		query string
		args  []interface{}
	}{
		{`DELETE FROM item2file WHERE itemid = ?`, []interface{}{id}},
		{`DELETE FROM item2inv WHERE itemid = ?`, []interface{}{id}},
		{`DELETE FROM item2soft WHERE itemid = ?`, []interface{}{id}},
		{`DELETE FROM itemlink WHERE itemid1 = ? OR itemid2 = ?`, []interface{}{id, id}},
		{`DELETE FROM tag2item WHERE itemid = ?`, []interface{}{id}},
		{`DELETE FROM actions WHERE itemid = ?`, []interface{}{id}},
		{`DELETE FROM items WHERE id = ?`, []interface{}{id}},
	} {
		if _, err := tx.ExecContext(ctx, statement.query, statement.args...); err != nil {
			return err
		}
	}
	return nil
}

// RecordItemAction 在事务外写入维护日志，用于保存失败等需要随失败结果留痕的场景
func (r *DomainRepository) RecordItemAction(ctx context.Context, itemID int64, description, result string) error {
	now := time.Now().Unix()
	_, err := r.ExecContext(ctx, `INSERT INTO actions (itemid, actiondate, description, invoiceinfo, isauto) VALUES (?, ?, ?, ?, 1)`,
		itemID, now, description, result)
	return err
}
