package repository

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"justixauto/internal/modules/retail/model"
)

func TestInstallmentTermsRepositoryUpdatesDraftVersioned(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: ""}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	d := &model.Deal{ID: "00000000-0000-0000-0000-000000000001", InstallmentDraft: &model.InstallmentDraft{PolicyID: "own-interest-free-equal"}, UpdatedAt: time.Now()}
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&model.Deal{}).Where("id = ? AND version = ?", d.ID, 7).Updates(dealUpdateFields(d))
	})
	if !strings.Contains(sql, "installment_draft") || !strings.Contains(sql, "version") {
		t.Fatalf("unexpected SQL: %s", sql)
	}
}
