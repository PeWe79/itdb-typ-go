package settings

import (
	"database/sql"
	"encoding/json"
	"slices"
	"testing"

	_ "modernc.org/sqlite"
)

func TestNormalizePermissionsExpandsImpliedRead(t *testing.T) {
	got := normalizePermissions([]string{"assets.items.manage"})
	if !slices.Contains(got, "assets.items.read") {
		t.Fatalf("expected implied read permission, got %v", got)
	}
}

func TestNormalizePermissionsExpandsTransitiveClosure(t *testing.T) {
	got := normalizePermissions([]string{"labels.print"})
	for _, expected := range []string{"labels.preview", "assets.items.read"} {
		if !slices.Contains(got, expected) {
			t.Fatalf("expected %s in closure, got %v", expected, got)
		}
	}
}

func TestNormalizePermissionsDropsUnknownKeys(t *testing.T) {
	got := normalizePermissions([]string{"assets.read", "settings.base.manage", "not.a.permission"})
	if slices.Contains(got, "assets.read") || slices.Contains(got, "not.a.permission") {
		t.Fatalf("expected unknown keys dropped, got %v", got)
	}
	if !slices.Contains(got, "settings.base.read") || !slices.Contains(got, "settings.base.manage") {
		t.Fatalf("expected settings.base pair expanded, got %v", got)
	}
}

func TestPermissionCatalogConsistency(t *testing.T) {
	keys := map[string]bool{}
	for _, item := range permissionCatalog {
		if keys[item.Key] {
			t.Fatalf("duplicate permission key: %s", item.Key)
		}
		keys[item.Key] = true
	}
	for _, item := range permissionCatalog {
		if item.ImpliedReadPermission != "" && !keys[item.ImpliedReadPermission] {
			t.Fatalf("implied read permission %s not in catalog", item.ImpliedReadPermission)
		}
		for _, implied := range item.ImpliedPermissions {
			if !keys[implied] {
				t.Fatalf("implied permission %s not in catalog", implied)
			}
		}
	}
	for _, key := range []string{
		"assets.items.read", "assets.items.manage", "assets.racks.manage",
		"dictionaries.tags.manage", "labels.preview", "labels.print", "labels.manage",
		"reports.read", "browse.read", "audit.read", "audit.manage", "settings.base.manage",
	} {
		if !keys[key] {
			t.Fatalf("missing expected permission key: %s", key)
		}
	}
}

func TestLegacyReplacementMapping(t *testing.T) {
	replacements := legacyReplacements()
	read := replacements["assets.read"]
	manage := replacements["assets.manage"]
	for _, key := range []string{"assets.items.read", "dictionaries.tags.read", "labels.preview", "reports.read", "browse.read", "audit.read"} {
		if !slices.Contains(read, key) {
			t.Fatalf("legacy assets.read should map to %s, got %v", key, read)
		}
	}
	for _, key := range []string{"assets.items.manage", "dictionaries.tags.manage", "labels.print", "labels.manage", "audit.read"} {
		if !slices.Contains(manage, key) {
			t.Fatalf("legacy assets.manage should map to %s, got %v", key, manage)
		}
	}
}

func TestMigrateLegacyRolePermissions(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE settings_roles (id INTEGER PRIMARY KEY AUTOINCREMENT, permissions TEXT NOT NULL DEFAULT '[]', updated_at INTEGER NOT NULL DEFAULT 0)"); err != nil {
		t.Fatalf("create table failed: %v", err)
	}
	if _, err := db.Exec("INSERT INTO settings_roles(permissions) VALUES (?)", `["assets.read","assets.manage","settings.base.read"]`); err != nil {
		t.Fatalf("insert role failed: %v", err)
	}
	if _, err := db.Exec("INSERT INTO settings_roles(permissions) VALUES (?)", `["reports.read"]`); err != nil {
		t.Fatalf("insert role failed: %v", err)
	}
	if err := migrateLegacyRolePermissions(db); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	var legacyRaw, keptRaw string
	if err := db.QueryRow("SELECT permissions FROM settings_roles WHERE id=1").Scan(&legacyRaw); err != nil {
		t.Fatalf("read role failed: %v", err)
	}
	if err := db.QueryRow("SELECT permissions FROM settings_roles WHERE id=2").Scan(&keptRaw); err != nil {
		t.Fatalf("read role failed: %v", err)
	}
	var migrated []string
	if err := json.Unmarshal([]byte(legacyRaw), &migrated); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	for _, key := range []string{"assets.items.read", "assets.items.manage", "labels.preview", "reports.read", "browse.read", "audit.read", "settings.base.read"} {
		if !slices.Contains(migrated, key) {
			t.Fatalf("expected %s migrated, got %v", key, migrated)
		}
	}
	if slices.Contains(migrated, "assets.read") || slices.Contains(migrated, "assets.manage") {
		t.Fatalf("legacy keys should be dropped, got %v", migrated)
	}
	if keptRaw != `["reports.read"]` {
		t.Fatalf("role without legacy keys should stay unchanged, got %s", keptRaw)
	}
}

func TestBuiltinRolePermissions(t *testing.T) {
	admin := builtinRolePermissions("admin")
	all := AllPermissionKeys()
	if len(admin) != len(all) {
		t.Fatalf("admin should hold all permissions, got %d of %d", len(admin), len(all))
	}
	viewer := builtinRolePermissions("viewer")
	if !slices.Contains(viewer, "labels.preview") || !slices.Contains(viewer, "reports.read") || !slices.Contains(viewer, "audit.read") {
		t.Fatalf("viewer missing read permissions, got %v", viewer)
	}
	for _, key := range viewer {
		if !isReadPermission(key) {
			t.Fatalf("viewer should only hold read permissions, got %s", key)
		}
	}
	operator := builtinRolePermissions("operator")
	if !slices.Contains(operator, "assets.items.manage") || slices.Contains(operator, "audit.manage") || slices.Contains(operator, "settings.base.manage") {
		t.Fatalf("operator permission set unexpected, got %v", operator)
	}
}
