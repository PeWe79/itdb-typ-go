package domain

type SoftwarePayload struct {
	InvoiceID        *int64  `json:"invoiceId"`
	SLicenseInfo     string  `json:"slicenseInfo"`
	Manufacturer     int64   `json:"manufacturerId"`
	Title            string  `json:"title"`
	Version          string  `json:"version"`
	Info             string  `json:"info"`
	PurchaseDate     string  `json:"purchaseDate"`
	LicenseQty       *int64  `json:"licenseQty"`
	LicenseType      int64   `json:"licenseType"`
	ItemLinks        []int64 `json:"itemLinks"`
	InvoiceLinks     []int64 `json:"invoiceLinks"`
	ContractLinks    []int64 `json:"contractLinks"`
	FileLinks        []int64 `json:"fileLinks"`
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
	ChangeNote       string  `json:"changeNote,omitempty"`
}

type InvoicePayload struct {
	VendorID         int64   `json:"vendorId"`
	BuyerID          int64   `json:"buyerId"`
	Number           string  `json:"number"`
	Description      string  `json:"description"`
	Date             string  `json:"date"`
	ItemLinks        []int64 `json:"itemLinks"`
	SoftwareLinks    []int64 `json:"softwareLinks"`
	ContractLinks    []int64 `json:"contractLinks"`
	FileLinks        []int64 `json:"fileLinks"`
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
	ChangeNote       string  `json:"changeNote,omitempty"`
}

type ContractPayload struct {
	TypeID           int64   `json:"typeId"`
	SubTypeID        int64   `json:"subTypeId"`
	ParentID         *int64  `json:"parentId"`
	Title            string  `json:"title"`
	Number           string  `json:"number"`
	Description      string  `json:"description"`
	Comments         string  `json:"comments"`
	TotalCost        string  `json:"totalCost"`
	ContractorID     int64   `json:"contractorId"`
	StartDate        string  `json:"startDate"`
	CurrentEnd       string  `json:"currentEndDate"`
	Renewals         string  `json:"renewals"`
	ItemLinks        []int64 `json:"itemLinks"`
	SoftwareLinks    []int64 `json:"softwareLinks"`
	InvoiceLinks     []int64 `json:"invoiceLinks"`
	FileLinks        []int64 `json:"fileLinks"`
	CleanupFileLinks []int64 `json:"cleanupFileLinks"`
	ChangeNote       string  `json:"changeNote,omitempty"`
}

type ContractEventPayload struct {
	SiblingID   int64  `json:"siblingId"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Description string `json:"description"`
}
