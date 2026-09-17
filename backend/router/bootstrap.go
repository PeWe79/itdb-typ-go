package router

import (
	"itdb-backend/router/common"
	"net/http"
)

func (a *App) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	lookups := map[string]interface{}{}

	queries := map[string]string{
		"itemtypes":        `SELECT id, typedesc, hassoftware FROM itemtypes ORDER BY typedesc`,
		"dpttypes":         `SELECT id, dptname FROM dpttypes ORDER BY dptname`,
		"statustypes":      `SELECT id, statusdesc, color FROM statustypes ORDER BY id`,
		"filetypes":        `SELECT id, typedesc FROM filetypes ORDER BY id`,
		"contracttypes":    `SELECT id, name FROM contracttypes ORDER BY id`,
		"contractsubtypes": `SELECT id, contypeid, name FROM contractsubtypes ORDER BY name`,
		"tags": `SELECT
            tags.id,
            tags.name,
            (SELECT COUNT(*) FROM tag2item WHERE tagid = tags.id) AS itemCount,
            (SELECT COUNT(*) FROM tag2software WHERE tagid = tags.id) AS softwareCount
        FROM tags ORDER BY name`,
		"agents":    `SELECT id, type, title, urls FROM agents ORDER BY title`,
		"users":     `SELECT id, username, usertype FROM users ORDER BY username`,
		"locations": `SELECT id, name, floor FROM locations ORDER BY name`,
		"locareas":  `SELECT id, locationid, areaname FROM locareas ORDER BY areaname`,
		"racks":     `SELECT id, locationid, locareaid, label, usize, model, revnums FROM racks ORDER BY id`,
		"items_ref": `SELECT items.id, items.label, items.model, items.sn, items.sn2, items.sn3, items.dnsname, items.principal, items.manufacturerid,
            itemtypes.typedesc AS itemtype, COALESCE(itemtypes.hassoftware, 1) AS hassoftware, agents.title AS manufacturer,
            statustypes.statusdesc AS status, statustypes.color AS statuscolor
            FROM items
            LEFT JOIN itemtypes ON itemtypes.id = items.itemtypeid
            LEFT JOIN agents ON agents.id = items.manufacturerid
            LEFT JOIN statustypes ON statustypes.id = items.status
            ORDER BY items.id DESC`,
		"software_ref": `SELECT software.id, software.stitle, software.sversion, software.manufacturerid, agents.title AS manufacturer
            FROM software
            LEFT JOIN agents ON agents.id = software.manufacturerid
            ORDER BY software.id DESC`,
		"invoices_ref": `SELECT invoices.id, invoices.number, invoices.description, invoices.date, invoices.vendorid, invoices.buyerid, agents.title AS vendor,
            COALESCE((
                SELECT GROUP_CONCAT(
                    '(' || CAST(files.id AS TEXT) || ') ' ||
                    CASE
                        WHEN TRIM(COALESCE(files.title, '')) = '' THEN COALESCE(files.fname, '')
                        WHEN TRIM(COALESCE(files.fname, '')) = '' THEN COALESCE(files.title, '')
                        ELSE COALESCE(files.title, '') || ' / ' || COALESCE(files.fname, '')
                    END,
                    ' | '
                )
                FROM files
                JOIN invoice2file ON files.id = invoice2file.fileid
                WHERE invoice2file.invoiceid = invoices.id
                ORDER BY files.id
            ), '') AS files
            FROM invoices
            LEFT JOIN agents ON agents.id = invoices.vendorid
            ORDER BY invoices.id DESC`,
		"contracts_ref": `SELECT contracts.id, contracts.number, contracts.title, contracts.startdate, contracts.currentenddate, agents.title AS contractor
            FROM contracts
            LEFT JOIN agents ON agents.id = contracts.contractorid
            ORDER BY contracts.id DESC`,
		"files_ref": `SELECT files.id, files.title, files.fname, files.date, files.type, filetypes.typedesc AS typeDesc FROM files LEFT JOIN filetypes ON filetypes.id = files.type ORDER BY files.id DESC`,
	}

	for key, q := range queries {
		rows, e := a.fetchRows(q)
		if e != nil {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
			return
		}
		lookups[key] = rows
	}

	common.WriteJSON(w, http.StatusOK, lookups)
}
