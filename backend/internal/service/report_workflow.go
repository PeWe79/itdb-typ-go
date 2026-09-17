package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"itdb-backend/internal/domain"
	"itdb-backend/internal/repository"
)

type ReportWorkflow struct{ repo repository.Repository }

func NewReportWorkflow(repo repository.Repository) *ReportWorkflow {
	return &ReportWorkflow{repo: repo}
}
func (s *ReportWorkflow) List() []domain.ReportDefinition { return domain.AllReportDefinitions() }
func (s *ReportWorkflow) Run(ctx context.Context, name string, limit int64) (domain.ReportDefinition, *sql.Rows, error) {
	report, ok := domain.FindReportByName(strings.TrimSpace(name))
	if !ok {
		return domain.ReportDefinition{}, nil, errors.New("report not found")
	}
	if limit <= 0 {
		limit = 1000
	}
	rows, e := s.repo.QueryContext(ctx, report.Query+" LIMIT ?", limit)
	return report, rows, e
}
