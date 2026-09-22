package auth

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// loginLockedError 登录锁定错误，Minutes 为按最近一次密码失败时间推算的剩余等待分钟数
type loginLockedError struct {
	Minutes int64
}

func (e loginLockedError) Error() string {
	return fmt.Sprintf("密码连续错误次数过多，请于 %d 分钟后再试", e.Minutes)
}

// ensureLoginAllowed 判定账号密码失败次数达到阈值且仍在锁定时长内时返回锁定错误
func (a *Router) ensureLoginAllowed(ctx context.Context, username string, maxFailures, lockMinutes int) error {
	normalized := strings.ToLower(strings.TrimSpace(username))
	var count int
	var latest int64
	if err := a.db.QueryRowContext(ctx, "SELECT COUNT(*),COALESCE(MAX(failed_at),0) FROM login_failure_log WHERE username=?", normalized).Scan(&count, &latest); err != nil {
		return err
	}
	if count < maxFailures {
		return nil
	}
	if remaining := latest + int64(lockMinutes*60) - time.Now().Unix(); remaining > 0 {
		return loginLockedError{Minutes: (remaining + 59) / 60}
	}
	return nil
}

// recordLoginFailure 记录一次密码失败，并顺带清理超过一天的旧记录防止表无限增长
func (a *Router) recordLoginFailure(ctx context.Context, username string) error {
	normalized := strings.ToLower(strings.TrimSpace(username))
	now := time.Now().Unix()
	if _, err := a.db.ExecContext(ctx, "DELETE FROM login_failure_log WHERE failed_at<=?", now-86400); err != nil {
		return err
	}
	_, err := a.db.ExecContext(ctx, "INSERT INTO login_failure_log(username,failed_at) VALUES(?,?)", normalized, now)
	return err
}

// clearLoginFailures 登录成功或找回密码重置成功后清除该账号的失败计数，返回实际清除的记录数
func (a *Router) clearLoginFailures(ctx context.Context, username string) (int64, error) {
	result, err := a.db.ExecContext(ctx, "DELETE FROM login_failure_log WHERE username=?", strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
