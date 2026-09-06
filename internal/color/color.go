// Copyright © 2026 The Homeport Team
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package color

import (
	"fmt"
	"strings"
)

// DefaultPalette is the ColorBrewer Set3 qualitative palette (12 colors).
var DefaultPalette = []string{
	"#8dd3c7",
	"#ffffb3",
	"#bebada",
	"#fb8072",
	"#80b1d3",
	"#fdb462",
	"#b3de69",
	"#fccde5",
	"#d9d9d9",
	"#bc80bd",
	"#ccebc5",
	"#ffed6f",
}

// NormalizeColor validates and canonicalizes a color. It accepts RGB hex codes
// (#RGB and #RRGGBB) and CSS named colors. Hex codes are normalized to
// uppercase #RRGGBB; named colors are resolved to their hex value. An error is
// returned for unrecognized or malformed colors.
func NormalizeColor(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", fmt.Errorf("color cannot be empty")
	}

	if strings.HasPrefix(s, "#") {
		return normalizeHex(s)
	}

	lower := strings.ToLower(s)
	if hex, ok := colorNames[lower]; ok {
		return hex, nil
	}
	return "", fmt.Errorf("unrecognized color %q: expected #RGB, #RRGGBB, or a CSS color name", input)
}

// NormalizeColorOr returns the normalized color, or fallback when invalid.
func NormalizeColorOr(input, fallback string) string {
	if c, err := NormalizeColor(input); err == nil {
		return c
	}
	return fallback
}

func normalizeHex(s string) (string, error) {
	body := strings.TrimPrefix(s, "#")
	switch len(body) {
	case 3:
		r := body[0]
		g := body[1]
		b := body[2]
		if !isHexTriplet(body) {
			return "", fmt.Errorf("invalid hex color %q", s)
		}
		return fmt.Sprintf("#%s%s%s%s%s%s", strings.ToUpper(string(r)), strings.ToUpper(string(r)),
			strings.ToUpper(string(g)), strings.ToUpper(string(g)),
			strings.ToUpper(string(b)), strings.ToUpper(string(b))), nil
	case 6:
		if !isHexTriplet(body) {
			return "", fmt.Errorf("invalid hex color %q", s)
		}
		return "#" + strings.ToUpper(body), nil
	default:
		return "", fmt.Errorf("invalid hex color %q: expected #RGB or #RRGGBB", s)
	}
}

func isHexTriplet(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// ColorForCategory resolves the display color for a category. When the
// category has an explicitly configured color it is used; otherwise a
// deterministic color is derived from the palette using the category name.
func ColorForCategory(name, configured string) string {
	if configured != "" {
		if c, err := NormalizeColor(configured); err == nil {
			return c
		}
	}
	return PaletteColor(name, DefaultPalette)
}

// PaletteColor picks a stable palette entry for a given key.
func PaletteColor(key string, palette []string) string {
	if len(palette) == 0 {
		return "#888888"
	}
	h := 0
	for i := 0; i < len(key); i++ {
		h = h*31 + int(key[i])
	}
	if h < 0 {
		h = -h
	}
	return palette[h%len(palette)]
}

// NextPaletteColor returns the palette entry that follows the given used
// count, cycling when exhausted. Used for sequential assignment of new
// categories during CSV import.
func NextPaletteColor(used int) string {
	if len(DefaultPalette) == 0 {
		return "#888888"
	}
	return DefaultPalette[used%len(DefaultPalette)]
}
