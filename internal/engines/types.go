package engines

import (
	"github.com/drakeafk/naudia/internal/ai"
	"github.com/drakeafk/naudia/internal/config"
	"github.com/drakeafk/naudia/internal/db"
)

type Runner struct {
	Config  config.Config
	Store   *db.DB
	AI      *ai.OllamaClient
	VaultID int64
}

type Issue struct {
	Category        string   `json:"category"`
	Severity        string   `json:"severity"`
	Description     string   `json:"description"`
	SourceNotes     []string `json:"source_notes"`
	SuggestedAction string   `json:"suggested_action"`
}

type ReportDetail struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Report struct {
	Title      string         `json:"title"`
	Summary    string         `json:"summary"`
	Details    []ReportDetail `json:"details,omitempty"`
	Issues     []Issue        `json:"issues"`
	Lines      []string       `json:"lines"`
	ReportPath string         `json:"report_path,omitempty"`
}
