package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"justixauto/internal/modules/inventory/model"
)

func TestVehicleFilterEligibleQueryShape(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	r := &VehicleRepository{db: db}
	rows := []model.VehicleRow{}
	stmt := r.listQuery(context.Background(), model.VehicleFilter{
		CompanyID: "owner", ModelID: "model", WarehouseID: "warehouse", Placement: "warehouse",
		Search: "AB%_\\Z", Eligible: true, Limit: 25, Offset: 100,
	}).Scan(&rows).Statement
	sql := stmt.SQL.String()
	for _, clause := range []string{
		"p.vehicle_id IS NOT NULL", "p.warehouse_id = $3", "vehicle_units.model_id = $4",
		"vehicle_units.vin LIKE $5 ESCAPE", "vehicle_units.owner_company_id = $6", "vehicle_units.vin <> ''",
		"NOT EXISTS (SELECT 1 FROM inventory_reservations", "ORDER BY inventory_vehicle_units.vin, inventory_vehicle_units.id", "LIMIT $7 OFFSET $8",
	} {
		if !strings.Contains(sql, clause) {
			t.Errorf("query missing %q: %s", clause, sql)
		}
	}
	if strings.Index(sql, "vehicle_units.model_id") > strings.Index(sql, "LIMIT") {
		t.Fatalf("filter must precede pagination: %s", sql)
	}
	vars := fmt.Sprint(stmt.Vars)
	if !strings.Contains(vars, "\\%") || !strings.Contains(vars, "\\_") {
		t.Fatalf("VIN wildcard was not escaped: %#v", stmt.Vars)
	}
}

func TestVehicleFilterGenericQueryKeepsCustodianVisibility(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	r := &VehicleRepository{db: db}
	rows := []model.VehicleRow{}
	sql := r.listQuery(context.Background(), model.VehicleFilter{CompanyID: "company"}).Scan(&rows).Statement.SQL.String()
	if !strings.Contains(sql, "owner_company_id = $1 OR w.company_id = $2") || strings.Contains(sql, "NOT EXISTS") {
		t.Fatalf("generic query visibility changed: %s", sql)
	}
}
