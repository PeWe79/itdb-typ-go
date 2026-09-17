package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/security"
)

var ErrInvalidCredentials = errors.New("invalid username or password")
var ErrUserNotProvisioned = errors.New("user not provisioned")

type LDAPAuthenticator func(context.Context, string, string) error
type authClaims struct {
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	UserType int64  `json:"userType"`
	Source   string `json:"source"`
	jwt.RegisteredClaims
}
type AuthWorkflow struct {
	repo   repository.AuthRepository
	secret string
	ldap   LDAPAuthenticator
}

func NewAuthWorkflow(repo repository.AuthRepository, secret string, ldap LDAPAuthenticator) *AuthWorkflow {
	return &AuthWorkflow{repo: repo, secret: secret, ldap: ldap}
}
func (s *AuthWorkflow) Login(ctx context.Context, req domain.AuthLoginRequest) (domain.AuthLoginResponse, error) {
	req.Username = strings.TrimSpace(req.Username)
	req.Mode = strings.ToLower(strings.TrimSpace(req.Mode))
	if req.Mode == "" {
		req.Mode = "local"
	}
	if req.Username == "" {
		return domain.AuthLoginResponse{}, errors.New("username is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return domain.AuthLoginResponse{}, errors.New("password is required")
	}
	if strings.TrimSpace(req.Password) == "" {
		return domain.AuthLoginResponse{}, errors.New("password is required")
	}
	isLDAP := req.Mode == "ldap"
	if isLDAP {
		if s.ldap == nil {
			return domain.AuthLoginResponse{}, errors.New("LDAP is unavailable")
		}
		if err := s.ldap(ctx, req.Username, req.Password); err != nil {
			return domain.AuthLoginResponse{}, err
		}
	}
	record, e := s.repo.FindAuthUser(ctx, req.Username)
	if e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			if isLDAP {
				return domain.AuthLoginResponse{}, ErrUserNotProvisioned
			}
			return domain.AuthLoginResponse{}, ErrInvalidCredentials
		}
		return domain.AuthLoginResponse{}, e
	}
	if !isLDAP {
		ok, legacy := security.VerifyPassword(record.Password, req.Password)
		if !ok {
			return domain.AuthLoginResponse{}, ErrInvalidCredentials
		}
		if legacy {
			if hashed, he := security.HashPassword(req.Password); he == nil && hashed != "" {
				_ = s.repo.UpdateAuthPassword(ctx, record.ID, hashed)
			}
		}
	}
	if record.Disabled != 0 {
		return domain.AuthLoginResponse{}, ErrUserNotProvisioned
	}
	user := domain.SessionUser{ID: record.ID, Username: record.Username, UserType: record.UserType, Source: req.Mode}
	if strings.EqualFold(user.Username, "admin") {
		user.UserType = 0
	}
	now := time.Now()
	claims := authClaims{UserID: user.ID, Username: user.Username, UserType: user.UserType, Source: user.Source, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(48 * time.Hour)), IssuedAt: jwt.NewNumericDate(now), Subject: user.Username}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, e := token.SignedString([]byte(s.secret))
	if e != nil {
		return domain.AuthLoginResponse{}, e
	}
	return domain.AuthLoginResponse{Token: signed, User: user}, nil
}
func (s *AuthWorkflow) Me(ctx context.Context, user domain.SessionUser) (map[string]interface{}, error) {
	desc, e := s.repo.UserDescription(ctx, user.ID)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	return map[string]interface{}{"id": user.ID, "username": user.Username, "userType": user.UserType, "userDesc": desc}, nil
}
