package ui

import (
	"github.com/charmbracelet/lipgloss"

	"argc.dev/chiquito/internal/syntax"
)

// theme maps syntax token types to lipgloss styles. The zero value of a missing
// entry is an unstyled (plain) cell.
type theme struct {
	styles      map[syntax.TokenType]lipgloss.Style
	searchMatch lipgloss.Style
	currentHit  lipgloss.Style
}

// themeByName returns a theme. Only "default" is defined today; unknown names
// fall back to it. (Config-driven themes expand in Phase 4.)
func themeByName(string) theme {
	c := func(s string) lipgloss.Color { return lipgloss.Color(s) }
	return theme{
		styles: map[syntax.TokenType]lipgloss.Style{
			syntax.Keyword:  lipgloss.NewStyle().Foreground(c("13")),
			syntax.Ident:    lipgloss.NewStyle(),
			syntax.Number:   lipgloss.NewStyle().Foreground(c("11")),
			syntax.String:   lipgloss.NewStyle().Foreground(c("10")),
			syntax.Comment:  lipgloss.NewStyle().Foreground(c("8")).Italic(true),
			syntax.Operator: lipgloss.NewStyle().Foreground(c("14")),
			syntax.Heading:  lipgloss.NewStyle().Foreground(c("12")).Bold(true),
			syntax.Emphasis: lipgloss.NewStyle().Italic(true),
			syntax.Code:     lipgloss.NewStyle().Foreground(c("10")),
			syntax.Link:     lipgloss.NewStyle().Foreground(c("14")).Underline(true),
		},
		searchMatch: lipgloss.NewStyle().Background(c("3")).Foreground(c("0")),
		currentHit:  lipgloss.NewStyle().Background(c("11")).Foreground(c("0")),
	}
}

func (t theme) style(tt syntax.TokenType) lipgloss.Style {
	if s, ok := t.styles[tt]; ok {
		return s
	}
	return lipgloss.NewStyle()
}
