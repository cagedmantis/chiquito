package ui

import (
	"strings"

	"argc.dev/chiquito/internal/config"
)

// keymap resolves key chords (as produced by tea.KeyMsg.String, e.g. "ctrl+f"
// or the multi-key "ctrl+x ctrl+s") to logical editor actions. It also records
// the set of first-keys that begin a multi-key sequence so the model knows when
// to wait for a second key.
type keymap struct {
	actions  map[string]string // chord -> action
	prefixes map[string]bool   // first key of a multi-key sequence
}

func newKeymap(cfg config.Config) keymap {
	km := keymap{
		actions:  make(map[string]string, len(cfg.Keys.Bindings)),
		prefixes: make(map[string]bool),
	}
	for action, chord := range cfg.Keys.Bindings {
		chord = strings.TrimSpace(chord)
		if chord == "" {
			continue
		}
		km.actions[chord] = action
		if i := strings.IndexByte(chord, ' '); i > 0 {
			km.prefixes[chord[:i]] = true
		}
	}
	return km
}

// isPrefix reports whether chord is the first key of a multi-key binding.
func (k keymap) isPrefix(chord string) bool { return k.prefixes[chord] }

// lookup returns the action bound to chord, if any.
func (k keymap) lookup(chord string) (string, bool) {
	a, ok := k.actions[chord]
	return a, ok
}
