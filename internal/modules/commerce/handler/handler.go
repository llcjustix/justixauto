package handler

import (
	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/auth"
)

// Handler mounts commerce's Echo routes.
type Handler struct {
	partnerships        *service.Partnership
	offers              *service.Offer
	deals               *service.Deal
	fulfilment          *service.Fulfilment
	invoices            *service.Invoice
	contracts           *service.Contract
	supply              *service.Supply
	reservations        *service.Reservation
	reservationPayments *service.ReservationPayment
	reservationStock    *service.ReservationStockService
	consignments        *service.Consignment
	returns             *service.ConsignmentReturn
}

// New builds the commerce handler.
func New(partnerships *service.Partnership, offers *service.Offer, deals *service.Deal, fulfilment *service.Fulfilment, invoices *service.Invoice, contracts *service.Contract, supply *service.Supply, reservations *service.Reservation, reservationPayments *service.ReservationPayment, reservationStock *service.ReservationStockService, consignments *service.Consignment, returns *service.ConsignmentReturn) *Handler {
	return &Handler{partnerships: partnerships, offers: offers, deals: deals, fulfilment: fulfilment, invoices: invoices, contracts: contracts, supply: supply, reservations: reservations, reservationPayments: reservationPayments, reservationStock: reservationStock, consignments: consignments, returns: returns}
}

func (h *Handler) Routes(g *echo.Group) {
	c := g.Group("", auth.RequireCompany())
	c.GET("/partnerships", h.listPartnerships, auth.Require(model.PermRead))
	c.GET("/partnerships/:id", h.getPartnership, auth.Require(model.PermRead))
	c.POST("/partnerships", h.requestPartnership, auth.Require(model.PermPartnershipsManage))
	c.POST("/partnerships/:id/:action", h.decidePartnership, auth.Require(model.PermPartnershipsManage))

	c.GET("/offers", h.listOffers, auth.Require(model.PermRead))
	c.GET("/offers/:id", h.getOffer, auth.Require(model.PermRead))
	c.POST("/offers", h.createOffer, auth.Require(model.PermOffersManage))
	c.POST("/offers/:id/versions", h.addOfferVersion, auth.Require(model.PermOffersManage))
	c.POST("/offers/:id/publish", h.publishOffer, auth.Require(model.PermOffersManage))
	c.POST("/offers/:id/withdraw", h.withdrawOffer, auth.Require(model.PermOffersManage))

	c.GET("/supply-lots", h.listSupply, auth.Require(model.PermRead))
	c.GET("/reservation-stock", h.listReservationStock, auth.Require(model.PermRead))
	c.GET("/supply-lots/:id", h.getSupply, auth.Require(model.PermRead))
	c.POST("/supply-lots", h.createSupply, auth.Require(model.PermTrade))
	c.PUT("/supply-lots/:id", h.updateSupply, auth.Require(model.PermTrade))
	c.GET("/reservations", h.listReservations, auth.Require(model.PermRead))
	c.GET("/reservations/:id", h.getReservation, auth.Require(model.PermRead))
	c.POST("/reservations", h.requestReservation, auth.Require(model.PermTrade))
	c.POST("/reservations/:id/supplier-confirmations", h.confirmReservation, auth.Require(model.PermTrade))
	c.POST("/reservations/:id/acceptances", h.acceptReservation, auth.Require(model.PermTrade))
	c.POST("/reservations/:id/extensions", h.extendReservation, auth.Require(model.PermTrade))
	c.POST("/reservations/:id/cancellations", h.cancelReservation, auth.Require(model.PermTrade))
	c.POST("/reservations/:id/continuation-proposals", h.proposeReservationContinuation, auth.Require(model.PermTrade))
	c.POST("/reservations/:id/continuation-acceptances", h.acceptReservationContinuation, auth.Require(model.PermTrade))
	c.POST("/reservations/:id/conversions", h.convertReservation, auth.Require(model.PermTrade))
	c.GET("/reservations/:id/allocations", h.reservationAllocations, auth.Require(model.PermRead))
	c.POST("/reservations/:id/allocations", h.allocateReservation, auth.Require(model.PermTrade))
	c.GET("/reservations/:id/accounting", h.reservationAccounting, auth.Require(model.PermRead))
	c.POST("/reservations/:id/payment-evidence", h.submitReservationPayment, auth.Require(model.PermTrade))
	c.POST("/reservation-payment-evidence/:id/accept", h.decideReservationPayment(true), auth.Require(model.PermPaymentsAccept))
	c.POST("/reservation-payment-evidence/:id/reject", h.decideReservationPayment(false), auth.Require(model.PermPaymentsAccept))
	c.POST("/reservations/:id/refund-requests", h.requestReservationRefund, auth.Require(model.PermTrade))
	c.POST("/reservations/:id/refund-confirmations", h.confirmReservationRefund, auth.Require(model.PermPaymentsAccept))
	c.GET("/consignments/orders/:id", h.getConsignment, auth.Require(model.PermRead))
	c.POST("/consignments/vehicles/:vehicleId/return-requests", h.requestConsignmentReturn, auth.Require(model.PermTrade))
	c.GET("/consignments/return-requests", h.listConsignmentReturns, auth.Require(model.PermRead))
	c.GET("/consignments/return-requests/:id", h.getConsignmentReturn, auth.Require(model.PermRead))
	c.POST("/consignments/return-requests/:id/confirm", h.decideConsignmentReturn(true), auth.Require(model.PermTrade))
	c.POST("/consignments/return-requests/:id/reject", h.decideConsignmentReturn(false), auth.Require(model.PermTrade))
	c.GET("/consignments/retail-candidates", h.listConsignmentRetailCandidates, auth.Require("retail.read"))

	h.dealRoutes(c)
	h.invoiceRoutes(c)
	h.contractRoutes(c)
}
