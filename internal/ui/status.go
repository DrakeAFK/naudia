package ui

import (
	"fmt"

	"github.com/drakeafk/naudia/internal/db"
)

func StatusCard(st db.Status, ollamaStatus, chatModel, embeddingModel, obsidianURI, obsidianCLI string) string {
	vector := "unavailable; semantic suggestions disabled"
	if st.VectorAvailable {
		vector = "sqlite-vec enabled"
	}
	return Card("Naudia Status", [][2]string{
		{"Vault", st.VaultName},
		{"Path", st.VaultPath},
		{"Notes", fmt.Sprintf("%d", st.Notes)},
		{"Last scan", emptyDash(st.LastScan)},
		{"Database", st.DatabasePath},
		{"Ollama", ollamaStatus},
		{"Chat model", chatModel},
		{"Embeddings", embeddingModel},
		{"Vector search", vector},
		{"Obsidian URI", obsidianURI},
		{"Obsidian CLI", obsidianCLI},
		{"Proposals", fmt.Sprintf("%d pending, %d applied", st.PendingProposals, st.AppliedProposals)},
	})
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
