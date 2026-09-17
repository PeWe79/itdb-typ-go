package service

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	_ "modernc.org/sqlite"
)

func ValidateSQLiteFile(filePath string) error {
	testDB, err := sql.Open("sqlite", filePath)
	if err != nil {
		return fmt.Errorf("无法打开数据库文件")
	}
	defer testDB.Close()
	var result string
	if err := testDB.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		return fmt.Errorf("数据库完整性校验失败")
	}
	if result != "ok" {
		return fmt.Errorf("数据库完整性校验未通过: %s", result)
	}
	var name string
	if err := testDB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='items' LIMIT 1").Scan(&name); err != nil {
		return fmt.Errorf("上传的数据库缺少 items 表，不是有效的 ITDB 数据库")
	}
	return nil
}
func CopyFile(src, dst string) error {
	in, e := os.Open(src)
	if e != nil {
		return e
	}
	defer in.Close()
	out, e := os.Create(dst)
	if e != nil {
		return e
	}
	if _, e = io.Copy(out, in); e != nil {
		_ = out.Close()
		return e
	}
	if e = out.Sync(); e != nil {
		_ = out.Close()
		return e
	}
	return out.Close()
}
