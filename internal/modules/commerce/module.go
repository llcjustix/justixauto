// Package commerce owns B2B trade between companies: partnerships, offers,
// quotations, orders, shipments and invoices. Other companies are known only
// through the Directory port.
//
// This package is the only one other code imports; internal/modules/commerce
// splits into model (GORM models, enums, value types), repository (GORM
// persistence), service (business rules) and handler (Echo routes) layers,
// wired together here.
package commerce

import (
	"context"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"justixauto/internal/modules/commerce/handler"
	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/modules/commerce/repository"
	"justixauto/internal/modules/commerce/service"
)

const (
	PermRead               = model.PermRead
	PermPartnershipsManage = model.PermPartnershipsManage
	PermOffersManage       = model.PermOffersManage
	PermTrade              = model.PermTrade
	PermInvoicesManage     = model.PermInvoicesManage
	PermPaymentsAccept     = model.PermPaymentsAccept
)

// Permissions are registered in the identity catalog at startup.
var Permissions = model.Permissions

// Company is what commerce needs to know about another company.
type Company = service.Company

// Files shares uploaded files with another company (implemented by documents).
type Files = service.Files

// Directory looks up companies (implemented by the identity module).
type Directory = service.Directory

// Model is what commerce needs to know about a catalogue model.
type Model = service.Model

type ModelSpecification = service.ModelSpecification

// Catalog looks up vehicle models (implemented by the inventory module).
type Catalog = service.Catalog

// StockVehicle is what commerce needs to know about a concrete vehicle.
type StockVehicle = service.StockVehicle

type (
	Delivery                   = service.Delivery
	CustodyMove                = service.CustodyMove
	ReturnCustodyMove          = service.ReturnCustodyMove
	ConsignmentRetailCandidate = service.RetailCandidate
)

// ReceiptBatch is the receipt state commerce reads from inventory.
type ReceiptBatch = service.ReceiptBatch

// Stock reserves and hands over vehicles (implemented by inventory).
type (
	Stock            = service.Stock
	ReservationStock = service.ReservationStock
)

// ContractRequisites is the private identity projection frozen into a B2B
// contract snapshot.
type ContractRequisites = service.ContractRequisites

// ContractParties resolves the two actual parties' private requisites after
// validating the order viewer.
type ContractParties = service.ContractParties

// ActualOrderParties is the narrow actual-order authorization source used by
// the app adapter before it asks identity for private requisites.
type ActualOrderParties = service.ActualOrderParties

// Order is exposed only as the actual-order party result consumed by the app
// composition adapter.
type (
	Order                    = model.Order
	ConsignmentUnit          = model.ConsignmentUnit
	ConsignmentSaleAuthority = service.SaleAuthority
)

// ReservationExpiryPass describes the bounded work completed by one automatic
// reservation-expiry scan.
type ReservationExpiryPass = service.ReservationExpiryPass

// ContractPartiesFactory creates the app-owned identity adapter only after an
// initialized actual-order lookup exists.
type ContractPartiesFactory func(*ActualOrderParties) ContractParties

// storeAdapter bridges repository.Store (which cannot import service, since
// repository implements service's interfaces structurally) to service.Store,
// whose InTx is self-referential and so needs the exact service.Store type.
type storeAdapter struct{ r *repository.Store }

func (a storeAdapter) Partnerships() service.PartnershipRepository { return a.r.Partnerships() }
func (a storeAdapter) Offers() service.OfferRepository             { return a.r.Offers() }
func (a storeAdapter) Deals() service.DealRepository               { return a.r.Deals() }
func (a storeAdapter) Fulfilment() service.FulfilmentRepository    { return a.r.Fulfilment() }
func (a storeAdapter) Invoices() service.InvoiceRepository         { return a.r.Invoices() }
func (a storeAdapter) Contracts() service.ContractRepository       { return a.r.Contracts() }
func (a storeAdapter) Supply() service.SupplyRepository            { return a.r.Supply() }
func (a storeAdapter) Reservations() service.ReservationRepository { return a.r.Reservations() }
func (a storeAdapter) ReservationPayments() service.ReservationPaymentRepository {
	return a.r.ReservationPayments()
}
func (a storeAdapter) Consignments() service.ConsignmentRepository { return a.r.Consignments() }
func (a storeAdapter) ConsignmentReturns() service.ConsignmentReturnRepository {
	return a.r.ConsignmentReturns()
}
func (a storeAdapter) Events() service.EventRepository          { return a.r.Events() }
func (a storeAdapter) Bind(ctx context.Context) context.Context { return a.r.Bind(ctx) }

func (a storeAdapter) InTx(ctx context.Context, fn func(service.Store) error) error {
	return a.r.InTx(ctx, func(rs *repository.Store) error { return fn(storeAdapter{rs}) })
}

var _ service.Store = storeAdapter{}

// Module wires commerce's layers and exposes the ports other modules use.
type Module struct {
	handler      *handler.Handler
	Contracts    *service.Contract
	reservations *service.Reservation
	consignments *service.Consignment
}

// New builds the commerce module.
func New(db *gorm.DB, now func() time.Time, directory Directory, catalog Catalog, stock ReservationStock, discovery ReservationStockDiscovery, files Files, contractParties ContractPartiesFactory) *Module {
	if now == nil {
		now = time.Now
	}
	d := service.NewDeps(storeAdapter{repository.NewStore(db)}, directory, catalog, stock, files, now)
	offers := service.NewOffer(d)
	actualOrders := service.NewActualOrderParties(d)
	var parties ContractParties
	if contractParties != nil {
		parties = contractParties(actualOrders)
	}
	contracts := service.NewContract(d, parties)
	payment := service.NewReservationPayment(d)
	reservations := service.NewReservation(d, stock)
	consignments := service.NewConsignment(d)
	h := handler.New(service.NewPartnership(d), offers, service.NewDeal(d, offers), service.NewFulfilment(d), service.NewInvoice(d), contracts, service.NewSupply(d), reservations, payment, service.NewReservationStockService(d, discovery), consignments, service.NewConsignmentReturn(d))
	return &Module{handler: h, Contracts: contracts, reservations: reservations, consignments: consignments}
}

// Register mounts the commerce routes under /api/v1/commerce.
func (m *Module) Register(api *echo.Group) { m.handler.Routes(api.Group("/commerce")) }

// ExpireDue runs one bounded, idempotent reservation-expiry pass. Runtime
// composition owns its cadence and replica exclusion.
func (m *Module) ExpireDue(ctx context.Context, limit int, cursor string) (ReservationExpiryPass, error) {
	return m.reservations.ExpireDue(ctx, limit, cursor)
}

// ConsignmentForVehicle is a discovery view; retail/return mutations recheck
// the same unit inside their own bound transaction.
func (m *Module) ConsignmentForVehicle(ctx context.Context, dealerID, vehicleID string) (*ConsignmentUnit, error) {
	return m.consignments.ActiveForVehicle(ctx, dealerID, vehicleID)
}

// CheckConsignmentSale is a trusted retail check. Only a truly absent live VIN
// returns (nil,nil), permitting the ordinary owner path. A present but
// unauthorized unit returns an error. Mutation callers pass their Store.Bind
// context and retain the row lock through the outer transaction.
func (m *Module) CheckConsignmentSale(ctx context.Context, sellerID, vehicleID string) (*ConsignmentSaleAuthority, error) {
	return m.consignments.CheckSale(ctx, sellerID, vehicleID)
}

func (m *Module) ReserveConsignmentSale(ctx context.Context, sellerID, vehicleID, dealID, actorID string) (*ConsignmentSaleAuthority, error) {
	return m.consignments.ReserveSale(ctx, sellerID, vehicleID, dealID, actorID)
}

func (m *Module) ConsignmentSaleForDeal(ctx context.Context, sellerID, vehicleID, dealID string) (*ConsignmentSaleAuthority, error) {
	return m.consignments.SaleForDeal(ctx, sellerID, vehicleID, dealID)
}

func (m *Module) CancelConsignmentSale(ctx context.Context, sellerID, vehicleID, dealID, actorID string) error {
	return m.consignments.CancelSale(ctx, sellerID, vehicleID, dealID, actorID)
}

func (m *Module) CompleteConsignmentSale(ctx context.Context, sellerID, vehicleID, dealID, deliveryRef, actorID string, at time.Time) error {
	return m.consignments.CompleteSale(ctx, sellerID, vehicleID, dealID, deliveryRef, actorID, at)
}
