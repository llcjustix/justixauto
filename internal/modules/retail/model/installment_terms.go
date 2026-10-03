package model

import "justixauto/internal/pkg/money"

// InstallmentDraft is the authoritative generated terms snapshot. It is not a
// final InstallmentPlan and deliberately carries no payable invoice identity.
type InstallmentDraft struct {
	PolicyID       string                `json:"policyId"`
	PolicyVersion  int                   `json:"policyVersion"`
	Price          money.Money           `json:"price"`
	DownPayment    money.Money           `json:"downPayment"`
	TermMonths     int                   `json:"termMonths"`
	FirstDueDate   string                `json:"firstDueDate"`
	ScheduledTotal money.Money           `json:"scheduledTotal"`
	RegularPayment money.Money           `json:"regularPayment"`
	Rows           []InstallmentDraftRow `json:"rows"`
}

type InstallmentDraftRow struct {
	Number  int         `json:"number"`
	DueDate string      `json:"dueDate"`
	Amount  money.Money `json:"amount"`
	Balance money.Money `json:"balance"`
}
