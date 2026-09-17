package service

import (
	"context"
	"database/sql"
	"strings"

	"itdb-backend/internal/repository"
)

type ItemWorkflow struct{ repo repository.ItemRepository }

func NewItemWorkflow(repo repository.ItemRepository) *ItemWorkflow { return &ItemWorkflow{repo: repo} }
func (s *ItemWorkflow) List(ctx context.Context, search string, limit, offset int64) (*sql.Rows, error) {
	return s.repo.ListItems(ctx, strings.TrimSpace(search), limit, offset)
}
func (s *ItemWorkflow) Get(ctx context.Context, id int64) (*sql.Rows, error) {
	return s.repo.GetItem(ctx, id)
}
