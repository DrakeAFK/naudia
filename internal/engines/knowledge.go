package engines

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DrakeAFK/naudia/internal/proposals"
)

func (r Runner) Decisions(ctx context.Context, project string) (Report, *proposals.Proposal, error) {
	return r.extractKnowledge(ctx, "Decisions", project, []string{"decided", "decision:"}, "Knowledge/DECISIONS.md", proposals.TypeProjectCompile)
}

func (r Runner) Questions(ctx context.Context, project string) (Report, *proposals.Proposal, error) {
	return r.extractKnowledge(ctx, "Questions", project, []string{"?", "question:"}, "Knowledge/QUESTIONS.md", proposals.TypeProjectCompile)
}

func (r Runner) Doctor(ctx context.Context) (Report, error) {
	status, err := r.Store.Status(ctx, r.Config.Vault.Path)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		Title:   "Doctor",
		Summary: "Naudia checked the local environment and vault metadata.",
		Lines: []string{
			fmt.Sprintf("Vault: %s", status.VaultName),
			fmt.Sprintf("Notes indexed: %d", status.Notes),
			fmt.Sprintf("Database: %s", status.DatabasePath),
			fmt.Sprintf("Embeddings stored: %d", status.EmbeddingsStored),
			fmt.Sprintf("Vector chunks: %d", status.VectorIndexed),
			fmt.Sprintf("Semantic search: %s", semanticSearchStatus(status.VectorAvailable, status.EmbeddingsStored, status.VectorIndexed, r.Config.Index.UseEmbeddings)),
			fmt.Sprintf("Pending proposals: %d", status.PendingProposals),
		},
	}
	if status.Notes == 0 {
		report.Issues = append(report.Issues, Issue{Category: "Index", Severity: "medium", Description: "No indexed notes were found.", SuggestedAction: "Run naudia scan."})
	}
	if r.AI == nil || r.AI.HealthCheck(ctx) != nil {
		report.Issues = append(report.Issues, Issue{Category: "Ollama", Severity: "medium", Description: "Ollama is unavailable.", SuggestedAction: "Start Ollama and pull the configured models."})
	}
	if r.Config.Index.UseEmbeddings && status.Notes > 0 && status.EmbeddingsStored == 0 {
		report.Issues = append(report.Issues, Issue{Category: "Embeddings", Severity: "low", Description: "No stored embeddings were found. Semantic candidates are inactive until embeddings are created.", SuggestedAction: "Run naudia scan after Ollama is online, or use naudia scan --no-embeddings if keyword-only operation is intentional."})
	}
	if status.VectorAvailable && status.EmbeddingsStored > 0 && status.VectorIndexed == 0 {
		report.Issues = append(report.Issues, Issue{Category: "Vector Search", Severity: "low", Description: "sqlite-vec is available, but no stored embeddings have been copied into the native vector table yet.", SuggestedAction: "Run naudia scan. Naudia will keep using Go cosine fallback until the native vector table is populated."})
	}
	return report, nil
}

func semanticSearchStatus(sqliteVec bool, embeddings int, vectorIndexed int, useEmbeddings bool) string {
	if !useEmbeddings {
		return "disabled by config"
	}
	if embeddings == 0 {
		if sqliteVec {
			return "sqlite-vec native KNN ready; no stored embeddings yet"
		}
		return "not active; no stored embeddings yet"
	}
	if sqliteVec {
		if vectorIndexed > 0 {
			return "sqlite-vec native KNN over stored embeddings"
		}
		return "Go cosine fallback over stored embeddings; sqlite-vec ready"
	}
	return "Go cosine fallback over stored embeddings"
}

func (r Runner) extractKnowledge(ctx context.Context, title, project string, needles []string, outputPath string, typ proposals.ProposalType) (Report, *proposals.Proposal, error) {
	chunks, err := r.Store.ListChunks(ctx, r.VaultID)
	if err != nil {
		return Report{}, nil, err
	}
	type sourceLine struct {
		Path string
		Line string
	}
	var matches []sourceLine
	for _, chunk := range chunks {
		if project != "" && !strings.Contains(strings.ToLower(chunk.NotePath+" "+chunk.Content), strings.ToLower(project)) {
			continue
		}
		for _, line := range strings.Split(chunk.Content, "\n") {
			trim := strings.TrimSpace(line)
			if trim == "" {
				continue
			}
			lower := strings.ToLower(trim)
			for _, needle := range needles {
				if strings.Contains(lower, needle) {
					matches = append(matches, sourceLine{Path: chunk.NotePath, Line: trim})
					break
				}
			}
		}
	}
	report := Report{Title: title, Summary: fmt.Sprintf("Naudia found %d sourced %s.", len(matches), strings.ToLower(title))}
	for _, match := range matches {
		report.Lines = append(report.Lines, fmt.Sprintf("%s: %s", match.Path, match.Line))
	}
	if len(matches) == 0 {
		return report, nil, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	for _, match := range matches {
		fmt.Fprintf(&b, "- %s (source: [[%s]])\n", match.Line, strings.TrimSuffix(match.Path, ".md"))
	}
	proposal := &proposals.Proposal{
		Type:      typ,
		Title:     "Create " + strings.ToLower(title) + " index",
		Summary:   "Create a sourced index from selected vault lines.",
		RiskLevel: proposals.RiskLow,
		CreatedAt: time.Now().UTC(),
		Actions: []proposals.ProposalAction{{
			ID:      "create-" + strings.ToLower(title),
			Kind:    proposals.ActionCreateNote,
			Path:    outputPath,
			Content: b.String(),
		}},
	}
	return report, proposal, nil
}
