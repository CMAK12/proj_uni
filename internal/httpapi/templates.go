package httpapi

import (
	"embed"
	"fmt"
	"html/template"
	"strings"
	"unicode"
)

//go:embed templates/*.html
var templateFS embed.FS

// templateFuncs are helpers available inside templates.
var templateFuncs = template.FuncMap{
	// pct renders a 0..1 fraction as a whole-percent string.
	"pct": func(f float64) string { return fmt.Sprintf("%.0f%%", f*100) },
	// pctWidth renders a 0..1 fraction as a CSS width value.
	"pctWidth": func(f float64) template.CSS { return template.CSS(fmt.Sprintf("%.0f%%", f*100)) },
	// initials returns up to two uppercase initials for an avatar.
	"initials": func(name string) string {
		out := make([]rune, 0, 2)
		for _, field := range strings.Fields(name) {
			r := []rune(field)
			if len(r) == 0 {
				continue
			}
			out = append(out, unicode.ToUpper(r[0]))
			if len(out) == 2 {
				break
			}
		}
		if len(out) == 0 {
			return "?"
		}
		return string(out)
	},
}
