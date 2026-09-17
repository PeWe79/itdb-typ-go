package service

import (
	"database/sql"
	"errors"
	"fmt"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/internal/common/statustypes"
	"strings"
)

func EnforceDictionaryDeleteRules(tx *sql.Tx, name string, id int64) error {
	switch name {
	case "itemtypes":
		var desc sql.NullString
		if err := tx.QueryRow(`SELECT typedesc FROM itemtypes WHERE id = ?`, id).Scan(&desc); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if IsBuiltinDictionaryRow("itemtypes", id, desc.String) {
			return errors.New("内置硬件类型不可删除")
		}
		var count int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM items WHERE itemtypeid = ?`, id).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("该硬件类型已被 %d 条硬件记录使用，无法删除", count)
		}
	case "filetypes":
		var desc sql.NullString
		if err := tx.QueryRow(`SELECT typedesc FROM filetypes WHERE id = ?`, id).Scan(&desc); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if IsBuiltinDictionaryRow("filetypes", id, desc.String) {
			return errors.New("内置文件类型不可删除")
		}
		var count int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM files WHERE type = ?`, id).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("该文件类型已被 %d 个文件记录使用，无法删除", count)
		}
	case "statustypes":
		var desc sql.NullString
		err := tx.QueryRow(`SELECT statusdesc FROM statustypes WHERE id = ?`, id).Scan(&desc)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if statustypes.IsProtectedStatusType(id, desc.String) {
			return errors.New("内置状态类型不可删除")
		}
		var count int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM items WHERE status = ?`, id).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("该状态类型已被 %d 条硬件记录使用，无法删除", count)
		}
	case "dpttypes":
		var count int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM items WHERE dptid = ?`, id).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("该所属部门已被 %d 条硬件记录使用，无法删除", count)
		}
	case "contracttypes":
		var typeName sql.NullString
		if err := tx.QueryRow(`SELECT name FROM contracttypes WHERE id = ?`, id).Scan(&typeName); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if IsBuiltinDictionaryRow("contracttypes", id, typeName.String) {
			return errors.New("内置合同类型不可删除")
		}
		var count int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM contracts WHERE type = ?`, id).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("该合同类型已被 %d 条合同记录使用，无法删除", count)
		}
		var subtypeCount int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM contracts WHERE subtype IN (SELECT id FROM contractsubtypes WHERE contypeid = ?)`, id).Scan(&subtypeCount); err != nil {
			return err
		}
		if subtypeCount > 0 {
			return fmt.Errorf("该合同类型下的合同子类型已被 %d 条合同记录使用，无法删除", subtypeCount)
		}
		if _, err := tx.Exec(`DELETE FROM contractsubtypes WHERE contypeid = ?`, id); err != nil {
			return err
		}
	case "tags":
		var itemCount, softCount int64
		if err := tx.QueryRow(`SELECT COUNT(*) FROM tag2item JOIN items ON items.id = tag2item.itemid WHERE tag2item.tagid = ?`, id).Scan(&itemCount); err != nil {
			return err
		}
		if err := tx.QueryRow(`SELECT COUNT(*) FROM tag2software JOIN software ON software.id = tag2software.softwareid WHERE tag2software.tagid = ?`, id).Scan(&softCount); err != nil {
			return err
		}
		var lines []string
		if itemCount > 0 {
			lines = append(lines, fmt.Sprintf("该标记已被 %d 条硬件记录使用，无法删除", itemCount))
		}
		if softCount > 0 {
			lines = append(lines, fmt.Sprintf("该标记已被 %d 条软件记录使用，无法删除", softCount))
		}
		if len(lines) > 0 {
			return errors.New(strings.Join(lines, "\n"))
		}
	case "contractsubtypes":
		var count int64
		if err := tx.QueryRow(`SELECT COUNT(id) FROM contracts WHERE subtype = ?`, id).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("该合同子类型已被 %d 条合同记录使用，无法删除", count)
		}
		return nil
	default:
		return errors.New("不支持的字典类型")
	}
	return nil
}

func EnforceDictionaryUpdateRules(tx *sql.Tx, name string, id int64) error {
	switch name {
	case "itemtypes":
		var desc sql.NullString
		if err := tx.QueryRow(`SELECT typedesc FROM itemtypes WHERE id = ?`, id).Scan(&desc); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if IsBuiltinDictionaryRow("itemtypes", id, desc.String) {
			return errors.New("内置硬件类型不可编辑")
		}
	case "contracttypes":
		var typeName sql.NullString
		if err := tx.QueryRow(`SELECT name FROM contracttypes WHERE id = ?`, id).Scan(&typeName); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if IsBuiltinDictionaryRow("contracttypes", id, typeName.String) {
			return errors.New("内置合同类型不可编辑")
		}
	case "filetypes":
		var desc sql.NullString
		if err := tx.QueryRow(`SELECT typedesc FROM filetypes WHERE id = ?`, id).Scan(&desc); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if IsBuiltinDictionaryRow("filetypes", id, desc.String) {
			return errors.New("内置文件类型不可编辑")
		}
	case "statustypes":
		var desc sql.NullString
		if err := tx.QueryRow(`SELECT statusdesc FROM statustypes WHERE id = ?`, id).Scan(&desc); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("状态类型不存在")
			}
			return err
		}
		if statustypes.IsProtectedStatusType(id, desc.String) {
			return errors.New("内置状态类型不可编辑")
		}
	}
	return nil
}

// BuiltinDictionaryNames 返回指定字典的内置名称清单，与内置编号共同构成内置保护依据
func BuiltinDictionaryNames(dictionary string) []string {
	switch dictionary {
	case "itemtypes":
		return []string{"服务器", "存储", "交换机", "电话", "安防"}
	case "filetypes":
		return []string{"照片", "手册", "发票", "报价", "订单", "服务", "报告", "许可证", "合同", "其他"}
	case "contracttypes":
		return []string{"支持 & 维护"}
	}
	return nil
}

// IsBuiltinDictionaryRow 判断字典行是否为内置数据：内置编号区间或内置名称（忽略大小写与首尾空格）命中即视为内置
func IsBuiltinDictionaryRow(dictionary string, id int64, text string) bool {
	switch dictionary {
	case "itemtypes":
		if id >= 0 && id <= 5 {
			return true
		}
	case "filetypes":
		if id >= 0 && id <= 10 {
			return true
		}
	case "contracttypes":
		if id >= 0 && id <= 1 {
			return true
		}
	default:
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(text))
	for _, builtin := range BuiltinDictionaryNames(dictionary) {
		if normalized == strings.ToLower(builtin) {
			return true
		}
	}
	return false
}

// LegacyBuiltinNameTranslations 返回旧平台内置字典英文名到当前项目中文名的映射（键为小写原名）
func LegacyBuiltinNameTranslations(dictionary string) map[string]string {
	switch dictionary {
	case "filetypes":
		return map[string]string{
			"photo": "照片", "manual": "手册", "invoice": "发票", "offer": "报价",
			"order": "订单", "service": "服务", "report": "报告", "license": "许可证",
			"other": "其他", "contract": "合同",
		}
	case "contracttypes":
		return map[string]string{"support & maintenance": "支持 & 维护"}
	}
	return nil
}

func DictionaryUniqueTextRule(name string) (table string, column string, label string, scopeColumn string, err error) {
	switch name {
	case "itemtypes":
		return "itemtypes", "typedesc", "硬件类型", "", nil
	case "filetypes":
		return "filetypes", "typedesc", "文件类型", "", nil
	case "statustypes":
		return "statustypes", "statusdesc", "状态类型", "", nil
	case "dpttypes":
		return "dpttypes", "dptname", "部门", "", nil
	case "contracttypes":
		return "contracttypes", "name", "合同类型", "", nil
	case "contractsubtypes":
		return "contractsubtypes", "name", "合同子类型", "contypeid", nil
	case "tags":
		return "tags", "name", "标记", "", nil
	default:
		return "", "", "", "", errors.New("不支持的字典类型")
	}
}

func EnforceDictionaryUniqueText(tx *sql.Tx, name string, body map[string]interface{}, excludeID int64) error {
	table, column, label, scopeColumn, err := DictionaryUniqueTextRule(name)
	if err != nil {
		return err
	}

	text := strings.TrimSpace(primitives.AsString(body[column]))
	if text == "" {
		return nil
	}

	query := fmt.Sprintf("SELECT id FROM %s WHERE LOWER(TRIM(COALESCE(%s, ''))) = LOWER(TRIM(?))", table, column)
	args := []interface{}{text}
	if scopeColumn != "" {
		query += fmt.Sprintf(" AND COALESCE(%s, 0) = ?", scopeColumn)
		args = append(args, primitives.AsInt64(body[scopeColumn]))
	}
	if excludeID > 0 {
		query += " AND id <> ?"
		args = append(args, excludeID)
	}
	query += " LIMIT 1"

	var existingID int64
	if err := tx.QueryRow(query, args...).Scan(&existingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	return fmt.Errorf("%s %s 已存在", label, text)
}

func DictionaryInsert(name string, body map[string]interface{}) (string, []interface{}, error) {
	switch name {
	case "itemtypes":
		hassoftware := int64(1)
		if raw, ok := body["hassoftware"]; ok {
			hassoftware = primitives.AsInt64(raw)
		}
		return `INSERT INTO itemtypes (typedesc, hassoftware) VALUES (?, ?)`, []interface{}{primitives.AsString(body["typedesc"]), hassoftware}, nil
	case "filetypes":
		return `INSERT INTO filetypes (typedesc) VALUES (?)`, []interface{}{primitives.AsString(body["typedesc"])}, nil
	case "statustypes":
		desc := primitives.AsString(body["statusdesc"])
		color := statustypes.NormalizeHexColor(primitives.AsString(body["color"]))
		if fixedColor, ok := statustypes.FixedStatusTypeColor(desc); ok {
			color = fixedColor
		}
		return `INSERT INTO statustypes (statusdesc, color) VALUES (?, ?)`, []interface{}{desc, color}, nil
	case "dpttypes":
		return `INSERT INTO dpttypes (dptname) VALUES (?)`, []interface{}{primitives.AsString(body["dptname"])}, nil
	case "contracttypes":
		return `INSERT INTO contracttypes (name) VALUES (?)`, []interface{}{primitives.AsString(body["name"])}, nil
	case "contractsubtypes":
		return `INSERT INTO contractsubtypes (name, contypeid) VALUES (?, ?)`, []interface{}{primitives.AsString(body["name"]), primitives.AsInt64(body["contypeid"])}, nil
	case "tags":
		return `INSERT INTO tags (name) VALUES (?)`, []interface{}{primitives.AsString(body["name"])}, nil
	default:
		return "", nil, errors.New("不支持的字典类型")
	}
}

func DictionaryUpdate(name string, id int64, body map[string]interface{}) (string, []interface{}, error) {
	switch name {
	case "itemtypes":
		return `UPDATE itemtypes SET typedesc = ?, hassoftware = ? WHERE id = ?`, []interface{}{primitives.AsString(body["typedesc"]), primitives.AsInt64(body["hassoftware"]), id}, nil
	case "filetypes":
		return `UPDATE filetypes SET typedesc = ? WHERE id = ?`, []interface{}{primitives.AsString(body["typedesc"]), id}, nil
	case "statustypes":
		desc := primitives.AsString(body["statusdesc"])
		color := statustypes.NormalizeHexColor(primitives.AsString(body["color"]))
		if fixedColor, ok := statustypes.FixedStatusTypeColor(desc); ok {
			color = fixedColor
		}
		return `UPDATE statustypes SET statusdesc = ?, color = ? WHERE id = ?`, []interface{}{desc, color, id}, nil
	case "dpttypes":
		return `UPDATE dpttypes SET dptname = ? WHERE id = ?`, []interface{}{primitives.AsString(body["dptname"]), id}, nil
	case "contracttypes":
		return `UPDATE contracttypes SET name = ? WHERE id = ?`, []interface{}{primitives.AsString(body["name"]), id}, nil
	case "contractsubtypes":
		return `UPDATE contractsubtypes SET name = ?, contypeid = ? WHERE id = ?`, []interface{}{primitives.AsString(body["name"]), primitives.AsInt64(body["contypeid"]), id}, nil
	case "tags":
		return `UPDATE tags SET name = ? WHERE id = ?`, []interface{}{primitives.AsString(body["name"]), id}, nil
	default:
		return "", nil, errors.New("不支持的字典类型")
	}
}

func DictionaryDelete(name string) (string, error) {
	switch name {
	case "itemtypes":
		return `DELETE FROM itemtypes WHERE id = ?`, nil
	case "filetypes":
		return `DELETE FROM filetypes WHERE id = ?`, nil
	case "statustypes":
		return `DELETE FROM statustypes WHERE id = ?`, nil
	case "dpttypes":
		return `DELETE FROM dpttypes WHERE id = ?`, nil
	case "contracttypes":
		return `DELETE FROM contracttypes WHERE id = ?`, nil
	case "contractsubtypes":
		return `DELETE FROM contractsubtypes WHERE id = ?`, nil
	case "tags":
		return `DELETE FROM tags WHERE id = ?`, nil
	default:
		return "", errors.New("不支持的字典类型")
	}
}
