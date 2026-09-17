// Package slug derives URL-friendly identifiers from document titles.
package slug

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// MaxLength caps generated slugs so unique indexes stay reasonable.
const MaxLength = 200

// Make converts a title into a lowercase, kebab-cased slug. Non-ASCII letters
// are transliterated to their closest ASCII form where possible (NFKD) and
// dropped otherwise. Returns "untitled" when nothing usable remains.
func Make(title string) string {
	decomposed := norm.NFKD.String(strings.ToLower(strings.TrimSpace(title)))

	var b strings.Builder
	b.Grow(len(decomposed))
	prevDash := false

	for _, r := range decomposed {
		switch {
		case unicode.Is(unicode.Mn, r): // combining marks from NFKD
			continue
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "untitled"
	}
	if len(out) > MaxLength {
		out = strings.Trim(out[:MaxLength], "-")
	}
	if out == "" {
		return "untitled"
	}
	return out
}
