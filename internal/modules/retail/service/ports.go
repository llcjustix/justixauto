package service

import (
	"context"
	"strconv"
	"time"

	"justixauto/internal/modules/retail/model"
)

// Store gives services the retail repositories and one-transaction runs.
type Store interface {
	CRM() CRMRepository
	Listings() ListingRepository
	Deals() DealRepository
	Contracts() ContractRepository
	Scoring() ScoringRepository
	Events() EventRepository
	InTx(ctx context.Context, fn func(Store) error) error
	// Bind carries the store's connection in ctx, so a call into another
	// module through a port joins it.
	Bind(ctx context.Context) context.Context
}

type ContractRepository interface {
	CreateTemplate(context.Context, *model.ContractTemplate) error
	Template(context.Context, string, string) (*model.ContractTemplate, error)
	Templates(context.Context, string) ([]model.ContractTemplate, error)
	UpdateTemplate(context.Context, *model.ContractTemplate, int64) error
	CreateContract(context.Context, *model.ContractVersion) error
	// UpdateDraft changes only the matching mutable draft row. It deliberately
	// does not touch provenance or any signed row.
	UpdateDraft(context.Context, *model.ContractVersion, int64) error
	// SignDraft changes only the matching mutable draft row. A signed version
	// is immutable and therefore never satisfies this predicate.
	SignDraft(context.Context, *model.ContractVersion, int64) error
	Contract(context.Context, string, string, string) (*model.ContractVersion, error)
	Contracts(context.Context, string, string) ([]model.ContractVersion, error)
}

type ScoringRepository interface {
	Create(context.Context, *model.DemoScoring) error
	Current(context.Context, string, string) (*model.DemoScoring, error)
}

type CRMRepository interface {
	CreateCustomer(ctx context.Context, c *model.Customer) error
	Customer(ctx context.Context, companyID, id string) (*model.Customer, error)
	Customers(ctx context.Context, companyID, query string, limit, offset int) ([]model.Customer, error)
	UpdateCustomer(ctx context.Context, c *model.Customer, expected int64) error
	CreateLead(ctx context.Context, l *model.Lead) error
	Lead(ctx context.Context, companyID, id string) (*model.Lead, error)
	Leads(ctx context.Context, f model.LeadFilter) ([]model.Lead, error)
	UpdateLead(ctx context.Context, l *model.Lead, expected int64) error
	AddContact(ctx context.Context, c *model.Contact) error
	Contacts(ctx context.Context, leadID string) ([]model.Contact, error)
	CreateTask(ctx context.Context, t *model.Task) error
	Task(ctx context.Context, companyID, id string) (*model.Task, error)
	Tasks(ctx context.Context, companyID, ownerID, status string, limit, offset int) ([]model.Task, error)
	UpdateTask(ctx context.Context, t *model.Task, expected int64) error
}

type ListingRepository interface {
	Create(ctx context.Context, l *model.Listing) error
	Get(ctx context.Context, companyID, id string) (*model.Listing, error)
	List(ctx context.Context, companyID, status string, limit, offset int) ([]model.Listing, error)
	Update(ctx context.Context, l *model.Listing, expected int64) error
	// WithdrawForVehicle closes open listings of a delivered vehicle.
	WithdrawForVehicle(ctx context.Context, vehicleID string, at time.Time) error
}

type DealRepository interface {
	Create(ctx context.Context, d *model.Deal) error
	Deal(ctx context.Context, companyID, id string) (*model.Deal, error)
	Deals(ctx context.Context, companyID string, branchIDs []string, status string, limit, offset int) ([]model.Deal, error)
	Update(ctx context.Context, d *model.Deal, expected int64) error
	LockDeal(ctx context.Context, companyID, id string) (*model.Deal, error)
	CreateInstallmentPlan(ctx context.Context, p *model.InstallmentPlan) error
	InstallmentPlan(ctx context.Context, dealID string) (*model.InstallmentPlan, error)
	CreateInvoice(ctx context.Context, i *model.Invoice) error
	Invoice(ctx context.Context, companyID, id string) (*model.Invoice, error)
	LockInvoice(ctx context.Context, companyID, id string) (*model.Invoice, error)
	LockInstallmentInvoices(ctx context.Context, companyID, dealID string, invoiceIDs []string) ([]model.Invoice, error)
	Invoices(ctx context.Context, dealID string) ([]model.Invoice, error)
	CreateInstallmentPayment(ctx context.Context, p *model.InstallmentPayment) error
	InstallmentPayment(ctx context.Context, companyID, id string) (*model.InstallmentPayment, error)
	InstallmentPaymentByReference(ctx context.Context, companyID, dealID, externalReference string) (*model.InstallmentPayment, error)
	InstallmentPaymentEvidence(ctx context.Context, paymentID string) ([]model.Evidence, error)
	UpdateInstallmentPayment(ctx context.Context, p *model.InstallmentPayment, expected int64) error
	AddEvidence(ctx context.Context, e *model.Evidence) error
	Evidence(ctx context.Context, invoiceID string) ([]model.Evidence, error)
	GetEvidence(ctx context.Context, id string) (*model.Evidence, error)
	UpdateEvidence(ctx context.Context, e *model.Evidence, expected int64) error
}

type EventRepository interface {
	Append(ctx context.Context, e *model.Event) error
	For(ctx context.Context, resourceType, id string) ([]model.Event, error)
}

// Company answers identity questions (implemented by identity).
type Company interface {
	IsMember(ctx context.Context, userID, companyID string) (bool, error)
	BranchOf(ctx context.Context, companyID, branchID string) (bool, error)
	ContractRequisitesFor(ctx context.Context, authorizedCompanyID, companyID string) (*CompanyRequisites, error)
}

type CompanyRequisites struct{ ID, Name, LegalName, RegistrationNumber, TaxID, BankRequisites, Address, Phone, Email string }

// Vehicle is what retail may know about a vehicle.
type Vehicle struct {
	ID, VIN, ModelID             string
	ModelSpecificationVersion    int
	ExteriorColor, InteriorColor string
	Owned                        bool // owned by the asking company
	InWarehouse                  bool
}

// Snapshot returns the immutable sale facts represented by this inventory
// response. It is called inside the retail reservation transaction.
func (v Vehicle) Snapshot() *model.VehicleSnapshot {
	return &model.VehicleSnapshot{
		VehicleID: v.ID, VIN: v.VIN, ModelID: v.ModelID,
		ModelSpecificationVersion: strconv.Itoa(v.ModelSpecificationVersion),
		ExteriorColor:             v.ExteriorColor, InteriorColor: v.InteriorColor,
	}
}

// Stock reserves and delivers vehicles (implemented by inventory). Calls with
// a ctx from Store.Bind join the retail transaction.
type Stock interface {
	Vehicle(ctx context.Context, companyID, id string) (*Vehicle, error)
	Reserve(ctx context.Context, companyID, dealID, vehicleID string) error
	Release(ctx context.Context, dealID, reason string) error
	// Deliver hands the vehicle to the retail customer: it leaves stock.
	Deliver(ctx context.Context, dealID, vehicleID, actorID string, at time.Time) error
}

// ConsignmentSale is the checked, live commerce authority for one VIN. Retail
// obtains it under its bound transaction and passes it to trusted stock calls.
type ConsignmentSale struct {
	UnitID, AgreementID, OrderID, OrderLineID, VehicleID, VIN string
	SupplierCompanyID, DealerCompanyID, HandoverFactID string
	Revision int64
}

// Consignment joins commerce authority and inventory's physical sale through
// application adapters. A nil authority is only valid for a truly absent live
// consignment VIN; every denial and read error must propagate.
type Consignment interface {
	CheckConsignmentSale(context.Context, string, string) (*ConsignmentSale, error)
	ReserveConsignmentSale(context.Context, string, string, string, string) (*ConsignmentSale, error)
	ConsignmentSaleForDeal(context.Context, string, string, string) (*ConsignmentSale, error)
	CancelConsignmentSale(context.Context, string, string, string, string) error
	CompleteConsignmentSale(context.Context, string, string, string, string, string, time.Time) error
	HoldConsignmentSale(context.Context, *ConsignmentSale, string, string, time.Time) error
	DeliverConsignmentSale(context.Context, *ConsignmentSale, string, string, time.Time) (string, error)
}

// Files checks and shares uploaded files (implemented by documents).
type Files interface {
	Share(ctx context.Context, ownerCompanyID, fileID, withCompanyID, resourceType, resourceID string) error
	BindPassport(ctx context.Context, ownerCompanyID, fileID, resourceType, resourceID string) error
}

// Insurance tells whether the insurer approved the deal's application
// (implemented by the insurance module).
type Insurance interface {
	Approved(ctx context.Context, companyID, dealID string) (bool, error)
}
