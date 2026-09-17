package domain

import (
	"errors"
	"strings"
)

func ValidateItem(req ItemPayload) error {
	if req.ItemTypeID == 0 {
		return errors.New("itemTypeId is required")
	}
	if req.ManufacturerID == 0 {
		return errors.New("manufacturerId is required")
	}
	if strings.TrimSpace(req.Principal) == "" {
		return errors.New("principal is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return errors.New("model is required")
	}
	if err := validateMACList(req.MACs); err != nil {
		return err
	}
	if err := validateIPList(req.IPv4, false); err != nil {
		return err
	}
	if err := validateIPList(req.IPv6, true); err != nil {
		return err
	}
	if err := validateRemAdmIPList(req.RemAdmIP); err != nil {
		return err
	}
	return nil
}
