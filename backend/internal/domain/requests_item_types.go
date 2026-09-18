package domain

type ItemPayload struct {
	Label            string  `json:"label"`
	ItemTypeID       int64   `json:"itemTypeId"`
	Function         string  `json:"function"`
	ManufacturerID   int64   `json:"manufacturerId"`
	WarrInfo         string  `json:"warrInfo"`
	Model            string  `json:"model"`
	SN               string  `json:"sn"`
	SN2              string  `json:"sn2"`
	SN3              string  `json:"sn3"`
	Origin           string  `json:"origin"`
	WarrantyMonths   *int64  `json:"warrantyMonths"`
	PurchaseDate     string  `json:"purchaseDate"`
	PurchPrice       string  `json:"purchPrice"`
	DNSName          string  `json:"dnsName"`
	DptID            *int64  `json:"dptId"`
	Principal        string  `json:"principal"`
	LocationID       *int64  `json:"locationId"`
	LocAreaID        *int64  `json:"locAreaId"`
	UserID           *int64  `json:"userId"`
	MaintenanceInfo  string  `json:"maintenanceInfo"`
	Comments         string  `json:"comments"`
	IsPart           int64   `json:"isPart"`
	RackID           *int64  `json:"rackId"`
	RackPosition     *int64  `json:"rackPosition"`
	RackPosDepth     *int64  `json:"rackPosDepth"`
	RackMountable    int64   `json:"rackMountable"`
	USize            *int64  `json:"uSize"`
	Status           int64   `json:"status"`
	MACs             string  `json:"macs"`
	IPv4             string  `json:"ipv4"`
	IPv6             string  `json:"ipv6"`
	RemAdmIP         string  `json:"remAdmIp"`
	HD               string  `json:"hd"`
	CPU              string  `json:"cpu"`
	CPUNo            string  `json:"cpuNo"`
	CoresPerCPU      string  `json:"coresPerCpu"`
	RAM              string  `json:"ram"`
	Raid             string  `json:"raid"`
	RaidConfig       string  `json:"raidConfig"`
	PanelPort        string  `json:"panelPort"`
	SwitchID         *int64  `json:"switchId"`
	SwitchPort       string  `json:"switchPort"`
	Ports            string  `json:"ports"`
	ItemLinks        []int64 `json:"itemLinks"`
	InvoiceLinks     []int64 `json:"invoiceLinks"`
	SoftwareLinks    []int64 `json:"softwareLinks"`
	ContractLinks    []int64 `json:"contractLinks"`
	FileLinks        []int64 `json:"fileLinks"`
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
	ChangeNote       string  `json:"changeNote"`
}

type TagMutationPayload struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}
