package domain

import "testing"

func TestValidateMACList(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr string
	}{
		{"empty skips", "", ""},
		{"single colon mac", "AA:BB:CC:DD:EE:FF", ""},
		{"dash mac", "AA-BB-CC-DD-EE-FF", ""},
		{"multi values", "AA:BB:CC:DD:EE:FF;11:22:33:44:55:66", ""},
		{"bad length", "AA:BB:CC", "invalid macs"},
		{"bad char", "AA:BB:CC:DD:EE:ZZ", "invalid macs"},
	}
	for _, tc := range cases {
		err := validateMACList(tc.value)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != tc.wantErr {
			t.Fatalf("%s: err=%v want %q", tc.name, err, tc.wantErr)
		}
	}
}

func TestValidateIPLists(t *testing.T) {
	if err := validateIPList("192.168.1.10,10.0.0.1", false); err != nil {
		t.Fatalf("valid ipv4 err=%v", err)
	}
	if err := validateIPList("999.1.1.1", false); err == nil || err.Error() != "invalid ipv4" {
		t.Fatalf("bad ipv4 err=%v", err)
	}
	if err := validateIPList("fe80::1", true); err != nil {
		t.Fatalf("valid ipv6 err=%v", err)
	}
	if err := validateIPList("not-an-ip", true); err == nil || err.Error() != "invalid ipv6" {
		t.Fatalf("bad ipv6 err=%v", err)
	}
	if err := validateRemAdmIPList("10.0.0.9"); err != nil {
		t.Fatalf("valid remadmip err=%v", err)
	}
	if err := validateRemAdmIPList("10.0.0"); err == nil || err.Error() != "invalid remadmip" {
		t.Fatalf("bad remadmip err=%v", err)
	}
}
