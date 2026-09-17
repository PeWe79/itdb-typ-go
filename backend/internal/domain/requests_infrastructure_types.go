package domain

type LocationPayload struct {
	// 地点名称
	Name string `json:"name"`
	// 楼层
	Floor string `json:"floor"`
	// 变更项说明
	ChangeNote string `json:"changeNote,omitempty"`
	// 保存时刻的最终区域清单
	Areas []string `json:"areas,omitempty"`
}

type RackPayload struct {
	// 地点编号
	LocationID int64 `json:"locationId"`
	// 区域编号
	LocAreaID *int64 `json:"locAreaId"`
	// 高度（U）
	USize int64 `json:"uSize"`
	// 编号方向（0=Bottom/1=Top）
	RevNums int64 `json:"revNums"`
	// 深度（mm）
	Depth int64 `json:"depth"`
	// 注释
	Comments string `json:"comments"`
	// 型号
	Model string `json:"model"`
	// 标签
	Label string `json:"label"`
	// 变更项说明
	ChangeNote string `json:"changeNote,omitempty"`
}

type LocAreaPayload struct {
	// 区域名称
	AreaName string `json:"areaName"`
}
