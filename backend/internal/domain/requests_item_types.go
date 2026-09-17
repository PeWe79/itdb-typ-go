package domain

type ItemPayload struct {
	// 标签
	Label string `json:"label"`
	// 硬件类型编号
	ItemTypeID int64 `json:"itemTypeId"`
	// 用途
	Function string `json:"function"`
	// 厂商编号
	ManufacturerID int64 `json:"manufacturerId"`
	// 维保信息
	WarrInfo string `json:"warrInfo"`
	// 型号
	Model string `json:"model"`
	// 设备序列号
	SN string `json:"sn"`
	// 序列号 2
	SN2 string `json:"sn2"`
	// Service Tag
	SN3 string `json:"sn3"`
	// 采购来源
	Origin string `json:"origin"`
	// 维保月数
	WarrantyMonths *int64 `json:"warrantyMonths"`
	// 采购日期
	PurchaseDate string `json:"purchaseDate"`
	// 采购价格
	PurchPrice string `json:"purchPrice"`
	// DNS 名称
	DNSName string `json:"dnsName"`
	// 所属部门编号
	DptID *int64 `json:"dptId"`
	// 负责人
	Principal string `json:"principal"`
	// 地点编号
	LocationID *int64 `json:"locationId"`
	// 区域编号
	LocAreaID *int64 `json:"locAreaId"`
	// 使用人编号
	UserID *int64 `json:"userId"`
	// 维护信息
	MaintenanceInfo string `json:"maintenanceInfo"`
	// 注释
	Comments string `json:"comments"`
	// 是否从属部件
	IsPart int64 `json:"isPart"`
	// 机架编号
	RackID *int64 `json:"rackId"`
	// 机架位置
	RackPosition *int64 `json:"rackPosition"`
	// 机架深度方向
	RackPosDepth *int64 `json:"rackPosDepth"`
	// 是否机架式
	RackMountable int64 `json:"rackMountable"`
	// 大小（U）
	USize *int64 `json:"uSize"`
	// 状态
	Status int64 `json:"status"`
	// MAC 地址
	MACs string `json:"macs"`
	// IPv4 地址
	IPv4 string `json:"ipv4"`
	// IPv6 地址
	IPv6     string `json:"ipv6"`
	RemAdmIP string `json:"remAdmIp"`
	// 硬盘
	HD string `json:"hd"`
	// CPU
	CPU string `json:"cpu"`
	// CPU 数量
	CPUNo       string `json:"cpuNo"`
	CoresPerCPU string `json:"coresPerCpu"`
	// 内存
	RAM string `json:"ram"`
	// 磁盘阵列
	Raid string `json:"raid"`
	// 阵列配置
	RaidConfig string `json:"raidConfig"`
	// 面板端口
	PanelPort string `json:"panelPort"`
	// 交换机编号
	SwitchID *int64 `json:"switchId"`
	// 交换机端口
	SwitchPort string `json:"switchPort"`
	// 网络端口
	Ports string `json:"ports"`
	// 内部硬件关联
	ItemLinks []int64 `json:"itemLinks"`
	// 单据关联
	InvoiceLinks []int64 `json:"invoiceLinks"`
	// 软件关联
	SoftwareLinks []int64 `json:"softwareLinks"`
	// 合同关联
	ContractLinks []int64 `json:"contractLinks"`
	// 文件关联
	FileLinks []int64 `json:"fileLinks"`
	// 保存后清理的文件编号
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
	// 变更项说明
	ChangeNote string `json:"changeNote"`
}

type TagMutationPayload struct {
	// 标记名称
	Name string `json:"name"`
	// 动作（add/remove）
	Action string `json:"action"`
}
