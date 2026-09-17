package service

import (
	"context"
	"database/sql"
	"sort"

	"itdb-backend/internal/repository"
	"itdb-backend/pkg/database"
)

type DetailRelationsWorkflow struct{ repo repository.Repository }

func NewDetailRelationsWorkflow(repo repository.Repository) *DetailRelationsWorkflow {
	return &DetailRelationsWorkflow{repo: repo}
}
func (s *DetailRelationsWorkflow) Load(ctx context.Context, resource string, id int64) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	switch resource {
	case "items":
		forward, e := s.ids(ctx, `SELECT itemid2 FROM itemlink WHERE itemid1=?`, id)
		if e != nil {
			return nil, e
		}
		reverse, e := s.ids(ctx, `SELECT itemid1 FROM itemlink WHERE itemid2=?`, id)
		if e != nil {
			return nil, e
		}
		set := map[int64]struct{}{}
		for _, v := range append(forward, reverse...) {
			if v > 0 {
				set[v] = struct{}{}
			}
		}
		links := make([]int64, 0, len(set))
		for v := range set {
			links = append(links, v)
		}
		sort.Slice(links, func(i, j int) bool { return links[i] < links[j] })
		out["itemLinks"] = links
		out["invoiceLinks"], e = s.ids(ctx, `SELECT invid FROM item2inv WHERE itemid=?`, id)
		if e != nil {
			return nil, e
		}
		out["softwareLinks"], e = s.ids(ctx, `SELECT softid FROM item2soft WHERE itemid=?`, id)
		if e != nil {
			return nil, e
		}
		out["contractLinks"], e = s.ids(ctx, `SELECT contractid FROM contract2item WHERE itemid=?`, id)
		if e != nil {
			return nil, e
		}
		out["fileLinks"], e = s.ids(ctx, `SELECT fileid FROM item2file WHERE itemid=?`, id)
		if e != nil {
			return nil, e
		}
		out["tags"], e = s.strings(ctx, `SELECT tags.name FROM tags JOIN tag2item ON tag2item.tagid=tags.id WHERE tag2item.itemid=? ORDER BY tags.name`, id)
		if e != nil {
			return nil, e
		}
		rows, e := s.repo.QueryContext(ctx, `SELECT * FROM actions WHERE itemid=? ORDER BY actiondate`, id)
		if e != nil {
			return nil, e
		}
		defer rows.Close()
		out["actions"], e = database.RowsToMaps(rows)
		return out, e
	case "software":
		return s.loadSimple(ctx, id, `SELECT itemid FROM item2soft WHERE softid=?`, `SELECT invid FROM soft2inv WHERE softid=?`, `SELECT contractid FROM contract2soft WHERE softid=?`, `SELECT fileid FROM software2file WHERE softwareid=?`, `SELECT tags.name FROM tags JOIN tag2software ON tag2software.tagid=tags.id WHERE tag2software.softwareid=? ORDER BY tags.name`, []string{"itemLinks", "invoiceLinks", "contractLinks", "fileLinks", "tags"})
	case "invoices":
		return s.loadSimple(ctx, id, `SELECT itemid FROM item2inv WHERE invid=?`, `SELECT softid FROM soft2inv WHERE invid=?`, `SELECT contractid FROM contract2inv WHERE invid=?`, `SELECT fileid FROM invoice2file WHERE invoiceid=?`, "", []string{"itemLinks", "softwareLinks", "contractLinks", "fileLinks"})
	case "contracts":
		return s.loadSimple(ctx, id, `SELECT itemid FROM contract2item WHERE contractid=?`, `SELECT softid FROM contract2soft WHERE contractid=?`, `SELECT invid FROM contract2inv WHERE contractid=?`, `SELECT fileid FROM contract2file WHERE contractid=?`, "", []string{"itemLinks", "softwareLinks", "invoiceLinks", "fileLinks"})
	case "files":
		return s.loadSimple(ctx, id, `SELECT itemid FROM item2file WHERE fileid=?`, `SELECT softwareid FROM software2file WHERE fileid=?`, `SELECT contractid FROM contract2file WHERE fileid=?`, `SELECT invoiceid FROM invoice2file WHERE fileid=?`, "", []string{"itemLinks", "softwareLinks", "contractLinks", "invoiceLinks"})
	}
	return out, nil
}
func (s *DetailRelationsWorkflow) loadSimple(ctx context.Context, id int64, q1, q2, q3, q4, q5 string, keys []string) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	for i, q := range []string{q1, q2, q3, q4} {
		v, e := s.ids(ctx, q, id)
		if e != nil {
			return nil, e
		}
		out[keys[i]] = v
	}
	if q5 != "" {
		v, e := s.strings(ctx, q5, id)
		if e != nil {
			return nil, e
		}
		out[keys[4]] = v
	}
	return out, nil
}
func (s *DetailRelationsWorkflow) ids(ctx context.Context, q string, id int64) ([]int64, error) {
	rows, e := s.repo.QueryContext(ctx, q, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var v sql.NullInt64
		if e := rows.Scan(&v); e != nil {
			return nil, e
		}
		if v.Valid {
			out = append(out, v.Int64)
		}
	}
	return out, rows.Err()
}
func (s *DetailRelationsWorkflow) strings(ctx context.Context, q string, id int64) ([]string, error) {
	rows, e := s.repo.QueryContext(ctx, q, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v sql.NullString
		if e := rows.Scan(&v); e != nil {
			return nil, e
		}
		out = append(out, v.String)
	}
	return out, rows.Err()
}
