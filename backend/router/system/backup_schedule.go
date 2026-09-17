package system

import (
	"context"
	"itdb-backend/router/settings"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"itdb-backend/internal/service"
)

// StartBackupScheduler 定时备份调度：每 20 秒检查一次 Cron 计划，命中分钟时执行一次数据库备份
func (a *Router) StartBackupScheduler() {
	if a.backupWorkflow == nil || !a.backupWorkflow.Available() {
		log.Printf("In-memory database detected, skipping backup scheduler")
		return
	}
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	lastRun := ""
	for range ticker.C {
		a.dbMu.Lock()
		base := settings.LoadSystemBaseConfig(context.Background(), a.db)
		a.dbMu.Unlock()
		if !base.BackupEnabled {
			continue
		}
		now := time.Now()
		minuteKey := now.Format("200601021504")
		if minuteKey == lastRun || !CronMatches(base.BackupCron, now) {
			continue
		}
		lastRun = minuteKey
		a.dbMu.Lock()
		path, missing, err := a.backupWorkflow.CreateScheduledBackup(context.Background(), now, strings.TrimSpace(a.cfg.UploadDir))
		a.dbMu.Unlock()
		if err != nil {
			log.Printf("Scheduled backup failed: %v", err)
			a.recordAuditEvent(context.Background(), "system", "", service.AuditModuleBackup, "定时备份数据库", "-", AuditBackupFailureDetail(err), service.AuditResultFailure)
			continue
		}
		log.Printf("Scheduled backup completed: %s", path)
		detail := appendMissingFilesDetail("已按定时备份计划 "+strings.TrimSpace(base.BackupCron)+" 执行数据库备份", missing, "，")
		a.recordAuditEvent(context.Background(), "system", "", service.AuditModuleBackup, "定时备份数据库", filepath.Base(path), detail, service.AuditResultSuccess)
		if removed, err := a.backupWorkflow.CleanupExpiredScheduledBackups(now, base.BackupRetentionDays); err != nil {
			log.Printf("Expired backup cleanup failed: %v", err)
		} else if removed > 0 {
			log.Printf("Expired backup cleanup removed %d file(s)", removed)
		}
	}
}

// CronMatches 判断时间是否命中五段 Cron 表达式（分 时 日 月 周）
func CronMatches(expr string, t time.Time) bool {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return false
	}
	return matchCronField(fields[0], t.Minute(), 0, 59) &&
		matchCronField(fields[1], t.Hour(), 0, 23) &&
		matchCronField(fields[2], t.Day(), 1, 31) &&
		matchCronField(fields[3], int(t.Month()), 1, 12) &&
		matchCronField(fields[4], int(t.Weekday()), 0, 6)
}

func matchCronField(field string, value, minValue, maxValue int) bool {
	for _, part := range strings.Split(field, ",") {
		if matchCronPart(strings.TrimSpace(part), value, minValue, maxValue) {
			return true
		}
	}
	return false
}

func matchCronPart(part string, value, minValue, maxValue int) bool {
	step := 1
	if index := strings.Index(part, "/"); index >= 0 {
		parsed, err := strconv.Atoi(part[index+1:])
		if err != nil || parsed <= 0 {
			return false
		}
		step = parsed
		part = part[:index]
	}
	lo := minValue
	hi := maxValue
	if part != "*" {
		dashIndex := strings.Index(part, "-")
		if dashIndex >= 0 {
			loParsed, errLo := strconv.Atoi(part[:dashIndex])
			hiParsed, errHi := strconv.Atoi(part[dashIndex+1:])
			if errLo != nil || errHi != nil {
				return false
			}
			lo = loParsed
			hi = hiParsed
		} else {
			parsed, err := strconv.Atoi(part)
			if err != nil {
				return false
			}
			lo = parsed
			hi = parsed
		}
	}
	if value < lo || value > hi {
		return false
	}
	if step == 1 {
		return true
	}
	return (value-lo)%step == 0
}
