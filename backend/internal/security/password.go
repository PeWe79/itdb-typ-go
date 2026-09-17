package security

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(plain string) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return "", nil
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func IsBcryptHash(stored string) bool {
	s := strings.TrimSpace(stored)
	return strings.HasPrefix(s, "$2a$") || strings.HasPrefix(s, "$2b$") || strings.HasPrefix(s, "$2y$")
}

func VerifyPassword(stored, plain string) (ok bool, legacyPlaintext bool) {
	if strings.TrimSpace(stored) == "" {
		return false, false
	}
	if IsBcryptHash(stored) {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(plain)) == nil, false
	}
	return stored == plain, true
}
