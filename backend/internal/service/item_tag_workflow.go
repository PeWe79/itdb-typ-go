package service

import (
	"context"
	"errors"
	"strings"

	"itdb-backend/internal/repository"
)

type ItemTagWorkflow struct {
	repo  repository.Repository
	tags  *TagWorkflow
	audit *AuditService
}

func NewItemTagWorkflow(repo repository.Repository, tags *TagWorkflow, audit *AuditService) *ItemTagWorkflow {
	return &ItemTagWorkflow{repo: repo, tags: tags, audit: audit}
}

func (s *ItemTagWorkflow) Mutate(ctx context.Context, actor UserActor, itemID int64, name, action string) error {
	name = strings.TrimSpace(name)
	add, err := normalizeTagAction(action)
	if err != nil {
		return err
	}
	tagID, err := s.tags.EnsureTagLogged(ctx, actor, name)
	if err != nil {
		return err
	}
	var query string
	var args []interface{}
	if add {
		query = `INSERT INTO tag2item (tagid,itemid) SELECT ?,? WHERE NOT EXISTS (SELECT 1 FROM tag2item WHERE tagid=? AND itemid=?)`
		args = []interface{}{tagID, itemID, tagID, itemID}
	} else {
		query = `DELETE FROM tag2item WHERE tagid=? AND itemid=?`
		args = []interface{}{tagID, itemID}
	}
	if _, err := s.repo.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	itemName, err := LoadItemName(ctx, s.repo, itemID)
	if err != nil {
		return err
	}
	verb := "已关联硬件"
	if !add {
		verb = "已解除关联硬件"
	}
	event := AuditEvent{
		Module: AuditModuleCatalog,
		Action: tagAssociationAction(add),
		Target: name,
		Detail: FormatSimpleName(name, tagID, false) + " " + verb + "：" + itemName,
		Result: AuditResultSuccess,
	}
	return s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

type SoftwareTagWorkflow struct {
	repo  repository.Repository
	tags  *TagWorkflow
	audit *AuditService
}

func NewSoftwareTagWorkflow(repo repository.Repository, tags *TagWorkflow, audit *AuditService) *SoftwareTagWorkflow {
	return &SoftwareTagWorkflow{repo: repo, tags: tags, audit: audit}
}

func (s *SoftwareTagWorkflow) Mutate(ctx context.Context, actor UserActor, softwareID int64, name, action string) error {
	name = strings.TrimSpace(name)
	add, err := normalizeTagAction(action)
	if err != nil {
		return err
	}
	tagID, err := s.tags.EnsureTagLogged(ctx, actor, name)
	if err != nil {
		return err
	}
	var query string
	var args []interface{}
	if add {
		query = `INSERT INTO tag2software (tagid,softwareid) SELECT ?,? WHERE NOT EXISTS (SELECT 1 FROM tag2software WHERE tagid=? AND softwareid=?)`
		args = []interface{}{tagID, softwareID, tagID, softwareID}
	} else {
		query = `DELETE FROM tag2software WHERE tagid=? AND softwareid=?`
		args = []interface{}{tagID, softwareID}
	}
	if _, err := s.repo.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	softwareName, err := LoadSoftwareName(ctx, s.repo, softwareID)
	if err != nil {
		return err
	}
	verb := "已关联软件"
	if !add {
		verb = "已解除关联软件"
	}
	event := AuditEvent{
		Module: AuditModuleCatalog,
		Action: tagAssociationAction(add),
		Target: name,
		Detail: FormatSimpleName(name, tagID, false) + " " + verb + "：" + softwareName,
		Result: AuditResultSuccess,
	}
	return s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

// normalizeTagAction 归一标记关联动作：add/associate 为关联，remove/delete 为解除
func normalizeTagAction(action string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "", "add", "associate":
		return true, nil
	case "remove", "delete", "deassociate":
		return false, nil
	default:
		return false, errors.New("action must be add or remove")
	}
}

// tagAssociationAction 标记关联的操作名：新增标记关联/删除标记关联
func tagAssociationAction(add bool) string {
	if add {
		return "新增标记关联"
	}
	return "删除标记关联"
}
