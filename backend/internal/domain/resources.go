package domain

// Resource identifies an ITDB asset or supporting resource exposed by the API.
type Resource string

const (
	ResourceItems        Resource = "items"
	ResourceSoftware     Resource = "software"
	ResourceInvoices     Resource = "invoices"
	ResourceContracts    Resource = "contracts"
	ResourceFiles        Resource = "files"
	ResourceAgents       Resource = "agents"
	ResourceUsers        Resource = "users"
	ResourceLocations    Resource = "locations"
	ResourceRacks        Resource = "racks"
	ResourceDictionaries Resource = "dictionaries"
	ResourceTags         Resource = "tags"
)
