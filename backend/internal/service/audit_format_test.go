package service

import (
	"context"
	"database/sql"
	"testing"
)

func TestFormatHardwareName(t *testing.T) {
	cases := []struct {
		name         string
		manufacturer string
		model        string
		itemType     string
		id           int64
		withID       bool
		expect       string
	}{
		{name: "full", manufacturer: "联想", model: "ThinkPad", itemType: "笔记本", id: 3, withID: true, expect: "联想 ThinkPad[笔记本, ID: 3]"},
		{name: "without id", manufacturer: "联想", model: "ThinkPad", itemType: "笔记本", id: 0, withID: false, expect: "联想 ThinkPad[笔记本]"},
		{name: "missing type", manufacturer: "联想", model: "ThinkPad", itemType: "", id: 3, withID: true, expect: "联想 ThinkPad[ID: 3]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatHardwareName(tc.manufacturer, tc.model, tc.itemType, tc.id, tc.withID)
			if got != tc.expect {
				t.Fatalf("got=%q want %q", got, tc.expect)
			}
		})
	}
}

func TestFormatLocationName(t *testing.T) {
	if got := FormatLocationName("机房A", []string{"区域1", "区域2"}, "3", 5, true); got != "机房A(区域: 区域1、区域2, 楼层: 3)[ID: 5]" {
		t.Fatalf("multi-area got=%q", got)
	}
	if got := FormatLocationName("机房A", nil, "3", 5, true); got != "机房A(楼层: 3)[ID: 5]" {
		t.Fatalf("no-area got=%q", got)
	}
}

func TestFormatRackName(t *testing.T) {
	if got := FormatRackName("R01", "机房A", "区域1", 7, true); got != "R01(机房A, 区域: 区域1)[ID: 7]" {
		t.Fatalf("with-area got=%q", got)
	}
	if got := FormatRackName("R01", "机房A", "", 7, true); got != "R01(机房A)[ID: 7]" {
		t.Fatalf("no-area got=%q", got)
	}
}

func TestAuditEntityDetail(t *testing.T) {
	if got := AuditEntityDetail("硬件X", AuditOutcomeUpdate, true, "硬件类型、机架", ""); got != "硬件X 已更新：硬件类型、机架配置" {
		t.Fatalf("update got=%q", got)
	}
	if got := AuditEntityDetail("硬件X", AuditOutcomeUpdate, false, "硬件类型", "机架位置冲突"); got != "硬件X 更新失败：机架位置冲突" {
		t.Fatalf("failure got=%q", got)
	}
	if got := AuditEntityDetail("硬件X", AuditOutcomeCreate, true, "", ""); got != "硬件X 已创建" {
		t.Fatalf("create got=%q", got)
	}
}

func TestIsNameConflictError(t *testing.T) {
	if !IsNameConflictError(NewConflictError("代理名称 %s 已存在", "all")) {
		t.Fatal("name-conflict error should be detected")
	}
	if IsNameConflictError(NewConflictError("该机架已被 %d 条硬件记录使用，无法删除", 2)) {
		t.Fatal("non-name conflict should not be detected")
	}
	if IsNameConflictError(errTestBoom) {
		t.Fatal("plain error should not be detected")
	}
}

// TestConflictReason 锁定失败详情原因的中文转换：校验错误经本地化词典翻译，冲突错误原样保留
func TestConflictReason(t *testing.T) {
	if got := ConflictReason(errString("invalid remadmip")); got != "远程管理 IP 格式无效" {
		t.Fatalf("localized got=%q", got)
	}
	if got := ConflictReason(NewConflictError("软件名称 %s 已存在", "all")); got != "软件名称 all 已存在" {
		t.Fatalf("conflict got=%q", got)
	}
	if got := ConflictReason(nil); got != "" {
		t.Fatalf("nil got=%q", got)
	}
}

var errTestBoom = error(errString("boom"))

type errString string

func (e errString) Error() string { return string(e) }

func TestTruncateAuditTarget(t *testing.T) {
	full := "类型一(子一、子二)、类型二、类型三(子三)"
	display, truncated := TruncateAuditTarget(full, 2)
	if !truncated || display != "类型一(子一、子二)、类型二..." {
		t.Fatalf("display=%q truncated=%v", display, truncated)
	}
	if display, truncated = TruncateAuditTarget("类型一、类型二", 2); truncated || display != "类型一、类型二" {
		t.Fatalf("no-truncation display=%q", display)
	}
}

func TestSplitAuditTargetNames(t *testing.T) {
	parts := SplitAuditTargetNames("类型一(子一、子二)、类型二")
	if len(parts) != 2 || parts[0] != "类型一(子一、子二)" || parts[1] != "类型二" {
		t.Fatalf("parts=%v", parts)
	}
}

func TestRecordEventTxWritesWithinTransaction(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(auditHistoryTestDDL); err != nil {
		t.Fatal(err)
	}
	audit := NewAuditService(db, 0)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	event := AuditEvent{Module: AuditModuleAssets, Action: "新增硬件", Target: "硬件编号 1", Detail: "硬件X[ID: 1] 已创建", Result: AuditResultSuccess}
	if err := audit.RecordEventTx(context.Background(), tx, "admin", "10.0.0.1", event); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var action, target, sqlText string
	if err := db.QueryRow(`SELECT action, target, sql FROM history LIMIT 1`).Scan(&action, &target, &sqlText); err != nil {
		t.Fatal(err)
	}
	if action != "新增硬件" || target != "硬件编号 1" || sqlText != "" {
		t.Fatalf("action=%s target=%s sql=%q", action, target, sqlText)
	}
}
