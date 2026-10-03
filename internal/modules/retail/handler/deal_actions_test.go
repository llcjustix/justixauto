package handler

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/modules/retail/service"
)

func TestDealActionProjectionCompletionFacts(t *testing.T) {
	completed := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	ready := true
	tests := []struct {
		name       string
		contract   *time.Time
		registered *time.Time
		want       []string
	}{
		{"neither", nil, nil, []string{"record-contract", "issue-invoice", "record-registration", "cancel", "create-installment-plan", "set-installment-terms"}},
		{"contract only", &completed, nil, []string{"issue-invoice", "record-registration", "cancel", "create-installment-plan", "set-installment-terms"}},
		{"registration only", nil, &completed, []string{"record-contract", "issue-invoice", "cancel", "create-installment-plan", "set-installment-terms"}},
		{"both", &completed, &completed, []string{"issue-invoice", "cancel", "deliver", "create-installment-plan", "set-installment-terms"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &service.DealView{
				Deal: model.Deal{Status: "reserved", ContractSignedOn: tt.contract, RegisteredOn: tt.registered},
				Checklist: service.Checklist{Contract: tt.contract != nil, Registered: tt.registered != nil, RegistrationPaid: tt.registered != nil, VehiclePayment: &ready, PolicyResolved: true},
				CanCreateInstallmentPlan: true,
				CanSetInstallmentTerms:   true,
			}
			got := toDeal(true)(v).AllowedActions
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("allowed actions = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDealActionProjectionRegistrationPolicy(t *testing.T) {
	complete, incomplete := true, false
	completed := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		scheme    string
		checklist service.Checklist
		deliver   bool
	}{
		{"cash registration absent", "cash", service.Checklist{Contract: true, VehiclePayment: &complete, RegistrationOptional: true, PolicyResolved: true}, true},
		{"cash contract absent", "cash", service.Checklist{VehiclePayment: &complete, RegistrationOptional: true, PolicyResolved: true}, false},
		{"cash payment incomplete", "cash", service.Checklist{Contract: true, VehiclePayment: &incomplete, RegistrationOptional: true, PolicyResolved: true}, false},
		{"cash default remains strict", "cash", service.Checklist{Contract: true, VehiclePayment: &complete, PolicyResolved: true}, false},
		{"own registration absent", "own-installment", service.Checklist{Contract: true, FirstInstallment: &complete, InsuranceApproved: &complete, RegistrationPaid: true, PolicyResolved: true}, false},
		{"own fee absent", "own-installment", service.Checklist{Contract: true, FirstInstallment: &complete, InsuranceApproved: &complete, Registered: true, PolicyResolved: true}, false},
		{"own ready", "own-installment", service.Checklist{Contract: true, FirstInstallment: &complete, InsuranceApproved: &complete, RegistrationPaid: true, Registered: true, PolicyResolved: true}, true},
		{"partner unresolved", "partner-finance", service.Checklist{Contract: true, RegistrationPaid: true, Registered: true}, false},
		{"zero checklist", "cash", service.Checklist{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := model.Deal{Status: "reserved", PaymentScheme: tt.scheme}
			if tt.checklist.Contract {
				d.ContractSignedOn = &completed
			}
			if tt.checklist.Registered {
				d.RegisteredOn = &completed
			}
			v := &service.DealView{Deal: d, Checklist: tt.checklist}
			dto := toDeal(true)(v)
			if slices.Contains(dto.AllowedActions, "deliver") != tt.deliver {
				t.Fatalf("deliver projection = %v, want %v", dto.AllowedActions, tt.deliver)
			}
			if slices.Contains(dto.AllowedActions, "record-registration") == tt.checklist.Registered {
				t.Fatalf("optional registration lost recording action: %v", dto.AllowedActions)
			}
			raw, err := json.Marshal(dto)
			if err != nil {
				t.Fatal(err)
			}
			var wire struct {
				Checklist map[string]json.RawMessage `json:"checklist"`
			}
			if err := json.Unmarshal(raw, &wire); err != nil {
				t.Fatal(err)
			}
			want := "false"
			if tt.checklist.RegistrationOptional {
				want = "true"
			}
			if got := string(wire.Checklist["registrationOptional"]); got != want {
				t.Fatalf("registrationOptional JSON = %q, want %s", got, want)
			}
			if !reflect.DeepEqual(dto.Checklist, &tt.checklist) {
				t.Fatal("checklist facts were changed in projection")
			}
			if list := toDeal(false)(v); list.Checklist != nil || len(list.AllowedActions) != 0 {
				t.Fatal("list projection gained detail/actions")
			}
		})
	}
}

func TestDealActionProjectionListAndNonReservedBehavior(t *testing.T) {
	completed := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	v := &service.DealView{
		Deal:                     model.Deal{Status: "delivered", ContractSignedOn: &completed, RegisteredOn: &completed},
		CanCreateInstallmentPlan: true,
		CanSetInstallmentTerms:   true,
	}
	if got := toDeal(false)(v).AllowedActions; !reflect.DeepEqual(got, []string{}) {
		t.Fatalf("list allowed actions = %v, want empty", got)
	}
	if got := toDeal(true)(v).AllowedActions; !reflect.DeepEqual(got, []string{"create-installment-plan", "set-installment-terms"}) {
		t.Fatalf("non-reserved allowed actions = %v", got)
	}
}
