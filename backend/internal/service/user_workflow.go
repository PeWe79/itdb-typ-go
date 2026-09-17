package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
	"itdb-backend/internal/security"
)

var ErrUsernameExists = errors.New("username already exists")
var ErrCannotRemoveAdmin = errors.New("cannot remove default admin user")

type UserActor struct {
	Username string
	IP       string
}

type UserWorkflow struct {
	repo  repository.UserRepository
	audit *AuditService
}

func NewUserWorkflow(repo repository.UserRepository, audit *AuditService) *UserWorkflow {
	return &UserWorkflow{repo: repo, audit: audit}
}

func (s *UserWorkflow) Update(ctx context.Context, actor UserActor, id int64, req domain.UserPayload) error {
	username, err := normalizeUser(req.Username)
	if err != nil {
		return err
	}
	exists, err := s.repo.UsernameExists(ctx, username, id)
	if err != nil {
		return err
	}
	if exists {
		return ErrUsernameExists
	}
	if strings.EqualFold(username, "admin") {
		req.UserType = 0
	}
	updatePassword := strings.TrimSpace(req.Password) != ""
	password := ""
	if updatePassword {
		password, err = security.HashPassword(req.Password)
		if err != nil {
			return err
		}
	}
	query := `UPDATE users SET username = ?, userdesc = ?, usertype = ? WHERE id = ?`
	args := []interface{}{username, req.UserDesc, req.UserType, id}
	if updatePassword {
		query = `UPDATE users SET username = ?, userdesc = ?, pass = ?, usertype = ? WHERE id = ?`
		args = []interface{}{username, req.UserDesc, password, req.UserType, id}
	}
	_, err = s.repo.ExecContext(ctx, query, args...)
	return err
}

// collectUserUsage 统计用户被硬件（使用人）与合同（备件录入人）引用的占用，
// 返回全部被引用场景的消息主体（不含“无法删除”后缀），供聚合提示
func (s *UserWorkflow) collectUserUsage(ctx context.Context, id int64) ([]string, error) {
	var messages []string
	var itemCount int64
	if err := s.repo.QueryRowContext(ctx, `SELECT COUNT(*) FROM items WHERE userid = ?`, id).Scan(&itemCount); err != nil {
		return nil, err
	}
	if itemCount > 0 {
		messages = append(messages, fmt.Sprintf("该用户已被 %d 条硬件记录使用，无法删除", itemCount))
	}
	contractCount, err := s.countContractsEnteredBy(ctx, id)
	if err != nil {
		return nil, err
	}
	if contractCount > 0 {
		messages = append(messages, fmt.Sprintf("该用户已被 %d 条合同记录使用，无法删除", contractCount))
	}
	return messages, nil
}

// countContractsEnteredBy 扫描合同备件串（到期前#到期后#生效日期#备注#录入日期#录入人），
// 统计任一备件由该用户录入的合同条数（按合同去重）
func (s *UserWorkflow) countContractsEnteredBy(ctx context.Context, id int64) (int64, error) {
	rows, err := s.repo.QueryContext(ctx, `SELECT COALESCE(renewals, '') FROM contracts WHERE TRIM(COALESCE(renewals, '')) <> ''`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	userID := strconv.FormatInt(id, 10)
	var count int64
	for rows.Next() {
		var renewals string
		if err := rows.Scan(&renewals); err != nil {
			return 0, err
		}
		for _, entry := range strings.Split(renewals, "|") {
			fields := strings.Split(entry, "#")
			if len(fields) >= 6 && strings.TrimSpace(fields[5]) == userID {
				count++
				break
			}
		}
	}
	return count, rows.Err()
}

func (s *UserWorkflow) Delete(ctx context.Context, actor UserActor, id int64) error {
	username, err := s.repo.FindUsername(ctx, id)
	if err != nil {
		return err
	}
	if strings.EqualFold(username, "admin") {
		return ErrCannotRemoveAdmin
	}
	messages, err := s.collectUserUsage(ctx, id)
	if err != nil {
		return err
	}
	if len(messages) > 0 {
		return NewConflictError("%s", strings.Join(messages, "\n"))
	}
	adminID, err := s.repo.FindAdminID(ctx)
	if err != nil {
		return err
	}
	tx, err := s.repo.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.repo.ReassignItems(ctx, tx, id, adminID); err != nil {
		return err
	}
	if err := s.repo.DeleteUser(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

func normalizeUser(username string) (string, error) {
	username = strings.TrimSpace(username)
	return username, domain.ValidateUserUsername(username)
}
