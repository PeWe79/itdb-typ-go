package domain

type SoftwarePayload struct {
	// 关联网单据编号
	InvoiceID *int64 `json:"invoiceId"`
	// 许可信息
	SLicenseInfo string `json:"slicenseInfo"`
	// 厂商编号
	Manufacturer int64 `json:"manufacturerId"`
	// 标题
	Title string `json:"title"`
	// 版本
	Version string `json:"version"`
	// 其它信息
	Info string `json:"info"`
	// 采购日期
	PurchaseDate string `json:"purchaseDate"`
	// 授权数量
	LicenseQty *int64 `json:"licenseQty"`
	// 授权类型
	LicenseType int64 `json:"licenseType"`
	// 关联硬件编号
	ItemLinks []int64 `json:"itemLinks"`
	// 关联单据编号
	InvoiceLinks []int64 `json:"invoiceLinks"`
	// 关联合同编号
	ContractLinks []int64 `json:"contractLinks"`
	// 关联文件编号
	FileLinks []int64 `json:"fileLinks"`
	// 保存后清理的文件编号
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
	// 变更项说明
	ChangeNote string `json:"changeNote,omitempty"`
}

type InvoicePayload struct {
	// 供应商编号
	VendorID int64 `json:"vendorId"`
	// 采购方编号
	BuyerID int64 `json:"buyerId"`
	// 单据编号
	Number string `json:"number"`
	// 描述
	Description string `json:"description"`
	// 单据日期
	Date string `json:"date"`
	// 关联硬件编号
	ItemLinks []int64 `json:"itemLinks"`
	// 关联软件编号
	SoftwareLinks []int64 `json:"softwareLinks"`
	// 关联合同编号
	ContractLinks []int64 `json:"contractLinks"`
	// 关联文件编号
	FileLinks []int64 `json:"fileLinks"`
	// 保存后清理的文件编号
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
	// 变更项说明
	ChangeNote string `json:"changeNote,omitempty"`
}

type ContractPayload struct {
	// 合同类型编号
	TypeID int64 `json:"typeId"`
	// 合同子类型编号
	SubTypeID int64 `json:"subTypeId"`
	// 上级合同编号
	ParentID *int64 `json:"parentId"`
	// 合同标题
	Title string `json:"title"`
	// 合同编号
	Number string `json:"number"`
	// 合同描述
	Description string `json:"description"`
	// 注释
	Comments string `json:"comments"`
	// 总成本
	TotalCost string `json:"totalCost"`
	// 承包方编号
	ContractorID int64 `json:"contractorId"`
	// 开始日期
	StartDate string `json:"startDate"`
	// 当前结束日期
	CurrentEnd string `json:"currentEndDate"`
	// 备件（续签）信息
	Renewals string `json:"renewals"`
	// 关联硬件编号
	ItemLinks []int64 `json:"itemLinks"`
	// 关联软件编号
	SoftwareLinks []int64 `json:"softwareLinks"`
	// 关联单据编号
	InvoiceLinks []int64 `json:"invoiceLinks"`
	// 关联文件编号
	FileLinks []int64 `json:"fileLinks"`
	// 保存后清理的文件编号
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
	// 变更项说明
	ChangeNote string `json:"changeNote,omitempty"`
}

type ContractEventPayload struct {
	// 同日相邻事件编号
	SiblingID int64 `json:"siblingId"`
	// 开始日期
	StartDate string `json:"startDate"`
	// 结束日期
	EndDate string `json:"endDate"`
	// 事件描述
	Description string `json:"description"`
}
