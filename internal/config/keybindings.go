package config

import "strings"

// NormalizeChord converts a human-written key chord into the canonical form
// produced by Bubble Tea's KeyMsg.String, so that configuration files may use
// either Emacs-style notation ("C-x C-s", "M-%") or Bubble Tea notation
// ("ctrl+x ctrl+s", "alt+%"). Multi-key sequences are space-separated; each key
// is normalized independently.
//
//	C-x  -> ctrl+x       M-x / Alt-x -> alt+x       S-x / Shift-x -> shift+x
//
// Already-canonical chords pass through unchanged.
func NormalizeChord(chord string) string {
	fields := strings.Fields(chord)
	for i, k := range fields {
		fields[i] = normalizeKey(k)
	}
	return strings.Join(fields, " ")
}

func normalizeKey(k string) string {
	// Repeatedly peel a recognized modifier prefix, translating it to the
	// canonical "<mod>+" form.
	var mods []string
	for {
		switch {
		case hasPrefixFold(k, "ctrl+"):
			mods, k = append(mods, "ctrl"), k[len("ctrl+"):]
		case hasPrefixFold(k, "alt+"):
			mods, k = append(mods, "alt"), k[len("alt+"):]
		case hasPrefixFold(k, "shift+"):
			mods, k = append(mods, "shift"), k[len("shift+"):]
		case hasPrefixFold(k, "C-"):
			mods, k = append(mods, "ctrl"), k[len("C-"):]
		case hasPrefixFold(k, "M-"), hasPrefixFold(k, "Alt-"):
			mods = append(mods, "alt")
			k = strings.TrimPrefix(strings.TrimPrefix(k, "M-"), "Alt-")
		case hasPrefixFold(k, "S-"), hasPrefixFold(k, "Shift-"):
			mods = append(mods, "shift")
			k = strings.TrimPrefix(strings.TrimPrefix(k, "S-"), "Shift-")
		default:
			// Lowercase the base key only when it is a named key (len > 1),
			// leaving single literal characters (including their case) intact.
			base := k
			if len(base) > 1 {
				base = strings.ToLower(base)
			}
			var b strings.Builder
			for _, m := range mods {
				b.WriteString(m)
				b.WriteByte('+')
			}
			b.WriteString(base)
			return b.String()
		}
	}
}

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}
