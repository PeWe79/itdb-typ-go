package service

import (
	"context"
	"database/sql"
	"fmt"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"strconv"
	"strings"
)

type LabelWorkflow struct{ repo repository.ToolRepository }

func NewLabelWorkflow(repo repository.ToolRepository) *LabelWorkflow {
	return &LabelWorkflow{repo: repo}
}
func (s *LabelWorkflow) Items(ctx context.Context, search, order string, limit, offset int64) (*sql.Rows, error) {
	return s.repo.ListLabelItems(ctx, strings.TrimSpace(search), domain.SafeLabelOrderExpr(order), limit, offset)
}
func (s *LabelWorkflow) Preview(ctx context.Context, req domain.LabelPreviewRequest) ([]map[string]interface{}, error) {
	rows, e := s.repo.PreviewLabelItems(ctx, req.ItemIDs)
	if e != nil || rows == nil {
		return []map[string]interface{}{}, e
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id int64
		var typ, man, model, sn, sn3, label, dns, ip4, ip6 string
		if e := rows.Scan(&id, &typ, &man, &model, &sn, &sn3, &label, &dns, &ip4, &ip6); e != nil {
			return nil, e
		}
		if strings.TrimSpace(sn) == "" {
			sn = sn3
		}
		out = append(out, map[string]interface{}{"id": id, "itemType": typ, "manufacturer": man, "model": model, "sn": sn, "label": label, "dnsName": dns, "ipv4": ip4, "ipv6": ip6, "text": fmt.Sprintf("%s %s %s %s", typ, man, model, sn), "headerText": req.HeaderText, "qrText": strings.TrimSpace(req.QRPrefix) + strconv.FormatInt(id, 10)})
	}
	return out, rows.Err()
}

var _ = primitives.AsString
