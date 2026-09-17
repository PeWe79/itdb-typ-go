package service

import (
	"context"
	"database/sql"
	"itdb-backend/internal/repository"
	"strings"
)

type SoftwareWorkflow struct{ repo repository.SoftwareRepository }

func NewSoftwareWorkflow(repo repository.SoftwareRepository) *SoftwareWorkflow {
	return &SoftwareWorkflow{repo: repo}
}
func (s *SoftwareWorkflow) List(ctx context.Context, search string, limit, offset int64) (*sql.Rows, error) {
	return s.repo.ListSoftware(ctx, strings.TrimSpace(search), limit, offset)
}
func (s *SoftwareWorkflow) Get(ctx context.Context, id int64) (*sql.Rows, error) {
	return s.repo.GetSoftware(ctx, id)
}
