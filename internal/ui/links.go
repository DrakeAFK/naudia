package ui

import (
	"fmt"

	"github.com/DrakeAFK/naudia/internal/engines"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type linkItem struct {
	s engines.LinkSuggestion
}

func (i linkItem) FilterValue() string { return i.s.SourceNote + " " + i.s.TargetNote }
func (i linkItem) Title() string       { return fmt.Sprintf("%s -> %s", i.s.SourceNote, i.s.TargetNote) }
func (i linkItem) Description() string {
	return fmt.Sprintf("line %d / %s / %s", i.s.LineNumber, i.s.Confidence, i.s.Reason)
}

type LinksModel struct {
	list list.Model
}

func NewLinksModel(suggestions []engines.LinkSuggestion) LinksModel {
	items := make([]list.Item, 0, len(suggestions))
	for _, suggestion := range suggestions {
		items = append(items, linkItem{s: suggestion})
	}
	l := list.New(items, list.NewDefaultDelegate(), 90, 24)
	l.Title = "Link Suggestions"
	l.SetFilteringEnabled(true)
	return LinksModel{list: l}
}

func (m LinksModel) Init() tea.Cmd { return nil }

func (m LinksModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m LinksModel) View() string {
	return m.list.View() + "\n" + KeyHelp()
}
