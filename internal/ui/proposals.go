package ui

import (
	"fmt"
	"strings"

	"github.com/DrakeAFK/naudia/internal/db"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type proposalItem struct {
	record db.ProposalRecord
}

func (i proposalItem) FilterValue() string { return i.record.Title }
func (i proposalItem) Title() string       { return fmt.Sprintf("%d  %s", i.record.ID, i.record.Title) }
func (i proposalItem) Description() string {
	return fmt.Sprintf("%s / %s / %s", i.record.Status, i.record.Type, i.record.Summary)
}

type ProposalListModel struct {
	list list.Model
	done bool
}

func NewProposalListModel(records []db.ProposalRecord) ProposalListModel {
	items := make([]list.Item, 0, len(records))
	for _, record := range records {
		items = append(items, proposalItem{record: record})
	}
	l := list.New(items, list.NewDefaultDelegate(), 90, 24)
	l.Title = "Pending Proposals"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return ProposalListModel{list: l}
}

func (m ProposalListModel) Init() tea.Cmd { return nil }

func (m ProposalListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" {
			m.done = true
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

func (m ProposalListModel) View() string {
	if m.done {
		return ""
	}
	return m.list.View() + "\n" + KeyHelp()
}

func ProposalTable(records []db.ProposalRecord) string {
	if len(records) == 0 {
		return Card("Pending Proposals", [][2]string{{"Status", "No proposals found"}})
	}
	var rows [][2]string
	for _, p := range records {
		value := fmt.Sprintf("%s  %s  %s", truncate(p.Title, 38), p.Status, p.Type)
		if strings.TrimSpace(p.Summary) != "" {
			value += "\n" + p.Summary
		}
		rows = append(rows, [2]string{fmt.Sprintf("#%d", p.ID), value})
	}
	if len(records) > 0 {
		rows = append(rows, [2]string{"Next", fmt.Sprintf("naudia show %d, then naudia apply %d --yes or naudia reject %d", records[0].ID, records[0].ID, records[0].ID)})
	}
	return Card("Pending Proposals", rows)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max-1]) + "."
}
