package service

import (
	"context"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"itdb-backend/pkg/database"
)

type BrowseWorkflow struct{ repo repository.ToolRepository }

func NewBrowseWorkflow(repo repository.ToolRepository) *BrowseWorkflow {
	return &BrowseWorkflow{repo: repo}
}
func (s *BrowseWorkflow) List(ctx context.Context, id string) ([]domain.BrowseNode, error) {
	if id == "" || id == "0" {
		return []domain.BrowseNode{{ID: "itemtypes", Label: "硬件类型", Leaf: false}, {ID: "showusers", Label: "用户", Leaf: false}, {ID: "showagents", Label: "代理", Leaf: false}}, nil
	}
	if id == "showagents" {
		return []domain.BrowseNode{{ID: "agents:items", Label: "硬件厂商"}, {ID: "agents:software", Label: "软件厂商"}, {ID: "agents:vendors", Label: "供应商"}, {ID: "agents:buyers", Label: "采购方"}, {ID: "agents:contractors", Label: "承包方"}}, nil
	}
	rows, prefix, leaf, e := s.repo.BrowseRows(ctx, id)
	if e != nil {
		return nil, e
	}
	if rows == nil {
		return []domain.BrowseNode{}, nil
	}
	defer rows.Close()
	maps, e := database.RowsToMaps(rows)
	if e != nil {
		return nil, e
	}
	return domain.ToBrowseNodes(maps, prefix, leaf, resourceForPrefix(prefix)), nil
}
func resourceForPrefix(prefix string) string {
	switch prefix {
	case "useritem:", "typeitem:", "agenthwitem:":
		return "items"
	case "agentswsoftware:":
		return "software"
	case "vendorinvoice:", "buyerinvoice:":
		return "invoices"
	case "contractorcontract:":
		return "contracts"
	default:
		return ""
	}
}
