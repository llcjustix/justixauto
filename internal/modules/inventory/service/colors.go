package service

import (
	"strings"

	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/validate"
)

// colorPalette accepts the legacy scalar form and the plural palette form.
// Scalars remain populated only when the resulting palette has one choice.
func colorPalette(v *apperr.Validation, scalarField, paletteField, scalar string, palette []string) (string, []string) {
	scalar = strings.TrimSpace(scalar)
	if palette == nil {
		return validate.Text(v, scalarField, scalar, 1, 50), []string{scalar}
	}

	colors := make([]string, 0, len(palette))
	seen := make(map[string]struct{}, len(palette))
	for _, color := range palette {
		color = validate.Text(v, paletteField, color, 1, 50)
		key := strings.ToLower(color)
		if color == "" {
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		colors = append(colors, color)
	}
	if len(colors) == 0 {
		v.Add(paletteField, "required")
		return "", colors
	}
	if scalar != "" && (len(colors) != 1 || !strings.EqualFold(scalar, colors[0])) {
		v.Add(scalarField, "must agree with the singleton color palette")
	}
	if len(colors) == 1 {
		return colors[0], colors
	}
	return "", colors
}
