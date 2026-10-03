// Package handler holds retail's Echo routes, request/response DTOs and
// mappers. It imports service, model and internal/pkg only; it must never
// import repository or gorm.
package handler

import (
	"encoding/json"
	"math/big"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/modules/retail/service"
	"justixauto/internal/pkg/httpx"
	"justixauto/internal/pkg/money"
)

type customerDTO struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	Phone       string    `json:"phone"`
	Revision    string    `json:"revision"`
	CreatedAt   time.Time `json:"createdAt"`
}

func toCustomer(c *model.Customer) customerDTO {
	return customerDTO{ID: c.ID, DisplayName: c.DisplayName, Phone: c.Phone, Revision: httpx.Revision(c.Version), CreatedAt: c.CreatedAt}
}

type contactDTO struct {
	Channel    string    `json:"channel"`
	Note       string    `json:"note"`
	ActorID    string    `json:"actorId"`
	OccurredAt time.Time `json:"occurredAt"`
}

type historyDTO struct {
	Type       string          `json:"type"`
	ActorID    string          `json:"actorId"`
	OccurredAt time.Time       `json:"occurredAt"`
	Reason     string          `json:"reason"`
	Details    json.RawMessage `json:"details"`
}

func toHistory(es []model.Event) []historyDTO {
	out := []historyDTO{}
	for _, e := range es {
		out = append(out, historyDTO{Type: e.EventType, ActorID: e.ActorUserID, OccurredAt: e.OccurredAt, Reason: e.Reason, Details: e.Details})
	}
	return out
}

type leadDTO struct {
	ID             string       `json:"id"`
	CustomerID     string       `json:"customerId"`
	Customer       *customerDTO `json:"customer,omitempty"`
	BranchID       string       `json:"branchId"`
	Source         string       `json:"source"`
	Stage          string       `json:"stage"`
	AssignedUserID *string      `json:"assignedUserId"`
	LostReason     string       `json:"lostReason"`
	DealID         *string      `json:"dealId"`
	Contacts       []contactDTO `json:"contacts,omitempty"`
	History        []historyDTO `json:"history,omitempty"`
	Revision       string       `json:"revision"`
	UpdatedAt      time.Time    `json:"updatedAt"`
}

func toLead(l *model.Lead) leadDTO {
	return leadDTO{
		ID: l.ID, CustomerID: l.CustomerID, BranchID: l.BranchID, Source: l.Source, Stage: l.Stage,
		AssignedUserID: l.AssignedUserID, LostReason: l.LostReason, DealID: l.DealID, Revision: httpx.Revision(l.Version), UpdatedAt: l.UpdatedAt,
	}
}

func toLeadView(v *service.LeadView) leadDTO {
	d := toLead(&v.Lead)
	c := toCustomer(&v.Customer)
	d.Customer, d.Contacts, d.History = &c, []contactDTO{}, toHistory(v.History)
	for _, ct := range v.Contacts {
		d.Contacts = append(d.Contacts, contactDTO{Channel: ct.Channel, Note: ct.Note, ActorID: ct.ActorID, OccurredAt: ct.OccurredAt})
	}
	return d
}

type taskDTO struct {
	ID          string     `json:"id"`
	CustomerID  string     `json:"customerId"`
	LeadID      *string    `json:"leadId"`
	DealID      *string    `json:"dealId"`
	OwnerUserID string     `json:"ownerUserId"`
	DueAt       time.Time  `json:"dueAt"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	CompletedAt *time.Time `json:"completedAt"`
	Revision    string     `json:"revision"`
}

func toTask(t *model.Task) taskDTO {
	return taskDTO{
		ID: t.ID, CustomerID: t.CustomerID, LeadID: t.LeadID, DealID: t.DealID, OwnerUserID: t.OwnerUserID,
		DueAt: t.DueAt, Title: t.Title, Status: t.Status, CompletedAt: t.CompletedAt, Revision: httpx.Revision(t.Version),
	}
}

type listingDTO struct {
	ID          string      `json:"id"`
	VehicleID   string      `json:"vehicleId"`
	Text        string      `json:"text"`
	AskingPrice money.Money `json:"askingPrice"`
	Status      string      `json:"status"`
	Revision    string      `json:"revision"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

func toListing(l *model.Listing) listingDTO {
	return listingDTO{
		ID: l.ID, VehicleID: l.VehicleID, Text: l.Text, AskingPrice: l.Price(), Status: l.Status,
		Revision: httpx.Revision(l.Version), UpdatedAt: l.UpdatedAt,
	}
}

type evidenceDTO struct {
	ID                string      `json:"id"`
	PaymentGroupID    *string     `json:"paymentGroupId"`
	Amount            money.Money `json:"amount"`
	PaidOn            string      `json:"paidOn"`
	ExternalReference string      `json:"externalReference"`
	AttachmentIDs     []string    `json:"attachmentIds"`
	Status            string      `json:"status"`
	DecisionReason    string      `json:"decisionReason"`
	AllowedActions    []string    `json:"allowedActions"`
	Revision          string      `json:"revision"`
}

type invoiceDTO struct {
	ID                string                  `json:"id"`
	DealID            string                  `json:"dealId"`
	Purpose           string                  `json:"purpose"`
	Amount            money.Money             `json:"amount"`
	RecipientSnapshot string                  `json:"recipientSnapshot"`
	DueDate           *string                 `json:"dueDate"`
	InstallmentNumber *int                    `json:"installmentNumber"`
	Status            string                  `json:"status"`
	Paid              money.Money             `json:"paid"`
	Pending           money.Money             `json:"pending"`
	Outstanding       money.Money             `json:"outstanding"`
	Available         money.Money             `json:"available"`
	AllowedActions    []string                `json:"allowedActions"`
	Evidence          []evidenceDTO           `json:"paymentEvidence"`
	PaymentGroups     []installmentPaymentDTO `json:"paymentGroups"`
	Revision          string                  `json:"revision"`
}

func toInvoice(v *service.InvoiceView) invoiceDTO {
	i := v.Invoice
	d := invoiceDTO{
		ID: i.ID, DealID: i.DealID, Purpose: i.Purpose, Amount: money.Money{AmountMinor: i.AmountMinor, Currency: i.Currency},
		RecipientSnapshot: i.RecipientSnapshot, Status: i.Status, Paid: v.Paid, Pending: v.Pending, Outstanding: v.Outstanding, InstallmentNumber: i.InstallmentNumber, AllowedActions: append([]string{}, v.AllowedActions...),
		Evidence: []evidenceDTO{}, PaymentGroups: []installmentPaymentDTO{}, Revision: httpx.Revision(i.Version),
	}
	available := new(big.Int).Sub(moneyAmount(v.Outstanding.AmountMinor), moneyAmount(v.Pending.AmountMinor))
	if available.Sign() < 0 {
		available.SetInt64(0)
	}
	d.Available = money.Of(available, i.Currency)
	if i.DueDate != nil {
		s := i.DueDate.Format(time.DateOnly)
		d.DueDate = &s
	}
	for _, e := range v.Evidence {
		d.Evidence = append(d.Evidence, evidenceDTO{
			ID: e.ID, PaymentGroupID: e.PaymentGroupID, Amount: money.Money{AmountMinor: e.AmountMinor, Currency: e.Currency},
			PaidOn: e.PaidOn.Format(time.DateOnly), ExternalReference: e.ExternalReference, AttachmentIDs: ids(e.AttachmentIDs), Status: e.Status,
			DecisionReason: e.DecisionReason, Revision: httpx.Revision(e.Version), AllowedActions: append([]string{}, v.EvidenceActions[e.ID]...),
		})
	}
	for i := range v.PaymentGroups {
		d.PaymentGroups = append(d.PaymentGroups, toInstallmentPayment(&v.PaymentGroups[i]))
	}
	return d
}

type installmentPaymentAllocationDTO struct {
	InvoiceID  string      `json:"invoiceId"`
	EvidenceID string      `json:"evidenceId"`
	Number     int         `json:"number"`
	Amount     money.Money `json:"amount"`
}

type installmentPaymentDTO struct {
	ID                string                            `json:"id"`
	DealID            string                            `json:"dealId"`
	InstallmentPlanID string                            `json:"installmentPlanId"`
	ClaimedAmount     money.Money                       `json:"claimedAmount"`
	PaidOn            string                            `json:"paidOn"`
	ExternalReference string                            `json:"externalReference"`
	AttachmentIDs     []string                          `json:"attachmentIds"`
	Status            string                            `json:"status"`
	DecisionReason    string                            `json:"decisionReason"`
	Revision          string                            `json:"revision"`
	AllowedActions    []string                          `json:"allowedActions"`
	Allocations       []installmentPaymentAllocationDTO `json:"allocations"`
}

func toInstallmentPayment(v *service.InstallmentPaymentView) installmentPaymentDTO {
	p := v.Payment
	out := installmentPaymentDTO{ID: p.ID, DealID: p.DealID, InstallmentPlanID: p.InstallmentPlanID,
		ClaimedAmount: money.Money{AmountMinor: p.AmountMinor, Currency: p.Currency}, PaidOn: p.PaidOn.Format(time.DateOnly),
		ExternalReference: p.ExternalReference, AttachmentIDs: ids(p.AttachmentIDs), Status: p.Status,
		DecisionReason: p.DecisionReason, Revision: httpx.Revision(p.Version),
		AllowedActions: append([]string{}, v.AllowedActions...), Allocations: []installmentPaymentAllocationDTO{}}
	for _, a := range v.Allocations {
		out.Allocations = append(out.Allocations, installmentPaymentAllocationDTO{InvoiceID: a.InvoiceID, EvidenceID: a.EvidenceID, Number: a.Number, Amount: a.Amount})
	}
	return out
}

func moneyAmount(raw string) *big.Int {
	n, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return new(big.Int)
	}
	return n
}

type installmentPlanRowDTO struct {
	Number         int         `json:"number"`
	InvoiceID      string      `json:"invoiceId"`
	DueDate        string      `json:"dueDate"`
	Amount         money.Money `json:"amount"`
	Paid           money.Money `json:"paid"`
	Pending        money.Money `json:"pending"`
	Outstanding    money.Money `json:"outstanding"`
	Available      money.Money `json:"available"`
	AllowedActions []string    `json:"allowedActions"`
}

type installmentPlanDTO struct {
	ID                   string                  `json:"id"`
	State                string                  `json:"state"`
	ContractReference    string                  `json:"contractReference"`
	ContractSignedOn     string                  `json:"contractSignedOn"`
	ContractFileIDs      []string                `json:"contractFileIds"`
	ContractTotal        money.Money             `json:"contractTotal"`
	DownPaymentInvoiceID string                  `json:"downPaymentInvoiceId"`
	DownPayment          money.Money             `json:"downPayment"`
	ScheduledTotal       money.Money             `json:"scheduledTotal"`
	Paid                 money.Money             `json:"paid"`
	Pending              money.Money             `json:"pending"`
	Outstanding          money.Money             `json:"outstanding"`
	Available            money.Money             `json:"available"`
	SettlementState      string                  `json:"settlementState"`
	AllowedActions       []string                `json:"allowedActions"`
	Payments             []installmentPaymentDTO `json:"payments"`
	Rows                 []installmentPlanRowDTO `json:"rows"`
}

type installmentDraftRowDTO struct {
	Number  int         `json:"number"`
	DueDate string      `json:"dueDate"`
	Amount  money.Money `json:"amount"`
	Balance money.Money `json:"balance"`
}

type installmentDraftDTO struct {
	PolicyID       string                   `json:"policyId"`
	PolicyVersion  int                      `json:"policyVersion"`
	Price          money.Money              `json:"price"`
	DownPayment    money.Money              `json:"downPayment"`
	TermMonths     int                      `json:"termMonths"`
	FirstDueDate   string                   `json:"firstDueDate"`
	ScheduledTotal money.Money              `json:"scheduledTotal"`
	RegularPayment money.Money              `json:"regularPayment"`
	Rows           []installmentDraftRowDTO `json:"rows"`
}

func toInstallmentDraft(d *model.InstallmentDraft) *installmentDraftDTO {
	if d == nil {
		return nil
	}
	out := &installmentDraftDTO{PolicyID: d.PolicyID, PolicyVersion: d.PolicyVersion, Price: d.Price, DownPayment: d.DownPayment, TermMonths: d.TermMonths, FirstDueDate: d.FirstDueDate, ScheduledTotal: d.ScheduledTotal, RegularPayment: d.RegularPayment, Rows: make([]installmentDraftRowDTO, 0, len(d.Rows))}
	for _, row := range d.Rows {
		out.Rows = append(out.Rows, installmentDraftRowDTO{Number: row.Number, DueDate: row.DueDate, Amount: row.Amount, Balance: row.Balance})
	}
	return out
}

func toInstallmentPlan(v *service.InstallmentPlanView) *installmentPlanDTO {
	if v == nil {
		return nil
	}
	p := v.Plan
	out := &installmentPlanDTO{ID: p.ID, State: v.State, ContractReference: p.ContractReference, ContractSignedOn: p.ContractSignedOn.Format(time.DateOnly), ContractFileIDs: ids(p.ContractFileIDs), ContractTotal: money.Money{AmountMinor: p.ContractTotalMinor, Currency: p.Currency}, DownPaymentInvoiceID: p.DownPaymentInvoiceID, DownPayment: money.Money{AmountMinor: p.DownPaymentMinor, Currency: p.Currency}, ScheduledTotal: v.ScheduledTotal, Paid: v.Paid, Pending: v.Pending, Outstanding: v.Outstanding, Available: v.Available, SettlementState: v.SettlementState, AllowedActions: append([]string{}, v.AllowedActions...), Payments: []installmentPaymentDTO{}, Rows: []installmentPlanRowDTO{}}
	for _, r := range v.Rows {
		i := toInvoice(&r.Invoice)
		due := ""
		if i.DueDate != nil {
			due = *i.DueDate
		}
		out.Rows = append(out.Rows, installmentPlanRowDTO{Number: r.Number, InvoiceID: i.ID, DueDate: due, Amount: i.Amount, Paid: i.Paid, Pending: i.Pending, Outstanding: i.Outstanding, Available: i.Available, AllowedActions: i.AllowedActions})
	}
	for i := range v.Payments {
		out.Payments = append(out.Payments, toInstallmentPayment(&v.Payments[i]))
	}
	return out
}

func dateString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.DateOnly)
	return &s
}

type dealDTO struct {
	ID                    string               `json:"id"`
	BranchID              string               `json:"branchId"`
	Customer              customerDTO          `json:"customer"`
	LeadID                *string              `json:"leadId"`
	VehicleID             string               `json:"vehicleId"`
	VehicleSnapshot       *vehicleSnapshotDTO  `json:"vehicleSnapshot"`
	PaymentScheme         string               `json:"paymentScheme"`
	Price                 money.Money          `json:"price"`
	Status                string               `json:"status"`
	StatusReason          string               `json:"statusReason"`
	ContractSignedOn      *string              `json:"contractSignedOn"`
	ContractReference     string               `json:"contractReference"`
	ContractFileIDs       []string             `json:"contractFileIds"`
	RegisteredOn          *string              `json:"registeredOn"`
	PlateNumber           string               `json:"plateNumber"`
	RegistrationReference string               `json:"registrationReference"`
	DeliveredAt           *time.Time           `json:"deliveredAt"`
	Invoices              []invoiceDTO         `json:"invoices,omitempty"`
	Checklist             *service.Checklist   `json:"checklist,omitempty"`
	AllowedActions        []string             `json:"allowedActions"`
	History               []historyDTO         `json:"history,omitempty"`
	InstallmentPlan       *installmentPlanDTO  `json:"installmentPlan"`
	InstallmentDraft      *installmentDraftDTO `json:"installmentDraft"`
	Revision              string               `json:"revision"`
	UpdatedAt             time.Time            `json:"updatedAt"`
}

type vehicleSnapshotDTO struct {
	VehicleID                 string `json:"vehicleId"`
	VIN                       string `json:"vin"`
	ModelID                   string `json:"modelId"`
	ModelSpecificationVersion string `json:"modelSpecificationVersion"`
	ExteriorColor             string `json:"exteriorColor"`
	InteriorColor             string `json:"interiorColor"`
}

func toVehicleSnapshot(s *model.VehicleSnapshot) *vehicleSnapshotDTO {
	if s == nil {
		return nil
	}
	return &vehicleSnapshotDTO{VehicleID: s.VehicleID, VIN: s.VIN, ModelID: s.ModelID,
		ModelSpecificationVersion: s.ModelSpecificationVersion, ExteriorColor: s.ExteriorColor, InteriorColor: s.InteriorColor}
}

func toDeal(full bool) func(*service.DealView) dealDTO {
	return func(v *service.DealView) dealDTO {
		d := v.Deal
		out := dealDTO{
			ID: d.ID, BranchID: d.BranchID, Customer: toCustomer(&v.Customer), LeadID: d.LeadID, VehicleID: d.VehicleID, VehicleSnapshot: toVehicleSnapshot(d.VehicleSnapshot),
			PaymentScheme: d.PaymentScheme, Price: d.Price(), Status: d.Status, StatusReason: d.StatusReason,
			ContractSignedOn: dateString(d.ContractSignedOn), ContractReference: d.ContractReference, ContractFileIDs: ids(d.ContractFileIDs),
			RegisteredOn: dateString(d.RegisteredOn), PlateNumber: d.PlateNumber, RegistrationReference: d.RegistrationReference,
			DeliveredAt: d.DeliveredAt, AllowedActions: []string{}, Revision: httpx.Revision(d.Version), UpdatedAt: d.UpdatedAt,
		}
		if full {
			out.Invoices = []invoiceDTO{}
			for i := range v.Invoices {
				out.Invoices = append(out.Invoices, toInvoice(&v.Invoices[i]))
			}
			c := v.Checklist
			out.Checklist, out.History, out.InstallmentPlan = &c, toHistory(v.History), toInstallmentPlan(v.InstallmentPlan)
			out.InstallmentDraft = toInstallmentDraft(v.InstallmentDraft)
			if d.Status == "reserved" {
				if d.ContractSignedOn == nil {
					out.AllowedActions = append(out.AllowedActions, "record-contract")
				}
				out.AllowedActions = append(out.AllowedActions, "issue-invoice")
				if d.RegisteredOn == nil {
					out.AllowedActions = append(out.AllowedActions, "record-registration")
				}
				out.AllowedActions = append(out.AllowedActions, "cancel")
				if c.Ready() {
					out.AllowedActions = append(out.AllowedActions, "deliver")
				}
			}
			if v.CanCreateInstallmentPlan {
				out.AllowedActions = append(out.AllowedActions, "create-installment-plan")
			}
			if v.CanSetInstallmentTerms {
				out.AllowedActions = append(out.AllowedActions, "set-installment-terms")
			}
		}
		return out
	}
}

func mapSlice[T, D any](items []T, f func(*T) D) []D {
	out := make([]D, len(items))
	for i := range items {
		out[i] = f(&items[i])
	}
	return out
}

func ids(raw []byte) []string {
	out := []string{}
	_ = json.Unmarshal(raw, &out)
	return out
}
