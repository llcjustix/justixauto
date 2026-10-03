package service

import (
	"reflect"
	"testing"

	"justixauto/internal/pkg/apperr"
)

func TestColorPaletteScalarCompatibility(t *testing.T) {
	var validation apperr.Validation
	scalar, colors := colorPalette(&validation, "exteriorColor", "exteriorColors", "  Blue  ", nil)
	if err := validation.Err(); err != nil || scalar != "Blue" || !reflect.DeepEqual(colors, []string{"Blue"}) {
		t.Fatalf("scalar=%q colors=%v err=%v", scalar, colors, err)
	}
}

func TestColorPaletteTrimsAndDeduplicates(t *testing.T) {
	var validation apperr.Validation
	scalar, colors := colorPalette(&validation, "exteriorColor", "exteriorColors", "", []string{" Blue ", "blue", "Red"})
	if err := validation.Err(); err != nil || scalar != "" || !reflect.DeepEqual(colors, []string{"Blue", "Red"}) {
		t.Fatalf("scalar=%q colors=%v err=%v", scalar, colors, err)
	}
}

func TestColorPaletteRejectsBlankAndConflictingForms(t *testing.T) {
	var empty apperr.Validation
	colorPalette(&empty, "exteriorColor", "exteriorColors", "Blue", []string{})
	if empty.Err() == nil {
		t.Fatal("explicit empty palette accepted")
	}
	var blank apperr.Validation
	colorPalette(&blank, "exteriorColor", "exteriorColors", "", []string{" "})
	if blank.Err() == nil {
		t.Fatal("blank palette accepted")
	}
	var conflict apperr.Validation
	colorPalette(&conflict, "exteriorColor", "exteriorColors", "Blue", []string{"Red"})
	if conflict.Err() == nil {
		t.Fatal("conflicting scalar and palette accepted")
	}
}

func TestSpecColorsUseEmptyScalarForMultipleChoices(t *testing.T) {
	var validation apperr.Validation
	_, _, _, spec := (SpecInput{
		Make: "Make", Model: "Model", Variant: "Variant", Year: 2026, BodyType: "SUV",
		ExteriorColors: []string{"Blue", "Red"}, InteriorColors: []string{"Black"}, Powertrain: "EV", Drivetrain: "AWD",
	}).validate(&validation)
	if err := validation.Err(); err != nil || spec.ExteriorColor != "" || spec.InteriorColor != "Black" {
		t.Fatalf("spec=%+v err=%v", spec, err)
	}
}
