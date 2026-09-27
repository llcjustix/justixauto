package repository

import "testing"

func TestLikeEscapeMatchesTextLiterally(t *testing.T) {
	for in, want := range map[string]string{
		"Авто":    "Авто",
		"50%":     `50\%`,
		"a_b":     `a\_b`,
		`back\sl`: `back\\sl`,
	} {
		if got := likeEscape(in); got != want {
			t.Errorf("likeEscape(%q) = %q, want %q", in, got, want)
		}
	}
}
