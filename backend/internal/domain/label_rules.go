package domain

type LabelPreviewRequest struct {
	ItemIDs    []int64 `json:"itemIds"`
	QRPrefix   string  `json:"qrPrefix"`
	HeaderText string  `json:"headerText"`
	PresetName string  `json:"presetName,omitempty"`
}
