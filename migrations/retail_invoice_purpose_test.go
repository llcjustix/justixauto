package migrations

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	lineComment        = regexp.MustCompile(`(?m)--[^\n]*`)
	purposeCheck        = regexp.MustCompile(`(?is)CHECK\s*\(\s*purpose\s+IN\s*\(([^)]*)\)\s*\)`)
	singleQuotedLiteral = regexp.MustCompile(`'([^']*)'`)
	dropPurposeCheck    = regexp.MustCompile(`DROP CONSTRAINT (IF EXISTS )?invoices_purpose_check\b`)
)

func TestRetailMonthlyInvoicePurposeMigration(t *testing.T) {
	legacy := embeddedSQL(t, "000012_retail_deals.up.sql")
	up := embeddedSQL(t, "000032_retail_monthly_invoice_purpose.up.sql")
	down := embeddedSQL(t, "000032_retail_monthly_invoice_purpose.down.sql")
	legacyPurposes := purposeAllowlist(t, legacy)

	t.Run("forward whitelist", func(t *testing.T) {
		assertSameStrings(t, purposeAllowlist(t, up), append(legacyPurposes, "monthly-installment"))
		assertConstraintReplacement(t, up)
	})

	t.Run("backward whitelist and guard", func(t *testing.T) {
		assertSameStrings(t, purposeAllowlist(t, down), legacyPurposes)
		if !strings.HasPrefix(strings.TrimSpace(down), "DO $$") || !strings.HasSuffix(strings.TrimSpace(down), "END $$;") {
			t.Fatalf("backward guard and replacement must share one DO block: %s", down)
		}
		guard := "IF EXISTS (SELECT 1 FROM retail_invoices WHERE purpose = 'monthly-installment') THEN"
		guardAt := strings.Index(down, guard)
		replacementAt := strings.Index(down, "ALTER TABLE retail_invoices")
		if guardAt < 0 || replacementAt < 0 || guardAt > replacementAt {
			t.Fatalf("monthly invoice guard must precede constraint replacement: %s", down)
		}
		if strings.Contains(strings.ToLower(down[guardAt:replacementAt]), "status") {
			t.Fatalf("monthly invoice guard must not filter by status: %s", down)
		}
		if !strings.Contains(down, "RAISE EXCEPTION") {
			t.Fatalf("monthly invoice guard must raise: %s", down)
		}
		assertConstraintReplacement(t, down)
	})

	t.Run("preservation", func(t *testing.T) {
		for _, sql := range []string{up, down} {
			withoutComments := lineComment.ReplaceAllString(sql, "")
			for _, prohibited := range []string{
				"CASCADE", "NOT VALID", "INSERT ", "UPDATE ", "DELETE ", "TRUNCATE ",
				"DROP TABLE", "DROP COLUMN", "DROP INDEX", "CREATE INDEX",
				"installment_plan", "installment_number",
			} {
				if strings.Contains(strings.ToUpper(withoutComments), strings.ToUpper(prohibited)) {
					t.Fatalf("migration contains prohibited %q: %s", prohibited, sql)
				}
			}
		}
	})
}

func embeddedSQL(t *testing.T, name string) string {
	t.Helper()
	b, err := FS.ReadFile(name)
	if err != nil {
		t.Fatalf("read embedded %s: %v", name, err)
	}
	return string(b)
}

func purposeAllowlist(t *testing.T, sql string) []string {
	t.Helper()
	matches := purposeCheck.FindAllStringSubmatch(lineComment.ReplaceAllString(sql, ""), -1)
	if len(matches) != 1 {
		t.Fatalf("expected exactly one purpose CHECK, found %d: %s", len(matches), sql)
	}
	literals := singleQuotedLiteral.FindAllStringSubmatch(matches[0][1], -1)
	purposes := make([]string, 0, len(literals))
	for _, literal := range literals {
		purposes = append(purposes, literal[1])
	}
	return purposes
}

func assertConstraintReplacement(t *testing.T, sql string) {
	t.Helper()
	if strings.Count(sql, "ALTER TABLE retail_invoices") != 1 ||
		len(dropPurposeCheck.FindAllString(sql, -1)) != 1 ||
		strings.Count(sql, "ADD CONSTRAINT invoices_purpose_check") != 1 {
		t.Fatalf("expected one atomic invoices_purpose_check replacement: %s", sql)
	}
}

func assertSameStrings(t *testing.T, got, want []string) {
	t.Helper()
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got purposes %v, want %v", got, want)
	}
}
