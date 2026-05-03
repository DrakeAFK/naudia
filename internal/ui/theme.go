package ui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Primary lipgloss.Color
	Muted   lipgloss.Color
	Success lipgloss.Color
	Warn    lipgloss.Color
	Danger  lipgloss.Color
	Border  lipgloss.Color
}

func DefaultTheme() Theme {
	return Theme{
		Primary: lipgloss.Color("69"),
		Muted:   lipgloss.Color("245"),
		Success: lipgloss.Color("35"),
		Warn:    lipgloss.Color("214"),
		Danger:  lipgloss.Color("196"),
		Border:  lipgloss.Color("240"),
	}
}

func (t Theme) Title() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
}

func (t Theme) MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted)
}

func (t Theme) Box() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 2)
}

func (t Theme) DangerStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Danger).Bold(true)
}

func (t Theme) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Success)
}
