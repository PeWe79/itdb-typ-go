package domain

import (
	"strconv"
	"strings"

	"itdb-backend/internal/common/primitives"
)

type BrowseNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Leaf     bool   `json:"leaf"`
	Resource string `json:"resource,omitempty"`
	EntityID int64  `json:"entityId,omitempty"`
}

func ToBrowseNodes(rows []map[string]interface{}, idPrefix string, leaf bool, resource string) []BrowseNode {
	nodes := make([]BrowseNode, 0, len(rows))
	for _, row := range rows {
		id := primitives.AsInt64(row["id"])
		label := strings.TrimSpace(primitives.AsString(row["nodetext"]))
		if label == "" {
			label = strings.TrimSpace(primitives.AsString(row["name"]))
		}
		if label == "" {
			label = strings.TrimSpace(primitives.AsString(row["typedesc"]))
		}
		if label == "" {
			label = strings.TrimSpace(primitives.AsString(row["username"]))
		}
		rowPrefix := strings.TrimSpace(primitives.AsString(row["nodeprefix"]))
		if rowPrefix == "" {
			rowPrefix = idPrefix
		}
		rowResource := strings.TrimSpace(primitives.AsString(row["resource"]))
		if rowResource == "" {
			rowResource = resource
		}
		nodes = append(nodes, BrowseNode{
			ID:       rowPrefix + strconv.FormatInt(id, 10),
			Label:    label,
			Leaf:     leaf,
			Resource: rowResource,
			EntityID: id,
		})
	}
	return nodes
}

func SafeLabelOrderExpr(raw string) string {
	switch strings.TrimSpace(raw) {
	case "id":
		return "items.id"
	case "id_desc":
		return "items.id DESC"
	case "model":
		return "items.model"
	case "status":
		return "items.status"
	default:
		return "itemtypes.typedesc"
	}
}
