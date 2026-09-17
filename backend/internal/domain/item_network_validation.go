package domain

import (
	"fmt"
	"net"
	"strings"
)

func splitNetworkValues(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '，' || r == '；' })
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func isValidMAC(value string) bool {
	normalized := strings.NewReplacer(":", "", "-", "").Replace(value)
	if len(normalized) != 12 {
		return false
	}
	for _, r := range normalized {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

func isValidIPv4(value string) bool {
	ip := net.ParseIP(value)
	return ip != nil && ip.To4() != nil && strings.Count(value, ".") == 3
}

func isValidIPv6(value string) bool {
	ip := net.ParseIP(value)
	return ip != nil && strings.Contains(value, ":")
}

func validateMACList(raw string) error {
	for _, value := range splitNetworkValues(raw) {
		if !isValidMAC(value) {
			return fmt.Errorf("invalid macs")
		}
	}
	return nil
}

func validateIPList(raw string, ipv6 bool) error {
	for _, value := range splitNetworkValues(raw) {
		if ipv6 {
			if !isValidIPv6(value) {
				return fmt.Errorf("invalid ipv6")
			}
			continue
		}
		if !isValidIPv4(value) {
			return fmt.Errorf("invalid ipv4")
		}
	}
	return nil
}

func validateRemAdmIPList(raw string) error {
	for _, value := range splitNetworkValues(raw) {
		if net.ParseIP(value) == nil {
			return fmt.Errorf("invalid remadmip")
		}
	}
	return nil
}
