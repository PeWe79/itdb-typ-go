package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/pkg/database"
	"itdb-backend/internal/repository"
	"strings"
)

type LabelPresetWorkflow struct {
	repo  repository.ToolRepository
	audit *AuditService
}

func NewLabelPresetWorkflow(repo repository.ToolRepository, audit *AuditService) *LabelPresetWorkflow {
	return &LabelPresetWorkflow{repo: repo, audit: audit}
}
func (s *LabelPresetWorkflow) List(ctx context.Context) (*sql.Rows, error) {
	return s.repo.ListLabelPresets(ctx)
}
func (s *LabelPresetWorkflow) Create(ctx context.Context, actor UserActor, body map[string]interface{}) (int64, bool, error) {
	name := strings.TrimSpace(primitives.AsString(body["name"]))
	if name == "" {
		return 0, false, fmt.Errorf("name is required")
	}
	a := labelPresetArgs(body)
	id, e := s.repo.FindLabelPresetIDByName(ctx, name)
	if e != nil {
		return 0, false, e
	}
	if id > 0 {
		old, e := loadLabelPresetRow(ctx, s.repo, id)
		if e != nil {
			return 0, false, e
		}
		if e := s.repo.UpdateLabelPreset(ctx, id, a...); e != nil {
			return 0, false, e
		}
		changes := labelPresetChangedFields(old, body)
		if len(changes) == 0 {
			return id, false, nil
		}
		event := AuditEvent{
			Module: AuditModuleLabels,
			Action: "更新标签预设",
			Target: name,
			Detail: FormatSimpleName(name, id, true) + " 已更新：" + strings.Join(changes, "、") + "配置",
			Result: AuditResultSuccess,
		}
		return id, false, s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
	}
	r, e := s.repo.CreateLabelPreset(ctx, a...)
	if e != nil {
		return 0, false, e
	}
	id, e = r.LastInsertId()
	if e != nil {
		return 0, false, e
	}
	event := AuditEvent{
		Module: AuditModuleLabels,
		Action: "新增标签预设",
		Target: name,
		Detail: FormatSimpleName(name, id, true) + " 已创建",
		Result: AuditResultSuccess,
	}
	return id, true, s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}
func (s *LabelPresetWorkflow) Delete(ctx context.Context, actor UserActor, id int64) error {
	var name string
	if err := s.repo.QueryRowContext(ctx, `SELECT COALESCE(name, '') FROM labelpapers WHERE id = ?`, id).Scan(&name); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	r, e := s.repo.DeleteLabelPreset(ctx, id)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	event := AuditEvent{
		Module: AuditModuleLabels,
		Action: "删除标签预设",
		Target: strings.TrimSpace(name),
		Detail: FormatSimpleName(name, id, true) + " 已删除",
		Result: AuditResultSuccess,
	}
	return s.audit.RecordEvent(ctx, actor.Username, actor.IP, event)
}

// labelPresetArgs 标签预设的写入参数列表
func labelPresetArgs(body map[string]interface{}) []interface{} {
	return []interface{}{primitives.AsInt64(body["rows"]), primitives.AsInt64(body["cols"]), primitives.AsString(body["lwidth"]), primitives.AsString(body["lheight"]), primitives.AsString(body["vpitch"]), primitives.AsString(body["hpitch"]), primitives.AsString(body["tmargin"]), primitives.AsString(body["bmargin"]), primitives.AsString(body["lmargin"]), primitives.AsString(body["rmargin"]), primitives.AsString(body["name"]), primitives.AsString(body["border"]), primitives.AsString(body["padding"]), primitives.AsString(body["fontsize"]), primitives.AsString(body["headerfontsize"]), primitives.AsString(body["barcodesize"]), primitives.AsString(body["idfontsize"]), primitives.AsInt64(body["wantbarcode"]), primitives.AsInt64(body["wantheadertext"]), primitives.AsInt64(body["wantheaderimage"]), primitives.AsString(body["headertext"]), primitives.AsString(body["image"]), primitives.AsString(body["imagewidth"]), primitives.AsString(body["imageheight"]), primitives.AsString(body["papersize"]), primitives.AsString(body["qrtext"]), primitives.AsInt64(body["wantnotext"]), primitives.AsInt64(body["wantraligntext"]), primitives.AsInt64(body["labelskip"])}
}

// labelPresetFieldNames 保存请求字段与编辑界面名称的对应关系（按界面分组顺序）
var labelPresetFieldNames = []struct {
	key   string
	label string
	kind  string
}{
	{"papersize", "纸张规格", "text"},
	{"rows", "行数", "number"},
	{"cols", "列数", "number"},
	{"labelskip", "顶部跳过标签数", "number"},
	{"lwidth", "标签宽度", "text"},
	{"lheight", "标签高度", "text"},
	{"hpitch", "水平间距", "text"},
	{"vpitch", "垂直间距", "text"},
	{"tmargin", "上边距", "text"},
	{"bmargin", "下边距", "text"},
	{"lmargin", "左边距", "text"},
	{"rmargin", "右边距", "text"},
	{"border", "边框颜色", "text"},
	{"padding", "文本填充", "text"},
	{"fontsize", "正文字号", "text"},
	{"idfontsize", "编号字号", "text"},
	{"headerfontsize", "标题字号", "text"},
	{"barcodesize", "二维码边长", "text"},
	{"imagewidth", "图片宽", "text"},
	{"imageheight", "图片高", "text"},
	{"wantbarcode", "打印二维码", "number"},
	{"wantheadertext", "打印页头文字", "number"},
	{"wantheaderimage", "打印页头图片", "number"},
	{"wantnotext", "仅打印条码", "number"},
	{"wantraligntext", "条码右侧显示文字", "number"},
	{"headertext", "页头文字", "text"},
	{"qrtext", "二维码 URL 前缀", "text"},
	{"image", "页头图片路径", "text"},
}

// loadLabelPresetRow 读取标签预设当前整行数据
func loadLabelPresetRow(ctx context.Context, exec repository.Executor, id int64) (map[string]interface{}, error) {
	rows, err := exec.QueryContext(ctx, `SELECT * FROM labelpapers WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	maps, err := database.RowsToMaps(rows)
	if err != nil {
		return nil, err
	}
	if len(maps) == 0 {
		return nil, sql.ErrNoRows
	}
	return maps[0], nil
}

// labelPresetChangedFields 对比库中旧值与保存请求，返回被修改配置项的界面名称清单；
// 数值与开关按整数比较，其余按去除首尾空白的文本比较
func labelPresetChangedFields(old map[string]interface{}, body map[string]interface{}) []string {
	changes := make([]string, 0)
	for _, field := range labelPresetFieldNames {
		var before, after string
		if field.kind == "number" {
			before = fmt.Sprintf("%d", primitives.AsInt64(old[field.key]))
			after = fmt.Sprintf("%d", primitives.AsInt64(body[field.key]))
		} else {
			before = strings.TrimSpace(primitives.AsString(old[field.key]))
			after = strings.TrimSpace(primitives.AsString(body[field.key]))
		}
		if before != after {
			changes = append(changes, field.label)
		}
	}
	return changes
}
