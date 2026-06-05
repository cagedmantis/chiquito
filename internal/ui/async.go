package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"argc.dev/chiquito/internal/config"
	"argc.dev/chiquito/internal/spell"
)

// --- messages --------------------------------------------------------------

// dictLoadedMsg delivers a dictionary loaded off the UI thread.
type dictLoadedMsg struct{ dict spell.Dictionary }

// spellTickMsg fires after the debounce interval; version identifies the edit it
// was scheduled for so superseded ticks are ignored.
type spellTickMsg struct{ version int }

// spellResultMsg carries spell-check results tagged with the doc version they
// were computed against.
type spellResultMsg struct {
	version int
	spans   []spell.Misspelling
}

// configTickMsg fires periodically to poll the config file for changes.
type configTickMsg struct{}

// --- lifecycle -------------------------------------------------------------

// Init implements tea.Model. It kicks off dictionary loading (if spell checking
// is enabled) and starts the config-file watch loop.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{watchConfigCmd()}
	if m.cfg.Features.SpellCheck {
		cmds = append(cmds, loadDictCmd())
	}
	return tea.Batch(cmds...)
}

// --- spell checking --------------------------------------------------------

func loadDictCmd() tea.Cmd {
	return func() tea.Msg { return dictLoadedMsg{dict: spell.Load()} }
}

// scheduleSpell returns a debounced tick command; the actual check runs only if
// no newer edit arrives before it fires (see spellTickMsg handling).
func (m *Model) scheduleSpell() tea.Cmd {
	if !m.cfg.Features.SpellCheck || m.checker == nil {
		return nil
	}
	v := m.docVersion
	return tea.Tick(spellDebounce, func(time.Time) tea.Msg {
		return spellTickMsg{version: v}
	})
}

// runSpellCmd snapshots the document and checks it in a goroutine, off the UI
// thread, so typing and loading are never blocked.
func (m *Model) runSpellCmd(version int) tea.Cmd {
	if m.checker == nil {
		return nil
	}
	text := m.ed.Text()
	checker := m.checker
	return func() tea.Msg {
		return spellResultMsg{version: version, spans: spell.Check(text, checker)}
	}
}

// --- config hot-reload -----------------------------------------------------

func watchConfigCmd() tea.Cmd {
	return tea.Tick(configPollInterval, func(time.Time) tea.Msg {
		return configTickMsg{}
	})
}

// maybeReloadConfig reloads and applies the config if the file changed, then
// reschedules the next poll. It returns a batch that always includes the next
// tick, plus any command produced by applying new settings.
func (m *Model) maybeReloadConfig() tea.Cmd {
	next := watchConfigCmd()
	mt, err := config.ModTime()
	if err != nil || !mt.After(m.configMod) {
		return next
	}
	m.configMod = mt
	cfg, _, err := config.Load()
	if err != nil {
		m.status = "Config reload failed: " + err.Error()
		return next
	}
	applyCmd := m.applyConfig(cfg)
	m.status = "Reloaded config"
	return tea.Batch(next, applyCmd)
}

// applyConfig adopts a newly loaded configuration: rebinds keys, re-themes,
// updates editor settings, and toggles spell checking. It returns a command to
// run if a newly enabled subsystem needs initialization (e.g. dictionary load).
func (m *Model) applyConfig(cfg config.Config) tea.Cmd {
	m.cfg = cfg
	m.km = newKeymap(cfg)
	m.theme = themeByName(cfg.Theme.Name)
	m.lineNumbers = cfg.Editor.LineNumbers
	m.tabWidth = cfg.Editor.TabWidth
	m.ed.SetTabStops(cfg.Editor.TabWidth, cfg.Editor.ExpandTabs)
	// Rebuild the highlighter so a changed theme (Chroma style) takes effect.
	m.hl = newHighlighter(m.ed.Name(), cfg.Theme.Name, m.theme)
	m.synStale = true

	switch {
	case cfg.Features.SpellCheck && m.checker == nil:
		return loadDictCmd()
	case !cfg.Features.SpellCheck:
		m.checker = nil
		m.spellSpans = nil
	}
	m.applyLayout()
	return nil
}
