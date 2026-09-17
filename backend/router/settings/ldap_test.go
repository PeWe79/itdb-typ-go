package settings

import (
	"strings"
	"testing"

	"github.com/go-ldap/ldap/v3"
)

func TestBuildLDAPUserFilterEscapesUsernameAndCombinesFilter(t *testing.T) {
	filter, err := buildLDAPUserFilter("sAMAccountName", "(memberOf=%{user})", "alice*)(cn=*)")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filter, "(sAMAccountName=alice\\2a\\29\\28cn=\\2a\\29)") {
		t.Fatalf("escaped user filter = %q", filter)
	}
	if !strings.HasPrefix(filter, "(&") || !strings.Contains(filter, "(memberOf=alice\\2a\\29\\28cn=\\2a\\29)") {
		t.Fatalf("combined LDAP filter = %q", filter)
	}
}

func TestBuildLDAPUserFilterUsesDefaultAttribute(t *testing.T) {
	filter, err := buildLDAPUserFilter("", "", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if filter != "(sAMAccountName=alice)" {
		t.Fatalf("default LDAP filter = %q", filter)
	}
}

func TestBuildLDAPUserFilterExpandsUsernamePlaceholder(t *testing.T) {
	filter, err := buildLDAPUserFilter("(sAMAccountName={username})", "", "hongzelong")
	if err != nil {
		t.Fatal(err)
	}
	if filter != "(sAMAccountName=hongzelong)" {
		t.Fatalf("username placeholder filter = %q", filter)
	}
	filter, err = buildLDAPUserFilter("(sAMAccountName=%{user})", "", "hongzelong")
	if err != nil {
		t.Fatal(err)
	}
	if filter != "(sAMAccountName=hongzelong)" {
		t.Fatalf("user placeholder filter = %q", filter)
	}
}

func TestBuildLDAPUserFilterWrapsGroupDN(t *testing.T) {
	groupDN := "cn=yunwei,cn=users,dc=vdesktop,dc=sunline,dc=cn"
	filter, err := buildLDAPUserFilter("(sAMAccountName={username})", groupDN, "hongzelong")
	if err != nil {
		t.Fatal(err)
	}
	expected := "(&(sAMAccountName=hongzelong)(memberOf=" + ldap.EscapeFilter(groupDN) + "))"
	if filter != expected {
		t.Fatalf("group DN filter = %q, want %q", filter, expected)
	}
}

func TestLDAPProviderOptionsFromConfig(t *testing.T) {
	options := ldapProviderOptionsFromConfig(`{"port":636,"useTLS":true,"startTLS":false,"insecureSkipVerify":"true","timeoutSeconds":"8"}`)
	if options.Port != 636 || !options.UseTLS || options.StartTLS || !options.InsecureSkipVerify || options.TimeoutSeconds != 8 {
		t.Fatalf("unexpected options: %+v", options)
	}
	if options := ldapProviderOptionsFromConfig(""); options != (ldapProviderOptions{}) {
		t.Fatalf("empty config should yield zero options: %+v", options)
	}
	if options := ldapProviderOptionsFromConfig("not-json"); options != (ldapProviderOptions{}) {
		t.Fatalf("invalid config should yield zero options: %+v", options)
	}
}
