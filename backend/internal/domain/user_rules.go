package domain

import (
	"errors"
	"strings"
)

func ValidateAgentTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("title is required")
	}
	return nil
}

func ValidateUserUsername(username string) error {
	if strings.TrimSpace(username) == "" {
		return errors.New("username is required")
	}
	return nil
}
