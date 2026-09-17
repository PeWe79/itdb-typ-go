package settings

import (
	"strings"
	"testing"
)

func TestUserChangeDetail(t *testing.T) {
	if got := userChangeDetail("alice", "a@x.com", "Alice", []string{"viewer"}, "b@x.com", "Alice", "", []string{"viewer"}); got != "用户 alice 已更新：邮箱已修改" {
		t.Fatalf("email-only got=%q", got)
	}
	got := userChangeDetail("alice", "a@x.com", "Alice", []string{"viewer"}, "b@x.com", "爱丽丝", "secret", []string{"operator"})
	expect := "用户 alice 已更新：邮箱、显示名称、密码、角色已修改"
	if got != expect {
		t.Fatalf("all-changed got=%q want %q", got, expect)
	}
	if got := userChangeDetail("alice", "a@x.com", "Alice", []string{"viewer"}, "a@x.com", "Alice", "", []string{"viewer"}); got != "" {
		t.Fatalf("no-change should skip audit, got=%q", got)
	}
	if got := userChangeDetail("alice", "", "", nil, "", "", "", []string{"viewer", "viewer"}); got != "用户 alice 已更新：角色已修改" {
		t.Fatalf("roles-only got=%q", got)
	}
}

func TestGroupChangeDetail(t *testing.T) {
	body := settingsGroupBody{Description: "新描述", MemberIDs: []string{"1", "2"}, RoleKeys: []string{"viewer"}}
	if got := groupChangeDetail("运维组", "旧描述", []int64{1}, []string{"viewer"}, body); got != "用户群组 运维组 已更新：描述、成员已修改" {
		t.Fatalf("partial got=%q", got)
	}
	if got := groupChangeDetail("运维组", "旧描述", []int64{1}, []string{"viewer"}, settingsGroupBody{Description: "旧描述", MemberIDs: []string{"1"}, RoleKeys: []string{"viewer"}}); got != "" {
		t.Fatalf("no-change got=%q", got)
	}
	full := groupChangeDetail("运维组", "旧描述", []int64{1}, []string{"viewer"}, settingsGroupBody{Description: "描述", MemberIDs: []string{"2"}, RoleKeys: []string{"operator"}})
	if full != "用户群组 运维组 已更新：描述、成员、角色已修改" {
		t.Fatalf("full got=%q", full)
	}
}

func TestRoleChangeDetail(t *testing.T) {
	if got := roleChangeDetail("auditor", "旧描述", []string{"audit.read"}, "新描述", []string{"audit.read"}); got != "用户角色 auditor 已更新：描述已修改" {
		t.Fatalf("desc-only got=%q", got)
	}
	got := roleChangeDetail("auditor", "旧描述", []string{"audit.read"}, "旧描述", []string{"audit.read", "audit.manage"})
	expect := "用户角色 auditor 已更新：权限已修改"
	if got != expect {
		t.Fatalf("perm-only got=%q want %q", got, expect)
	}
	if got := roleChangeDetail("auditor", "旧", []string{"audit.read"}, "新", []string{"browse.read"}); got != "用户角色 auditor 已更新：描述、权限已修改" {
		t.Fatalf("both got=%q", got)
	}
	if got := roleChangeDetail("auditor", "同", []string{"a"}, "同", []string{"a"}); got != "" {
		t.Fatalf("no-change should skip audit, got=%q", got)
	}
}

func TestEqualStringSetsIgnoresOrderAndDuplicates(t *testing.T) {
	if !equalStringSets([]string{"b", "a", "b"}, []string{"a", "b"}) {
		t.Fatal("sets should be equal ignoring order and duplicates")
	}
	if equalStringSets([]string{"a"}, []string{"a", "b"}) {
		t.Fatal("different sizes should not be equal")
	}
	if !equalInt64Sets([]int64{2, 1}, []int64{1, 2}) {
		t.Fatal("int64 sets should ignore order")
	}
	if !strings.Contains(mergeChangeItems([]string{"描述", "成员"}), "描述、成员已修改") {
		t.Fatal("mergeChangeItems should append 已修改 suffix")
	}
}
