package settings

import (
	"context"
	"database/sql"
	"time"
)

// RecordSettingsUserLogin 记录用户最近登录时间，供用户配置列表展示。
func RecordSettingsUserLogin(ctx context.Context, db *sql.DB, userID int64) error {
	now := time.Now().Unix()
	_, err := db.ExecContext(ctx, "INSERT INTO settings_user_profiles(user_id,source,last_login_at,created_at,updated_at) VALUES(?,'local',?,?,?) ON CONFLICT(user_id) DO UPDATE SET last_login_at=excluded.last_login_at,updated_at=excluded.updated_at", userID, now, now, now)
	return err
}
