package service

import (
	"context"
	"database/sql"

	"itdb-backend/internal/repository"
)

type ItemActionWorkflow struct {
	repo  repository.ItemRepository
	audit *AuditService
}

func NewItemActionWorkflow(repo repository.ItemRepository, audit *AuditService) *ItemActionWorkflow {
	return &ItemActionWorkflow{repo: repo, audit: audit}
}
func (s *ItemActionWorkflow) List(ctx context.Context, itemID int64) (*sql.Rows, error) {
	return s.repo.ListItemActions(ctx, itemID)
}
