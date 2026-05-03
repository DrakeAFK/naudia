package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
)

func Card(title string, rows [][2]string) string {
	theme := DefaultTheme()
	var b strings.Builder
	if title != "" {
		b.WriteString(theme.Title().Render(title))
		b.WriteString("\n\n")
	}
	for _, row := range rows {
		fmt.Fprintf(&b, "%-16s %s\n", row[0], row[1])
	}
	return theme.Box().Render(strings.TrimRight(b.String(), "\n"))
}

func ErrorCard(title string, lines ...string) string {
	theme := DefaultTheme()
	var b strings.Builder
	b.WriteString(theme.DangerStyle().Render(title))
	b.WriteString("\n\n")
	for _, line := range lines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return theme.Box().Render(strings.TrimRight(b.String(), "\n"))
}

func Markdown(md string) string {
	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(100))
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return strings.TrimRight(out, "\n")
}

func KeyHelp() string {
	return DefaultTheme().MutedStyle().Render("up/down: move  enter: details  esc: back  q: quit")
}
