// Package handler holds commerce's Echo routes, request/response DTOs and
// mappers. It imports service, model and internal/pkg only; it must never
// import repository or gorm.
package handler

import (
	"time"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/commerce/service"
	"justixauto/internal/pkg/httpx"
)

// consignmentRetailCandidateDTO intentionally omits all agreement and
// financial fields; inventory owns subsequent vehicle-detail hydration.
type consignmentRetailCandidateDTO struct {
	VehicleID string `json:"vehicleId"`
}

func toConsignmentRetailCandidate(v service.RetailCandidate) consignmentRetailCandidateDTO {
	return consignmentRetailCandidateDTO{VehicleID: v.VehicleID}
}

type consignmentUnitDTO struct {
	ID                  string     `json:"id"`
	OrderLineID         string     `json:"orderLineId"`
	VehicleID           string     `json:"vehicleId"`
	VIN                 string     `json:"vin"`
	Status              string     `json:"status"`
	Revision            string     `json:"revision"`
	HandedOverAt        *time.Time `json:"handedOverAt,omitempty"`
	CurrentRetailDealID *string    `json:"currentRetailDealId,omitempty"`
	PendingReturnID     *string    `json:"pendingReturnId,omitempty"`
	SoldAt              *time.Time `json:"soldAt,omitempty"`
	ReturnedAt          *time.Time `json:"returnedAt,omitempty"`
}
type consignmentDTO struct {
	ID                   string               `json:"id"`
	OrderID              string               `json:"orderId"`
	ReservationID        string               `json:"reservationId"`
	SupplierCompanyID    string               `json:"supplierCompanyId"`
	DealerCompanyID      string               `json:"dealerCompanyId"`
	BookingDigest        string               `json:"bookingDigest"`
	ContinuationDigest   string               `json:"continuationDigest"`
	ContinuationRevision string               `json:"continuationRevision"`
	Revision             string               `json:"revision"`
	Units                []consignmentUnitDTO `json:"units"`
}

func toConsignment(v *service.ConsignmentView) consignmentDTO {
	a := v.Agreement
	out := consignmentDTO{
		ID: a.ID, OrderID: a.OrderID, ReservationID: a.ReservationID,
		SupplierCompanyID: a.SupplierCompanyID, DealerCompanyID: a.DealerCompanyID,
		BookingDigest: a.BookingDigest, ContinuationDigest: a.ContinuationDigest,
		ContinuationRevision: httpx.Revision(a.ContinuationRevision), Revision: httpx.Revision(a.Version),
		Units: make([]consignmentUnitDTO, 0, len(v.Units)),
	}
	for _, u := range v.Units {
		out.Units = append(out.Units, consignmentUnitDTO{
			ID: u.ID, OrderLineID: u.OrderLineID,
			VehicleID: u.VehicleID, VIN: u.VIN, Status: string(u.Status), Revision: httpx.Revision(u.Version),
			HandedOverAt: u.HandedOverAt, CurrentRetailDealID: u.CurrentRetailDealID, PendingReturnID: u.PendingReturnID,
			SoldAt: u.SoldAt, ReturnedAt: u.ReturnedAt,
		})
	}
	return out
}

type counterpartyDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Country string `json:"country"`
}

func party(c service.Company) counterpartyDTO {
	return counterpartyDTO{ID: c.ID, Name: c.Name, Country: c.Country}
}

func mapSlice[T, D any](items []T, f func(*T) D) []D {
	out := make([]D, len(items))
	for i := range items {
		out[i] = f(&items[i])
	}
	return out
}

func paging(c *echo.Context) (int, int, error) {
	limit, err := httpx.IntQuery(c, "limit")
	if err != nil {
		return 0, 0, err
	}
	offset, err := httpx.IntQuery(c, "offset")
	return limit, offset, err
}
