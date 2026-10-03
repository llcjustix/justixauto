package migrations

import (
	"strings"
	"testing"
)

func TestVehicleColorsMigration(t *testing.T) {
	up := embeddedSQL(t, "000034_vehicle_colors.up.sql")
	down := embeddedSQL(t, "000034_vehicle_colors.down.sql")
	for _, join := range []string{
		"u.model_id = s.model_id AND u.spec_version = s.spec_version",
		"b.model_id = s.model_id AND b.spec_version = s.spec_version",
	} {
		if !strings.Contains(up, join) {
			t.Fatalf("backfill does not pin exact model version: %q", join)
		}
	}
	for _, prohibited := range []string{"current_spec_version", "commerce.", "UPDATE retail_deals"} {
		if strings.Contains(strings.ToLower(up), strings.ToLower(prohibited)) {
			t.Fatalf("history-unsafe migration content %q", prohibited)
		}
	}
	for _, guard := range []string{"vehicle selections exist", "receipt selections exist", "multi-color palettes", "retail vehicle snapshots"} {
		if !strings.Contains(down, guard) {
			t.Fatalf("down migration missing safety guard %q", guard)
		}
	}
}
