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
		b.WriteString(formatCardRow(row[0], row[1]))
	}
	return theme.Box().Render(strings.TrimRight(b.String(), "\n"))
}

func formatCardRow(label, value string) string {
	const labelWidth = 16
	const valueWidth = 84
	lines := wrapText(value, valueWidth)
	if len(lines) == 0 {
		lines = []string{""}
	}
	var b strings.Builder
	for i, line := range lines {
		if i == 0 {
			fmt.Fprintf(&b, "%-*s %s\n", labelWidth, label, line)
			continue
		}
		fmt.Fprintf(&b, "%-*s %s\n", labelWidth, "", line)
	}
	return b.String()
}

func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var out []string
	for _, paragraph := range strings.Split(s, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := ""
		for _, word := range words {
			parts := splitLongWord(word, width)
			for _, part := range parts {
				if line == "" {
					line = part
					continue
				}
				if len(line)+1+len(part) <= width {
					line += " " + part
					continue
				}
				out = append(out, line)
				line = part
			}
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func splitLongWord(word string, width int) []string {
	if len(word) <= width {
		return []string{word}
	}
	var out []string
	for len(word) > width {
		out = append(out, word[:width])
		word = word[width:]
	}
	if word != "" {
		out = append(out, word)
	}
	return out
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
