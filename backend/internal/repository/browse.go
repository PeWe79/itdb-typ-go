package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func (r *DomainRepository) BrowseRows(ctx context.Context, id string) (*sql.Rows, string, bool, error) {
	id = strings.TrimSpace(id)
	queries := map[string]struct{ query, prefix string }{
		"showusers":          {`SELECT id, username AS nodetext FROM users ORDER BY username`, "users:"},
		"itemtypes":          {`SELECT id, typedesc AS nodetext FROM itemtypes ORDER BY typedesc`, "itemtypes:"},
		"agents:items":       {`SELECT id, title AS nodetext FROM agents WHERE type & 8 ORDER BY nodetext`, "agenthw:"},
		"agents:software":    {`SELECT id, title AS nodetext FROM agents WHERE type & 2 ORDER BY nodetext`, "agentsw:"},
		"agents:vendors":     {`SELECT id, title AS nodetext FROM agents WHERE type & 4 ORDER BY nodetext`, "agentvendor:"},
		"agents:buyers":      {`SELECT id, title AS nodetext FROM agents WHERE type & 1 ORDER BY nodetext`, "agentbuyer:"},
		"agents:contractors": {`SELECT id, title AS nodetext FROM agents WHERE type & 16 ORDER BY nodetext`, "agentcontractor:"},
	}
	if item, ok := queries[id]; ok {
		rows, err := r.QueryContext(ctx, item.query)
		return rows, item.prefix, false, err
	}
	if strings.HasPrefix(id, "users:") {
		return r.browseByID(ctx, id, "users:", "useritem:", true, "items", `SELECT items.id, agents.title || ' ' || items.model || ' [' || itemtypes.typedesc || ', ID:' || items.id || ']' AS nodetext FROM items, agents, itemtypes WHERE items.itemtypeid = itemtypes.id AND items.userid = ? AND agents.id = items.manufacturerid ORDER BY agents.title`)
	}
	if strings.HasPrefix(id, "itemtypes:") {
		return r.browseByID(ctx, id, "itemtypes:", "typeitem:", true, "items", `SELECT items.id, agents.title || ' ' || items.model || ' [' || itemtypes.typedesc || ', ID:' || items.id || ']' AS nodetext FROM items, agents, itemtypes WHERE items.itemtypeid = itemtypes.id AND agents.id = items.manufacturerid AND items.itemtypeid = ? ORDER BY agents.title`)
	}
	if strings.HasPrefix(id, "agenthw:") {
		return r.browseByID(ctx, id, "agenthw:", "agenthwitem:", true, "items", `SELECT items.id, items.model || ' [' || itemtypes.typedesc || ', ID:' || items.id || ']' AS nodetext FROM items, itemtypes WHERE items.manufacturerid = ? AND items.itemtypeid = itemtypes.id ORDER BY nodetext`)
	}
	if strings.HasPrefix(id, "agentsw:") {
		return r.browseByID(ctx, id, "agentsw:", "agentswsoftware:", true, "software", `SELECT software.id, software.stitle || ' ' || software.sversion || ' [ID:' || software.id || ']' AS nodetext FROM software WHERE manufacturerid = ?`)
	}
	if strings.HasPrefix(id, "agentvendor:") {
		return r.browseByID(ctx, id, "agentvendor:", "", true, "", `SELECT node.id, node.nodetext, node.nodeprefix, node.resource FROM (
			SELECT invoices.id AS id, invoices.number || ' ' || date(invoices.date, 'unixepoch') || ' [ID:' || invoices.id || ']' AS nodetext, 'vendorinvoice:' AS nodeprefix, 'invoices' AS resource FROM invoices WHERE invoices.vendorid = ?
			UNION ALL
			SELECT items.id AS id, items.model || ' [' || itemtypes.typedesc || ', ID:' || items.id || ']' AS nodetext, 'vendoritem:' AS nodeprefix, 'items' AS resource FROM items, itemtypes WHERE items.itemtypeid = itemtypes.id AND TRIM(items.origin) != '' AND TRIM(items.origin) = (SELECT TRIM(title) FROM agents WHERE agents.id = ?)
		) AS node ORDER BY node.nodetext`)
	}
	if strings.HasPrefix(id, "agentbuyer:") {
		return r.browseByID(ctx, id, "agentbuyer:", "buyerinvoice:", true, "invoices", `SELECT invoices.id, invoices.number || ' ' || date(invoices.date, 'unixepoch') || ' [ID:' || invoices.id || ']' AS nodetext FROM invoices WHERE buyerid = ? ORDER BY invoices.date`)
	}
	if strings.HasPrefix(id, "agentcontractor:") {
		return r.browseByID(ctx, id, "agentcontractor:", "contractorcontract:", true, "contracts", `SELECT contracts.id, contracts.number || ' ' || date(contracts.startdate, 'unixepoch') || ' [ID:' || contracts.id || ']' AS nodetext FROM contracts WHERE contractorid = ? ORDER BY contracts.startdate`)
	}
	return nil, "", false, nil
}

func (r *DomainRepository) browseByID(ctx context.Context, raw, prefix, nodePrefix string, leaf bool, resource, query string) (*sql.Rows, string, bool, error) {
	value := strings.TrimPrefix(raw, prefix)
	var id int64
	if _, err := fmt.Sscan(value, &id); err != nil || id <= 0 {
		return nil, "", false, fmt.Errorf("invalid browse node id")
	}
	args := make([]interface{}, strings.Count(query, "?"))
	for i := range args {
		args[i] = id
	}
	rows, err := r.QueryContext(ctx, query, args...)
	return rows, nodePrefix, leaf, err
}
