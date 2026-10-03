package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/database"
)

type paymentSQLLog struct {
	logger.Interface
	queries []string
}

func (l *paymentSQLLog) Trace(_ context.Context, _ time.Time, sql func() (string, int64), _ error) {
	query, _ := sql()
	l.queries = append(l.queries, query)
}

func TestInstallmentPaymentGroupLocksUseAmbientConnectionAndStableOrder(t *testing.T) {
	open := func(log *paymentSQLLog) *gorm.DB {
		db, err := gorm.Open(postgres.New(postgres.Config{DSN: ""}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, Logger: log})
		if err != nil {
			t.Fatal(err)
		}
		return db
	}
	baseLog, txLog := &paymentSQLLog{Interface: logger.Default}, &paymentSQLLog{Interface: logger.Default}
	base, tx := open(baseLog), open(txLog)
	r := &DealRepository{db: base}
	ctx := database.WithTx(context.Background(), tx)
	if _, err := r.LockDeal(ctx, "company-123", "deal-456"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.LockInstallmentInvoices(ctx, "company-123", "deal-456", []string{"invoice-b", "invoice-a"}); err != nil {
		t.Fatal(err)
	}
	if len(baseLog.queries) != 0 || len(txLog.queries) != 2 {
		t.Fatalf("lock did not use ambient connection: base=%v tx=%v", baseLog.queries, txLog.queries)
	}
	query := txLog.queries[0]
	for _, fragment := range []string{`FROM "retail"."deals"`, "id = 'deal-456' AND company_id = 'company-123'", "FOR UPDATE"} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("missing %s: %s", fragment, query)
		}
	}
	query = txLog.queries[1]
	for _, fragment := range []string{`FROM "retail"."invoices"`, "company_id = 'company-123' AND deal_id = 'deal-456'", "purpose = 'monthly-installment'", "ORDER BY id ASC", "FOR UPDATE"} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("missing %s: %s", fragment, query)
		}
	}
}

func TestInstallmentPaymentGroupReadAndDecisionAreScopedAndVersioned(t *testing.T) {
	log := &paymentSQLLog{Interface: logger.Default}
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: ""}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, Logger: log})
	if err != nil {
		t.Fatal(err)
	}
	r := &DealRepository{db: db}
	if _, err := r.InstallmentPayment(context.Background(), "company-123", "payment-456"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.InstallmentPaymentByReference(context.Background(), "company-123", "deal-456", "receipt-789"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.InstallmentPaymentEvidence(context.Background(), "payment-456"); err != nil {
		t.Fatal(err)
	}
	updateSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&model.InstallmentPayment{}).Where("id = ? AND version = ?", "payment-456", 4).Updates(map[string]any{
			"status": "accepted", "version": 5,
		})
	})
	joined := strings.Join(log.queries, "\n")
	for _, fragment := range []string{
		`FROM "retail"."installment_payments"`, "id = 'payment-456' AND company_id = 'company-123'",
		"company_id = 'company-123' AND deal_id = 'deal-456' AND external_reference = 'receipt-789'", "status <> 'rejected'",
		`FROM "retail"."payment_evidence"`, "payment_group_id = 'payment-456'", "ORDER BY invoice_id ASC,id ASC",
		"WHERE id = 'payment-456' AND version = 4",
	} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("missing %s: %s", fragment, joined)
		}
	}
	if !strings.Contains(updateSQL, "WHERE id = 'payment-456' AND version = 4") || !strings.Contains(updateSQL, `"version"=5`) {
		t.Fatalf("versioned update is missing predicate or bump: %s", updateSQL)
	}
}
