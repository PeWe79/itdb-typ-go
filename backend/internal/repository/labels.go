package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

func (r *DomainRepository) ListLabelItems(ctx context.Context, search, orderExpr string, limit, offset int64) (*sql.Rows, error) {
	if orderExpr == "" {
		orderExpr = "itemtypes.typedesc"
	}
	q := `SELECT items.id,items.status,COALESCE(TRIM(statustypes.statusdesc),'') AS statusdesc,COALESCE(TRIM(statustypes.color),'') AS statuscolor,itemtypes.typedesc AS itemtype,agents.title AS manufacturer,items.model,items.sn,items.sn3,items.label FROM items JOIN itemtypes ON items.itemtypeid=itemtypes.id JOIN agents ON agents.id=items.manufacturerid LEFT JOIN statustypes ON statustypes.id=items.status`
	a := []interface{}{}
	if strings.TrimSpace(search) != "" {
		q += ` WHERE printf('%04d-%s',items.id,COALESCE(itemtypes.typedesc,'')) LIKE ? OR (CAST(items.id AS TEXT)||'-'||COALESCE(itemtypes.typedesc,'')) LIKE ? OR (COALESCE(agents.title,'-')||'-'||COALESCE(items.model,'-')||'-'||COALESCE(NULLIF(TRIM(items.sn),''),'-')||CASE WHEN TRIM(COALESCE(items.label,''))='' THEN '' ELSE '-'||TRIM(items.label) END) LIKE ?`
		x := "%" + strings.TrimSpace(search) + "%"
		a = []interface{}{x, x, x}
	}
	q += ` ORDER BY CASE COALESCE(TRIM(statustypes.statusdesc),'') WHEN '使用中' THEN 0 WHEN '库存' THEN 1 WHEN '有故障' THEN 2 WHEN '报废' THEN 3 ELSE 4 END,items.status,` + orderExpr + `,itemtypes.typedesc,items.manufacturerid,items.id,items.sn,items.sn2,items.sn3 LIMIT ? OFFSET ?`
	a = append(a, limit, offset)
	return r.QueryContext(ctx, q, a...)
}
func (r *DomainRepository) PreviewLabelItems(ctx context.Context, ids []int64) (*sql.Rows, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	p := make([]string, len(ids))
	a := make([]interface{}, len(ids))
	for i, id := range ids {
		p[i] = "?"
		a[i] = id
	}
	return r.QueryContext(ctx, `SELECT items.id,itemtypes.typedesc AS itemtype,agents.title AS manufacturer,items.model,items.sn,items.sn3,items.label,items.dnsname,items.ipv4,items.ipv6 FROM items JOIN itemtypes ON itemtypes.id=items.itemtypeid JOIN agents ON agents.id=items.manufacturerid WHERE items.id IN (`+strings.Join(p, ",")+") ORDER BY items.id", a...)
}

func (r *DomainRepository) ListLabelPresets(ctx context.Context) (*sql.Rows, error) {
	return r.QueryContext(ctx, `SELECT * FROM labelpapers ORDER BY id ASC`)
}
func (r *DomainRepository) CreateLabelPreset(ctx context.Context, args ...interface{}) (sql.Result, error) {
	return r.ExecContext(ctx, `INSERT INTO labelpapers (rows,cols,lwidth,lheight,vpitch,hpitch,tmargin,bmargin,lmargin,rmargin,name,border,padding,fontsize,headerfontsize,barcodesize,idfontsize,wantbarcode,wantheadertext,wantheaderimage,headertext,image,imagewidth,imageheight,papersize,qrtext,wantnotext,wantraligntext,labelskip) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, args...)
}
func (r *DomainRepository) FindLabelPresetIDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	e := r.QueryRowContext(ctx, `SELECT id FROM labelpapers WHERE TRIM(name)=? LIMIT 1`, name).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		return 0, nil
	}
	if e != nil {
		return 0, e
	}
	return id, nil
}
func (r *DomainRepository) UpdateLabelPreset(ctx context.Context, id int64, args ...interface{}) error {
	a := append(args, id)
	_, e := r.ExecContext(ctx, `UPDATE labelpapers SET rows=?,cols=?,lwidth=?,lheight=?,vpitch=?,hpitch=?,tmargin=?,bmargin=?,lmargin=?,rmargin=?,name=?,border=?,padding=?,fontsize=?,headerfontsize=?,barcodesize=?,idfontsize=?,wantbarcode=?,wantheadertext=?,wantheaderimage=?,headertext=?,image=?,imagewidth=?,imageheight=?,papersize=?,qrtext=?,wantnotext=?,wantraligntext=?,labelskip=? WHERE id=?`, a...)
	return e
}
func (r *DomainRepository) DeleteLabelPreset(ctx context.Context, id int64) (sql.Result, error) {
	return r.ExecContext(ctx, `DELETE FROM labelpapers WHERE id=?`, id)
}
