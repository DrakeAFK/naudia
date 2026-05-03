package ui

import (
	"fmt"

	"github.com/drakeafk/naudia/internal/db"
)

func StatusCard(st db.Status, ollamaStatus, chatModel, embeddingModel, obsidianURI, obsidianCLI string) string {
	vector := "keyword search only"
	if st.VectorAvailable {
		vector = "sqlite-vec native KNN ready"
		if st.EmbeddingsStored > 0 {
			vector = "sqlite-vec native KNN"
		}
	} else if st.EmbeddingsStored > 0 {
		vector = "Go cosine fallback"
	}
	return Card("Naudia Status", [][2]string{
		{"Vault", st.VaultName},
		{"Path", st.VaultPath},
		{"Notes", fmt.Sprintf("%d", st.Notes)},
		{"Last scan", emptyDash(st.LastScan)},
		{"Database", st.DatabasePath},
		{"Ollama", ollamaStatus},
		{"Chat model", chatModel},
		{"Embedding model", embeddingModel},
		{"Embedded chunks", fmt.Sprintf("%d", st.EmbeddingsStored)},
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
