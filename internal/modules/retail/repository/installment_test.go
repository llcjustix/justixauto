package repository

import (
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"justixauto/internal/modules/retail/model"
)

func TestInstallmentPlanRepositoryQueryShape(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: ""}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB { return tx.Where("deal_id = ?", "deal-id").Take(&model.InstallmentPlan{}) })
	if !strings.Contains(sql, `FROM "retail"."installment_plans"`) || !strings.Contains(sql, "deal_id") {
		t.Fatalf("unexpected plan query: %s", sql)
	}
}
