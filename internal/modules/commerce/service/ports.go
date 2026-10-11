package service

import (
	"context"
	"time"

	"justixauto/internal/modules/commerce/model"
)

// Store gives services the commerce repositories and one-transaction runs.
type Store interface {
	Partnerships() PartnershipRepository
	Offers() OfferRepository
	Deals() DealRepository
	Fulfilment() FulfilmentRepository
	Invoices() InvoiceRepository
	Contracts() ContractRepository
	Supply() SupplyRepository
	Reservations() ReservationRepository
	ReservationPayments() ReservationPaymentRepository
	Consignments() ConsignmentRepository
	ConsignmentReturns() ConsignmentReturnRepository
	Events() EventRepository
	InTx(ctx context.Context, fn func(Store) error) error
	// Bind returns ctx carrying this store's transaction, so other modules
	// called through ports commit or roll back together with it.
	Bind(ctx context.Context) context.Context
}

type ConsignmentReturnRepository interface {
	Create(ctx context.Context, row *model.ConsignmentReturn) error
	Get(ctx context.Context, id string) (*model.ConsignmentReturn, error)
	List(ctx context.Context, companyID, orderID string, limit, offset int) ([]model.ConsignmentReturn, error)
	Lock(ctx context.Context, id string) (*model.ConsignmentReturn, error)
	Decide(ctx context.Context, row *model.ConsignmentReturn, expected int64) error
}

// ConsignmentRepository is always used through Store.InTx for mutations.
// LockUnitByVehicle holds the same live VIN row used by retail and returns.
type ConsignmentRepository interface {
	CreateAgreement(ctx context.Context, agreement *model.ConsignmentAgreement) error
	AgreementByOrder(ctx context.Context, orderID string) (*model.ConsignmentAgreement, error)
	Agreement(ctx context.Context, id string) (*model.ConsignmentAgreement, error)
	CreateUnit(ctx context.Context, unit *model.ConsignmentUnit) error
	Unit(ctx context.Context, agreementID, vehicleID string) (*model.ConsignmentUnit, error)
	LockUnitByVehicle(ctx context.Context, vehicleID string) (*model.ConsignmentUnit, error)
	UpdateUnit(ctx context.Context, unit *model.ConsignmentUnit, expected int64) error
	Units(ctx context.Context, agreementID string) ([]model.ConsignmentUnit, error)
	RetailCandidates(ctx context.Context, dealerID string, limit, offset int) ([]model.ConsignmentUnit, error)
}

type ReservationRepository interface {
	Create(ctx context.Context, r *model.Reservation, lines []model.ReservationLine) error
	Get(ctx context.Context, companyID, id string) (*model.Reservation, []model.ReservationLine, error)
	List(ctx context.Context, companyID string, limit, offset int) ([]model.Reservation, error)
	Lines(ctx context.Context, reservationID string) ([]model.ReservationLine, error)
	ForUpdate(ctx context.Context, id string) (*model.Reservation, []model.ReservationLine, error)
	ByOrderForUpdate(ctx context.Context, orderID string) (*model.Reservation, []model.ReservationLine, error)
	DueCandidates(ctx context.Context, now time.Time, afterAt *time.Time, afterID string, limit int) ([]model.ReservationDueCandidate, error)
	Update(ctx context.Context, r *model.Reservation, expected int64) error
}

// ReservationPaymentRepository is used only after the source reservation row
// has been locked in the caller's commerce transaction.
type ReservationPaymentRepository interface {
	Evidence(ctx context.Context, reservationID string) ([]model.ReservationEvidence, error)
	GetEvidence(ctx context.Context, id string) (*model.ReservationEvidence, error)
	AddEvidence(ctx context.Context, e *model.ReservationEvidence) error
	DecideEvidence(ctx context.Context, e *model.ReservationEvidence, expected int64) error
	Refund(ctx context.Context, reservationID string) (*model.ReservationRefund, error)
	CreateRefund(ctx context.Context, refund *model.ReservationRefund) error
	ConfirmRefund(ctx context.Context, refund *model.ReservationRefund) error
	CreditForReservation(ctx context.Context, reservationID string) (*model.ReservationCredit, error)
	CreditForOrder(ctx context.Context, orderID string) (*model.ReservationCredit, error)
	CreateCredit(ctx context.Context, credit *model.ReservationCredit) error
	AttachInvoice(ctx context.Context, reservationID, invoiceID string, at time.Time) error
}

// ContractRepository persists company-private templates and order contract
// versions.  The service, rather than a handler, supplies every scope value.
type ContractRepository interface {
	CreateTemplate(ctx context.Context, t *model.ContractTemplate) error
	Template(ctx context.Context, companyID, id string) (*model.ContractTemplate, error)
	Templates(ctx context.Context, companyID string, limit, offset int) ([]model.ContractTemplate, error)
	UpdateTemplate(ctx context.Context, t *model.ContractTemplate, expected int64) error
	Create(ctx context.Context, c *model.ContractVersion) error
	Contract(ctx context.Context, viewerCompanyID, id string) (*model.ContractVersion, error)
	ForOrder(ctx context.Context, viewerCompanyID, orderID string, limit, offset int) ([]model.ContractVersion, error)
	Update(ctx context.Context, c *model.ContractVersion, expected int64) error
	HasBaseSigned(ctx context.Context, orderID string) (bool, error)
}

type PartnershipRepository interface {
	Create(ctx context.Context, p *model.Partnership) error
	// Get returns a partnership the company takes part in, else ErrNotFound.
	Get(ctx context.Context, companyID, id string) (*model.Partnership, error)
	List(ctx context.Context, f model.PartnershipFilter) ([]model.Partnership, error)
	Update(ctx context.Context, p *model.Partnership, expected int64) error
	// ActiveBetween reports whether the two companies have an active partnership.
	ActiveBetween(ctx context.Context, a, b string) (bool, error)
}

type EventRepository interface {
	Append(ctx context.Context, e *model.Event) error
	ForResource(ctx context.Context, resourceType, id string) ([]model.Event, error)
}

type OfferRepository interface {
	Create(ctx context.Context, o *model.Offer, v *model.OfferVersion) error
	// GetOwn returns the supplier's own offer, else ErrNotFound.
	GetOwn(ctx context.Context, supplierID, id string) (*model.Offer, error)
	// GetVisible returns a published offer the buyer may see, else ErrNotFound.
	GetVisible(ctx context.Context, buyerID, id string) (*model.Offer, error)
	// VisibleByPublishedVersion finds the visible offer whose published version is versionID.
	VisibleByPublishedVersion(ctx context.Context, buyerID, versionID string) (*model.Offer, error)
	ListOwn(ctx context.Context, supplierID string, limit, offset int) ([]model.Offer, error)
	ListVisible(ctx context.Context, buyerID string, limit, offset int) ([]model.Offer, error)
	Versions(ctx context.Context, offerID string) ([]model.OfferVersion, error)
	Version(ctx context.Context, offerID, versionID string) (*model.OfferVersion, error)
	AddVersion(ctx context.Context, v *model.OfferVersion) error
	Update(ctx context.Context, o *model.Offer, expected int64) error
	MarkPublished(ctx context.Context, versionID string, at time.Time) error
}

type DealRepository interface {
	CreateRFQ(ctx context.Context, r *model.RFQ) error
	// RFQ returns an RFQ visible to the company: the buyer always, the
	// supplier once it was sent.
	RFQ(ctx context.Context, companyID, id string) (*model.RFQ, error)
	RFQs(ctx context.Context, companyID string, limit, offset int) ([]model.RFQ, error)
	UpdateRFQ(ctx context.Context, r *model.RFQ, expected int64) error
	AddQuotation(ctx context.Context, q *model.Quotation) error
	Quotations(ctx context.Context, rfqID string) ([]model.Quotation, error)

	CreateOrder(ctx context.Context, o *model.Order) error
	// Order returns an order the company is a party of.
	Order(ctx context.Context, companyID, id string) (*model.Order, error)
	Orders(ctx context.Context, companyID string, limit, offset int) ([]model.Order, error)
	UpdateOrder(ctx context.Context, o *model.Order, expected int64) error
	AddAddendum(ctx context.Context, a *model.Addendum) error
	Addenda(ctx context.Context, orderID string) ([]model.Addendum, error)
	DecideAddendum(ctx context.Context, a *model.Addendum) error
}

type FulfilmentRepository interface {
	Allocations(ctx context.Context, orderID string) ([]model.Allocation, error)
	AddAllocations(ctx context.Context, as []model.Allocation) error
	SetAllocationStatus(ctx context.Context, orderID string, vehicleIDs []string, status string, shipmentID *string) error
	CreateShipment(ctx context.Context, s *model.Shipment) error
	AddShipmentLines(ctx context.Context, ls []model.ShipmentLine) error
	// ShipmentLines lists what was shipped by quantity for the order.
	ShipmentLines(ctx context.Context, orderID string) ([]model.ShipmentLine, error)
	Shipment(ctx context.Context, id string) (*model.Shipment, error)
	Shipments(ctx context.Context, orderID string) ([]model.Shipment, error)
	UpdateShipment(ctx context.Context, s *model.Shipment, expected int64) error
	AddMilestone(ctx context.Context, m *model.Milestone) error
	Milestones(ctx context.Context, shipmentID string) ([]model.Milestone, error)
}

type InvoiceRepository interface {
	Create(ctx context.Context, i *model.Invoice) error
	// Get returns an invoice the company is a party of.
	Get(ctx context.Context, companyID, id string) (*model.Invoice, error)
	ForOrder(ctx context.Context, orderID string) ([]model.Invoice, error)
	Update(ctx context.Context, i *model.Invoice, expected int64) error
	AddEvidence(ctx context.Context, e *model.Evidence) error
	Evidence(ctx context.Context, invoiceID string) ([]model.Evidence, error)
	GetEvidence(ctx context.Context, id string) (*model.Evidence, error)
	UpdateEvidence(ctx context.Context, e *model.Evidence, expected int64) error
}

// SupplyRepository owns the future-capacity ledger. Claim operations are
// exposed now for P17; callers must operate under Store.InTx and lock the lot
// before changing claims.
type SupplyRepository interface {
	Create(ctx context.Context, lot *model.SupplyLot) error
	Lot(ctx context.Context, id string) (*model.SupplyLot, error)
	LockLot(ctx context.Context, id string) (*model.SupplyLot, error)
	ListSupplier(ctx context.Context, supplierID string, limit, offset int) ([]model.SupplyLot, error)
	ListVisible(ctx context.Context, dealerID string, limit, offset int) ([]model.SupplyLot, error)
	Update(ctx context.Context, lot *model.SupplyLot, expected int64) error
	Capacity(ctx context.Context, lotID string) (model.SupplyCapacity, error)
	ClaimsForUpdate(ctx context.Context, lotID string) ([]model.SupplyClaim, error)
	CreateClaims(ctx context.Context, claims []model.SupplyClaim) error
	UpdateClaim(ctx context.Context, claim *model.SupplyClaim) error
	ClaimAllocations(ctx context.Context, reservationID string) ([]model.SupplyClaimAllocation, error)
	AppendClaimAllocation(ctx context.Context, association model.SupplyClaimAllocation, replaceID string) error
	RetireClaimAllocation(ctx context.Context, claimID, associationID string) error
}

// Company is what commerce needs to know about another company.
type Company struct {
	ID, Name, Kind, Country string
	UserType                *string
	Verification            string
	Active                  bool
}

// Files shares uploaded files with another company (implemented by documents).
type Files interface {
	Share(ctx context.Context, ownerCompanyID, fileID, withCompanyID, resourceType, resourceID string) error
}

// ContractRequisites is the private company projection frozen into an order
// contract. It is intentionally separate from Directory's public profile.
type ContractRequisites struct {
	ID, Name, LegalName, RegistrationNumber, TaxID, BankRequisites, Address, Phone, Email string
}

// ContractParties resolves private requisites for the actual parties of an
// order. Implementations must authorize viewerCompanyID against orderID before
// asking identity for either private projection.
type ContractParties interface {
	ForOrder(ctx context.Context, viewerCompanyID, orderID string) (buyer, supplier ContractRequisites, err error)
}

// Directory looks up companies (implemented by the identity module).
// It returns apperr.ErrNotFound for unknown IDs.
type Directory interface {
	Company(ctx context.Context, id string) (*Company, error)
}

// Model is what commerce needs to know about a catalogue model.
type Model struct {
	ID, Name                    string
	CurrentSpecificationVersion string
	Specifications              []ModelSpecification
}

// ModelSpecification exposes the palette of one immutable catalog version.
type ModelSpecification struct {
	Version                        string
	ExteriorColors, InteriorColors []string
}

// Catalog looks up vehicle models (implemented by the inventory module).
// It returns apperr.ErrNotFound for unknown IDs.
type Catalog interface {
	Model(ctx context.Context, id string) (*Model, error)
}

// StockVehicle is what commerce needs to know about a concrete vehicle.
type StockVehicle struct {
	ID, VIN, ModelID                                        string
	ModelSpecificationVersion, ExteriorColor, InteriorColor string
}

// ReceiptBatch is the inventory-owned current state needed to render a
// quantity shipment on its originating order.
type ReceiptBatch struct {
	ID, WarehouseID, ModelID                                string
	ModelSpecificationVersion, ExteriorColor, InteriorColor string
	ConfirmedQuantity                                       int
	IdentifiedCount, UnidentifiedCount                      int
	Revision                                                int64
}

// Delivery carries the selected incoming facts, independently of order history.
type Delivery struct {
	ToCompanyID, ToWarehouseID, ModelID                     string
	ModelSpecificationVersion, ExteriorColor, InteriorColor string
	Quantity                                                int
	VINs                                                    []string
	ActorUserID                                             string
	At                                                      time.Time
}

// CustodyMove is the trusted, bound physical consignment movement. Commerce
// supplies the agreement and exact order holder; inventory verifies the stock.
type CustodyMove struct {
	OwnerCompanyID, SourceCustodianCompanyID        string
	DestinationCompanyID, DestinationWarehouseID    string
	OrderID, AgreementID, OperationRef, ActorUserID string
	VehicleID                                       string
	At                                              time.Time
}

// ReturnCustodyMove is the trusted one-VIN return; no order holder is supplied.
type ReturnCustodyMove struct {
	OwnerCompanyID, SourceCustodianCompanyID, DestinationWarehouseID string
	AgreementID, RequestID, ActorUserID, VehicleID string
	At time.Time
}

// Stock reserves and hands over vehicles (implemented by inventory). Calls
// made with a ctx from Store.Bind join the commerce transaction.
type Stock interface {
	Vehicle(ctx context.Context, companyID, id string) (*StockVehicle, error)
	// OwnsWarehouse returns apperr.ErrNotFound unless id is a warehouse of companyID.
	OwnsWarehouse(ctx context.Context, companyID, id string) error
	Reserve(ctx context.Context, companyID, orderID string, vehicleIDs []string) error
	Release(ctx context.Context, orderID string, vehicleIDs []string, reason string) error
	Transfer(ctx context.Context, orderID string, vehicleIDs []string, toCompanyID, toWarehouseID, actorID string, at time.Time) error
	Custody(ctx context.Context, move CustodyMove) (string, error)
	ReturnCustody(ctx context.Context, move ReturnCustodyMove) (string, error)
	// Deliver puts quantity vehicles of a model into the receiving company's
	// warehouse as a receipt batch (vins may cover only part of them) and
	// returns the batch ID.
	Deliver(ctx context.Context, delivery Delivery) (string, error)
	// ReceiptBatches reads the current state of selected receipt batches owned
	// by companyID. It must return an error when any requested batch is absent.
	ReceiptBatches(ctx context.Context, companyID string, ids []string) ([]ReceiptBatch, error)
}

type ReservationStock interface {
	Stock
	ReserveReservation(ctx context.Context, companyID, reservationID string, vehicleIDs []string) error
	ReleaseReservation(ctx context.Context, reservationID string, vehicleIDs []string, reason string) error
	ReassignReservationToOrder(ctx context.Context, reservationID, orderID string, vehicleIDs []string) error
}
