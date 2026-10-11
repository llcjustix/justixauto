package model

import "time"

type ContractState string

const (
	ContractDraft  ContractState = "draft"
	ContractSigned ContractState = "signed"
)

type ContractKind string

const (
	ContractInitial   ContractKind = "contract"
	ContractAmendment ContractKind = "amendment"
)

type ContractTemplate struct {
	ID        string `gorm:"primaryKey;type:uuid"`
	CompanyID string `gorm:"type:uuid"`
	Name      string
	Body      string
	Version   int64
	CreatedBy string `gorm:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ContractTemplate) TableName() string { return "commerce_contract_templates" }

// ContractVersion is one persisted documentary version. JSON snapshots are
// raw validated JSON so fields introduced after this binary are retained.
type ContractVersion struct {
	ID                   string  `gorm:"primaryKey;type:uuid"`
	OrderID              string  `gorm:"type:uuid"`
	CompanyID            string  `gorm:"type:uuid"`
	TemplateID           *string `gorm:"type:uuid"`
	TemplateVersion      *int64
	PriorSignedVersionID *string `gorm:"type:uuid"`
	Kind                 ContractKind
	State                ContractState
	Reference            string
	SignedOn             *time.Time `gorm:"type:date"`
	ExternalFileIDs      []byte     `gorm:"type:jsonb"`
	SignedFileIDs        []byte     `gorm:"type:jsonb"`
	Content              string
	PartySnapshot        []byte `gorm:"type:jsonb"`
	TermsSnapshot        []byte `gorm:"type:jsonb"`
	TermsDigest          string
	SourceOrderRevision  int64
	CreatedBy            string `gorm:"type:uuid"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Version              int64
}

func (ContractVersion) TableName() string { return "commerce_contract_versions" }

type ContractPartySnapshot struct {
	Buyer    ContractRequisiteSnapshot `json:"buyer"`
	Supplier ContractRequisiteSnapshot `json:"supplier"`
}
type ContractRequisiteSnapshot struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	LegalName          string `json:"legalName"`
	RegistrationNumber string `json:"registrationNumber"`
	TaxID              string `json:"taxId"`
	BankRequisites     string `json:"bankRequisites"`
	Address            string `json:"address"`
	Phone              string `json:"phone"`
	Email              string `json:"email"`
}
type ContractVehicleSnapshot struct {
	ID     string `json:"id"`
	LineID string `json:"lineId"`
	VIN    string `json:"vin"`
	Status string `json:"status"`
}
type ContractSnapshot struct {
	Parties  ContractPartySnapshot     `json:"parties"`
	Vehicles []ContractVehicleSnapshot `json:"vehicles"`
}
