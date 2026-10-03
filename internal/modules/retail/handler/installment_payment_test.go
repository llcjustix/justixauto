package handler

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"justixauto/internal/pkg/apperr"
)

func TestInstallmentPaymentDecisionRequiresIfMatchBeforeService(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest("POST", "/installment-payments/group/accept", strings.NewReader(`{"confirmation":true}`))
	ctx := e.NewContext(req, httptest.NewRecorder())
	if err := (&Handler{}).decideInstallmentPayment(true)(ctx); !errors.Is(err, apperr.ErrPreconditionRequired) {
		t.Fatalf("missing If-Match error = %v", err)
	}
}
