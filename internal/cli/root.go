package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/drakeafk/naudia/internal/ai"
	"github.com/drakeafk/naudia/internal/app"
	"github.com/drakeafk/naudia/internal/config"
	"github.com/drakeafk/naudia/internal/contextpack"
	"github.com/drakeafk/naudia/internal/db"
	"github.com/drakeafk/naudia/internal/engines"
	"github.com/drakeafk/naudia/internal/obsidian"
	"github.com/drakeafk/naudia/internal/proposals"
	"github.com/drakeafk/naudia/internal/ui"
	"github.com/drakeafk/naudia/internal/util"
	"github.com/drakeafk/naudia/internal/vault"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	vaultPath  string
	configPath string
	output     string
	debug      bool
	json       bool
	markdown   bool
}

var rootOpts rootOptions

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "naudia",
		Short:         "Naudia is a local-first AI steward for Obsidian.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringVar(&rootOpts.vaultPath, "vault", "", "Obsidian vault path")
	cmd.PersistentFlags().StringVar(&rootOpts.configPath, "config", "", "Naudia config path")
	cmd.PersistentFlags().StringVarP(&rootOpts.output, "output", "o", "", "output format: pretty, json, markdown, quiet")
	cmd.PersistentFlags().BoolVar(&rootOpts.debug, "debug", false, "enable debug logging")
	cmd.PersistentFlags().BoolVar(&rootOpts.json, "json", false, "write JSON output")
	cmd.PersistentFlags().BoolVar(&rootOpts.markdown, "markdown", false, "write Markdown output")
	cmd.AddCommand(
		initCmd(),
		statusCmd(),
		scanCmd(),
		reviewCmd(),
		dailyCmd(),
		projectCmd(),
		linksCmd(),
		tasksCmd(),
		decisionsCmd(),
		questionsCmd(),
		structureCmd(),
		templatesCmd(),
		doctorCmd(),
		proposalsCmd(),
		showCmd(),
		applyCmd(),
		rejectCmd(),
		rollbackCmd("rollback"),
		rollbackCmd("undo"),
		askCmd(),
	)
	return cmd
}

func withApp(cmd *cobra.Command, fn func(context.Context, *app.App) error) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	a, err := app.New(ctx, app.Options{VaultPath: rootOpts.vaultPath, ConfigPath: rootOpts.configPath, Debug: rootOpts.debug})
	if err != nil {
		return err
	}
	defer a.Close()
	return fn(ctx, a)
}

func runner(ctx context.Context, a *app.App) (engines.Runner, error) {
	vaultID, err := a.DB.VaultID(ctx, a.Config.Vault.Path)
	if err != nil {
		return engines.Runner{}, err
	}
	return engines.Runner{Config: a.Config, Store: a.DB, AI: a.AI, VaultID: vaultID}, nil
}

func proposalManager(ctx context.Context, a *app.App) (proposals.Manager, error) {
	vaultID, err := a.DB.VaultID(ctx, a.Config.Vault.Path)
	if err != nil {
		return proposals.Manager{}, err
	}
	return proposals.Manager{VaultPath: a.Config.Vault.Path, VaultID: vaultID, Store: a.DB}, nil
}

func initCmd() *cobra.Command {
	var vaultPath, vaultName, chatModel, embeddingModel string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Naudia for a vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			if vaultPath == "" {
				if rootOpts.vaultPath != "" {
					vaultPath = rootOpts.vaultPath
				} else {
					wd, _ := os.Getwd()
					vaultPath = promptDefault(cmd, "Vault path", wd)
				}
			}
			abs, err := filepath.Abs(vaultPath)
			if err != nil {
				return err
			}
			info, err := os.Stat(abs)
			if err != nil {
				return fmt.Errorf("vault path is not available: %w", err)
			}
			if !info.IsDir() {
				return fmt.Errorf("vault path is not a directory: %s", abs)
			}
			cfg := config.Default()
			cfg.Vault.Path = abs
			if vaultName != "" {
				cfg.Vault.Name = vaultName
			} else {
				cfg.Vault.Name = obsidian.DeriveVaultName(abs)
				if isTerminalStdin() {
					cfg.Vault.Name = promptDefault(cmd, "Vault name", cfg.Vault.Name)
				}
			}
			if chatModel != "" {
				cfg.Ollama.ChatModel = chatModel
			}
			if embeddingModel != "" {
				cfg.Ollama.EmbeddingModel = embeddingModel
			}
			if err := util.EnsureNaudiaDirs(abs); err != nil {
				return err
			}
			cfgPath := filepath.Join(abs, ".naudia", "config.toml")
			if err := config.Save(cfgPath, cfg); err != nil {
				return err
			}
			dbPath := filepath.Join(abs, cfg.Index.DatabasePath)
			store, err := db.Open(ctx, dbPath, nil)
			if err != nil {
				return err
			}
			defer store.Close()
			if _, err := store.UpsertVault(ctx, cfg.Vault.Name, cfg.Vault.Path, cfg.Obsidian.UseCLI, cfg.Obsidian.UseURI); err != nil {
				return err
			}
			ollama := "offline"
			modelSummary := "unavailable"
			client := ai.NewOllamaClient(cfg.Ollama.Host, cfg.Ollama.ChatModel, cfg.Ollama.EmbeddingModel)
			if err := client.HealthCheck(ctx); err == nil {
				ollama = "online"
				if models, err := client.ListModels(ctx); err == nil {
					modelSummary = modelAvailability(models, cfg.Ollama.ChatModel, cfg.Ollama.EmbeddingModel)
				}
			}
			cliStatus := "unavailable"
			if obsidian.CLIAvailable(cfg.Obsidian.CLICommand) {
				cliStatus = "available"
			}
			if format(cmd) == "quiet" {
				return nil
			}
			if !obsidian.LooksLikeVault(abs) {
				fmt.Fprintln(cmd.OutOrStdout(), ui.ErrorCard("Vault Has No Markdown Notes", "Naudia initialized successfully, but no Markdown files were found yet.", "Add notes or run scan after Obsidian creates them."))
			}
			fmt.Fprintln(cmd.OutOrStdout(), ui.Card("Naudia Initialized", [][2]string{
				{"Vault", cfg.Vault.Name},
				{"Path", cfg.Vault.Path},
				{"Config", cfgPath},
				{"Database", dbPath},
				{"Ollama", ollama},
				{"Models", modelSummary},
				{"sqlite-vec", vectorStatus(store.VectorAvailable)},
				{"Obsidian URI", enabledDisabled(cfg.Obsidian.UseURI)},
				{"Obsidian CLI", cliStatus},
			}))
			return nil
		},
	}
	cmd.Flags().StringVar(&vaultPath, "vault", "", "vault path")
	cmd.Flags().StringVar(&vaultName, "vault-name", "", "vault name for Obsidian URIs")
	cmd.Flags().StringVar(&chatModel, "model", "", "Ollama chat model")
	cmd.Flags().StringVar(&embeddingModel, "embedding-model", "", "Ollama embedding model")
	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show Naudia environment status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				st, err := a.DB.Status(ctx, a.Config.Vault.Path)
				if err != nil {
					return err
				}
				ollamaStatus := "offline"
				if err := a.AI.HealthCheck(ctx); err == nil {
					ollamaStatus = "online"
				}
				cliStatus := "unavailable"
				if obsidian.CLIAvailable(a.Config.Obsidian.CLICommand) {
					cliStatus = "available"
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, map[string]any{"status": st, "ollama": ollamaStatus, "obsidian_uri": a.Config.Obsidian.UseURI, "obsidian_cli": cliStatus})
				}
				uriStatus := "disabled"
				if a.Config.Obsidian.UseURI {
					uriStatus = "enabled"
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.StatusCard(st, ollamaStatus, a.Config.Ollama.ChatModel, a.Config.Ollama.EmbeddingModel, uriStatus, cliStatus))
				return nil
			})
		},
	}
}

func scanCmd() *cobra.Command {
	var noEmbeddings, force, quiet bool
	var folder string
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan and index the vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				stats, warnings, err := scanVault(ctx, a, scanOptions{NoEmbeddings: noEmbeddings, Force: force, Folder: folder})
				if err != nil {
					return err
				}
				if quiet || format(cmd) == "quiet" {
					return nil
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, map[string]any{"stats": stats, "warnings": warnings})
				}
				rows := [][2]string{
					{"Notes indexed", strconv.Itoa(stats.NotesIndexed)},
					{"Notes skipped", strconv.Itoa(stats.NotesSkipped)},
					{"Chunks indexed", strconv.Itoa(stats.ChunksIndexed)},
					{"Embeddings", strconv.Itoa(stats.EmbeddingsStored)},
				}
				if len(warnings) > 0 {
					rows = append(rows, [2]string{"Warnings", strings.Join(warnings, "; ")})
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.Card("Scan Complete", rows))
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&noEmbeddings, "no-embeddings", false, "skip embedding generation")
	cmd.Flags().BoolVar(&force, "force", false, "rebuild unchanged note details")
	cmd.Flags().StringVar(&folder, "folder", "", "scan a vault folder")
	cmd.Flags().BoolVar(&quiet, "quiet", false, "suppress output")
	return cmd
}

type scanOptions struct {
	NoEmbeddings bool
	Force        bool
	Folder       string
}

func scanVault(ctx context.Context, a *app.App, opts scanOptions) (db.ScanStats, []string, error) {
	vaultID, err := a.DB.VaultID(ctx, a.Config.Vault.Path)
	if err != nil {
		return db.ScanStats{}, nil, err
	}
	scanner := vault.NewScanner(a.Config.Vault.Path, a.Config.Ignore.Patterns).WithFolder(opts.Folder)
	result, err := scanner.Scan()
	if err != nil {
		return db.ScanStats{}, result.Warnings, err
	}
	stats, err := a.DB.IndexScan(ctx, vaultID, result, a.Config.Index.ChunkSize, a.Config.Index.ChunkOverlap, opts.Force)
	if err != nil {
		return stats, result.Warnings, err
	}
	if !opts.NoEmbeddings && a.Config.Index.UseEmbeddings {
		if err := a.AI.HealthCheck(ctx); err == nil {
			for {
				chunks, err := a.DB.ChunksNeedingEmbeddings(ctx, vaultID, a.Config.Ollama.EmbeddingModel, 32)
				if err != nil {
					return stats, result.Warnings, err
				}
				if len(chunks) == 0 {
					break
				}
				inputs := make([]string, 0, len(chunks))
				for _, chunk := range chunks {
					inputs = append(inputs, chunk.Content)
				}
				embeddings, err := a.AI.Embed(ctx, inputs)
				if err != nil {
					result.Warnings = append(result.Warnings, "embedding generation failed: "+err.Error())
					break
				}
				for i, embedding := range embeddings {
					if err := a.DB.StoreEmbedding(ctx, chunks[i], a.Config.Ollama.EmbeddingModel, embedding.Vector); err != nil {
						return stats, result.Warnings, err
					}
					stats.EmbeddingsStored++
				}
				if len(chunks) < 32 {
					break
				}
			}
		} else {
			result.Warnings = append(result.Warnings, "Ollama unavailable; embeddings skipped")
		}
	}
	return stats, result.Warnings, nil
}

func reviewCmd() *cobra.Command {
	var noAI, interactive bool
	var folder string
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Review vault health and prepare proposals",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				_, _, _ = scanVault(ctx, a, scanOptions{NoEmbeddings: true, Folder: folder})
				r, err := runner(ctx, a)
				if err != nil {
					return err
				}
				report, props, err := r.Review(ctx, noAI, folder)
				if err != nil {
					return err
				}
				pm, err := proposalManager(ctx, a)
				if err != nil {
					return err
				}
				for _, p := range props {
					_, _ = pm.Save(ctx, p)
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, map[string]any{"report": report, "proposals": props})
				}
				if format(cmd) == "markdown" {
					fmt.Fprintln(cmd.OutOrStdout(), ui.Markdown(ui.ReportView(report)))
					return nil
				}
				if interactive {
					_, err := tea.NewProgram(ui.NewReviewModel(report, props)).Run()
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.ReportView(report))
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&noAI, "no-ai", false, "use deterministic review only")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "open interactive review UI")
	cmd.Flags().StringVar(&folder, "folder", "", "review a folder")
	cmd.Flags().Bool("today", false, "focus today's notes")
	cmd.Flags().Bool("week", false, "focus this week")
	cmd.Flags().Bool("templates", false, "focus templates")
	cmd.Flags().Bool("orphans", false, "focus orphan notes")
	cmd.Flags().Bool("tasks", false, "focus tasks")
	cmd.Flags().Bool("structure", false, "focus structure")
	return cmd
}

func dailyCmd() *cobra.Command {
	var dateText string
	var apply, showContext, interactive bool
	cmd := &cobra.Command{
		Use:   "daily",
		Short: "Distill a daily note",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				_, _, _ = scanVault(ctx, a, scanOptions{NoEmbeddings: true})
				r, err := runner(ctx, a)
				if err != nil {
					return err
				}
				result, err := r.Daily(ctx, dateText, showContext)
				if err != nil {
					return err
				}
				pm, err := proposalManager(ctx, a)
				if err != nil {
					return err
				}
				id, err := pm.Save(ctx, result.Proposal)
				if err != nil {
					return err
				}
				result.Proposal.ID = id
				if apply {
					if _, err := pm.Apply(ctx, id); err != nil {
						return err
					}
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, result)
				}
				if showContext {
					fmt.Fprintln(cmd.OutOrStdout(), result.Review)
					fmt.Fprintln(cmd.OutOrStdout(), "\nContext:\n"+contextpack.Table(result.Context))
					return nil
				}
				if interactive {
					_, err := tea.NewProgram(ui.NewDailyModel("Daily Review "+result.Date, result.Review, 100, 28)).Run()
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.Markdown(result.Review))
				fmt.Fprintf(cmd.OutOrStdout(), "\nProposal %d prepared. Review with `naudia show %d`.\n", id, id)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&dateText, "date", "", "daily note date")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply generated proposal")
	cmd.Flags().BoolVar(&showContext, "show-context", false, "show selected context")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "open interactive daily UI")
	cmd.Flags().Bool("week", false, "distill the current week")
	cmd.Flags().Bool("create-permanent-notes", false, "propose permanent notes")
	cmd.Flags().Bool("move-tasks", false, "propose task moves")
	return cmd
}

func projectCmd() *cobra.Command {
	var apply, showContext bool
	var folder, generate string
	cmd := &cobra.Command{
		Use:   "project <name>",
		Short: "Compile project memory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				_, _, _ = scanVault(ctx, a, scanOptions{NoEmbeddings: false})
				r, err := runner(ctx, a)
				if err != nil {
					return err
				}
				result, err := r.Project(ctx, args[0])
				if err != nil {
					return err
				}
				_ = folder
				_ = generate
				pm, err := proposalManager(ctx, a)
				if err != nil {
					return err
				}
				id, err := pm.Save(ctx, result.Proposal)
				if err != nil {
					return err
				}
				if apply {
					if _, err := pm.Apply(ctx, id); err != nil {
						return err
					}
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, result)
				}
				if showContext {
					fmt.Fprintln(cmd.OutOrStdout(), contextpack.Table(result.Context))
					return nil
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.Card("Project Memory", [][2]string{{"Project", result.Name}, {"Context items", strconv.Itoa(len(result.Context.Items))}, {"Proposal", fmt.Sprintf("%d", id)}}))
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&apply, "apply", false, "apply generated proposal")
	cmd.Flags().BoolVar(&showContext, "show-context", false, "show context pack")
	cmd.Flags().StringVar(&folder, "folder", "", "target project folder")
	cmd.Flags().StringVar(&generate, "generate", "", "comma-separated files to generate")
	return cmd
}

func linksCmd() *cobra.Command {
	var note, folder string
	var orphans, apply, interactive bool
	cmd := &cobra.Command{
		Use:   "links",
		Short: "Suggest backlinks and graph improvements",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				_, _, _ = scanVault(ctx, a, scanOptions{NoEmbeddings: true})
				r, err := runner(ctx, a)
				if err != nil {
					return err
				}
				if orphans {
					list, err := r.OrphanNotes(ctx)
					if err != nil {
						return err
					}
					if format(cmd) == "json" {
						return writeJSON(cmd, list)
					}
					fmt.Fprintln(cmd.OutOrStdout(), strings.Join(list, "\n"))
					return nil
				}
				result, err := r.Links(ctx, note, folder)
				if err != nil {
					return err
				}
				var id int64
				if result.Proposal != nil {
					pm, err := proposalManager(ctx, a)
					if err != nil {
						return err
					}
					id, err = pm.Save(ctx, result.Proposal)
					if err != nil {
						return err
					}
					if apply {
						if _, err := pm.Apply(ctx, id); err != nil {
							return err
						}
					}
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, result)
				}
				if interactive {
					_, err := tea.NewProgram(ui.NewLinksModel(result.Suggestions)).Run()
					return err
				}
				rows := [][2]string{{"Suggestions", strconv.Itoa(len(result.Suggestions))}}
				if id > 0 {
					rows = append(rows, [2]string{"Proposal", fmt.Sprintf("%d", id)})
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.Card("Link Suggestions", rows))
				for _, s := range result.Suggestions {
					fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s line %d (%s)\n", s.SourceNote, s.TargetNote, s.LineNumber, s.Confidence)
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&note, "note", "", "focus a note")
	cmd.Flags().StringVar(&folder, "folder", "", "focus a folder")
	cmd.Flags().BoolVar(&orphans, "orphans", false, "list orphan notes")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply generated proposal")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "open interactive links UI")
	return cmd
}

func tasksCmd() *cobra.Command {
	var project string
	var apply bool
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Extract and group tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				_, _, _ = scanVault(ctx, a, scanOptions{NoEmbeddings: true})
				r, err := runner(ctx, a)
				if err != nil {
					return err
				}
				report, prop, err := r.Tasks(ctx, project)
				if err != nil {
					return err
				}
				var id int64
				if prop != nil {
					pm, _ := proposalManager(ctx, a)
					id, _ = pm.Save(ctx, prop)
					if apply && id > 0 {
						_, err = pm.Apply(ctx, id)
						if err != nil {
							return err
						}
					}
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, map[string]any{"report": report, "proposal_id": id})
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.ReportView(report))
				if id > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "\nProposal %d prepared.\n", id)
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "filter by project")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply generated proposal")
	cmd.Flags().Bool("today", false, "focus today")
	cmd.Flags().Bool("week", false, "focus week")
	cmd.Flags().Bool("include-inferred", false, "include inferred tasks")
	return cmd
}

func decisionsCmd() *cobra.Command {
	var project string
	var apply bool
	cmd := &cobra.Command{
		Use:   "decisions",
		Short: "Extract sourced decisions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				_, _, _ = scanVault(ctx, a, scanOptions{NoEmbeddings: true})
				r, err := runner(ctx, a)
				if err != nil {
					return err
				}
				report, prop, err := r.Decisions(ctx, project)
				if err != nil {
					return err
				}
				return writeReportAndMaybeProposal(cmd, ctx, a, report, prop, apply)
			})
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "filter by project")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply generated proposal")
	return cmd
}

func questionsCmd() *cobra.Command {
	var project string
	var apply bool
	cmd := &cobra.Command{
		Use:   "questions",
		Short: "Extract unresolved questions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				_, _, _ = scanVault(ctx, a, scanOptions{NoEmbeddings: true})
				r, err := runner(ctx, a)
				if err != nil {
					return err
				}
				report, prop, err := r.Questions(ctx, project)
				if err != nil {
					return err
				}
				return writeReportAndMaybeProposal(cmd, ctx, a, report, prop, apply)
			})
		},
	}
	cmd.Flags().StringVar(&project, "project", "", "filter by project")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply generated proposal")
	return cmd
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check Naudia environment health",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				r, err := runner(ctx, a)
				if err != nil {
					return err
				}
				report, err := r.Doctor(ctx)
				if err != nil {
					return err
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, report)
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.ReportView(report))
				return nil
			})
		},
	}
}

func writeReportAndMaybeProposal(cmd *cobra.Command, ctx context.Context, a *app.App, report engines.Report, prop *proposals.Proposal, apply bool) error {
	var id int64
	if prop != nil {
		pm, err := proposalManager(ctx, a)
		if err != nil {
			return err
		}
		var saveErr error
		id, saveErr = pm.Save(ctx, prop)
		if saveErr != nil {
			return saveErr
		}
		if apply {
			if _, err := pm.Apply(ctx, id); err != nil {
				return err
			}
		}
	}
	if format(cmd) == "json" {
		return writeJSON(cmd, map[string]any{"report": report, "proposal_id": id})
	}
	fmt.Fprintln(cmd.OutOrStdout(), ui.ReportView(report))
	if id > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nProposal %d prepared.\n", id)
	}
	return nil
}

func structureCmd() *cobra.Command {
	var propose, apply, interactive bool
	cmd := &cobra.Command{
		Use:   "structure",
		Short: "Analyze vault structure",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				_, _, _ = scanVault(ctx, a, scanOptions{NoEmbeddings: true})
				r, _ := runner(ctx, a)
				report, prop, err := r.Structure(ctx, propose || apply)
				if err != nil {
					return err
				}
				var id int64
				if prop != nil {
					pm, _ := proposalManager(ctx, a)
					id, err = pm.Save(ctx, prop)
					if err != nil {
						return err
					}
					if apply {
						return fmt.Errorf("structure moves are high risk; review proposal %d and run naudia apply %d", id, id)
					}
				}
				if interactive {
					_, err := tea.NewProgram(ui.NewReviewModel(report, nil)).Run()
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.ReportView(report))
				if id > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "\nProposal %d prepared.\n", id)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&propose, "propose", false, "prepare move proposals")
	cmd.Flags().BoolVar(&apply, "apply", false, "prepare apply flow")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "open interactive UI")
	return cmd
}

func templatesCmd() *cobra.Command {
	var folder string
	var apply bool
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Analyze and improve templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				r, _ := runner(ctx, a)
				report, prop, err := r.Templates(ctx, folder)
				if err != nil {
					return err
				}
				var id int64
				if prop != nil {
					pm, _ := proposalManager(ctx, a)
					id, err = pm.Save(ctx, prop)
					if err != nil {
						return err
					}
					if apply {
						if _, err := pm.Apply(ctx, id); err != nil {
							return err
						}
					}
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.ReportView(report))
				if id > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "\nProposal %d prepared.\n", id)
				}
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&folder, "folder", "Templates", "template folder")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply generated proposal")
	cmd.Flags().Bool("project", false, "focus project template")
	cmd.Flags().Bool("daily", false, "focus daily template")
	return cmd
}

func proposalsCmd() *cobra.Command {
	var all, pending, applied, interactive bool
	cmd := &cobra.Command{
		Use:   "proposals",
		Short: "List proposals",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				vaultID, _ := a.DB.VaultID(ctx, a.Config.Vault.Path)
				status := "pending"
				if applied {
					status = "applied"
				}
				if pending {
					status = "pending"
				}
				records, err := a.DB.ListProposals(ctx, vaultID, status, all)
				if err != nil {
					return err
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, records)
				}
				if interactive {
					_, err := tea.NewProgram(ui.NewProposalListModel(records)).Run()
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.ProposalTable(records))
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "show all proposals")
	cmd.Flags().BoolVar(&pending, "pending", false, "show pending proposals")
	cmd.Flags().BoolVar(&applied, "applied", false, "show applied proposals")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "open interactive list")
	return cmd
}

func showCmd() *cobra.Command {
	var patchOnly bool
	cmd := &cobra.Command{
		Use:   "show <proposal-id>",
		Short: "Show proposal details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				pm, _ := proposalManager(ctx, a)
				p, rec, err := pm.Load(ctx, id)
				if err != nil {
					return err
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, p)
				}
				if patchOnly {
					fmt.Fprintln(cmd.OutOrStdout(), rec.PatchText)
					return nil
				}
				fmt.Fprintln(cmd.OutOrStdout(), proposals.RenderDetails(*p, rec))
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&patchOnly, "patch", false, "show patch only")
	return cmd
}

func applyCmd() *cobra.Command {
	var yes, interactive bool
	cmd := &cobra.Command{
		Use:   "apply <proposal-id|all>",
		Short: "Apply a proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				pm, _ := proposalManager(ctx, a)
				ids, err := proposalIDs(ctx, a, args[0])
				if err != nil {
					return err
				}
				for _, id := range ids {
					p, rec, err := pm.Load(ctx, id)
					if err != nil {
						return err
					}
					if interactive {
						_, _ = tea.NewProgram(ui.NewDiffModel(rec.PatchText, 100, 28)).Run()
					}
					if p.RequiresConfirmation || p.RiskLevel == proposals.RiskHigh || !yes {
						ok, err := ui.ConfirmID(os.Stdin, cmd.OutOrStdout(), id, riskMessage(p, yes))
						if err != nil {
							return err
						}
						if !ok {
							return fmt.Errorf("proposal %d was not confirmed", id)
						}
					}
					if git := util.GitStatusForPath(a.Config.Vault.Path); git.InRepo && git.Dirty && (p.RiskLevel == proposals.RiskHigh || len(p.Actions) > 5) {
						fmt.Fprintln(cmd.OutOrStdout(), ui.ErrorCard("Git Working Tree Has Changes", "Naudia detected existing Git changes before applying this proposal.", "Review `git status --short` if this proposal touches many files."))
					}
					result, err := pm.Apply(ctx, id)
					if err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Applied proposal %d (%d actions). Roll back with `naudia rollback %d`.\n", id, len(result.Applied), id)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "approve non-high-risk proposal")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "show interactive diff before applying")
	return cmd
}

func rejectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reject <proposal-id|all>",
		Short: "Reject a proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				ids, err := proposalIDs(ctx, a, args[0])
				if err != nil {
					return err
				}
				for _, id := range ids {
					if err := a.DB.UpdateProposalStatus(ctx, id, "rejected"); err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Rejected proposal %d.\n", id)
				}
				return nil
			})
		},
	}
	return cmd
}

func rollbackCmd(name string) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   name + " <proposal-id>",
		Short: "Roll back an applied proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				if force {
					ok, err := ui.ConfirmID(os.Stdin, cmd.OutOrStdout(), id, "Force rollback may overwrite manual edits made after this proposal was applied.")
					if err != nil {
						return err
					}
					if !ok {
						return errors.New("force rollback was not confirmed")
					}
				}
				pm, _ := proposalManager(ctx, a)
				result, err := pm.Rollback(ctx, id, force)
				if err != nil {
					if len(result.Conflicts) > 0 {
						fmt.Fprintln(cmd.OutOrStdout(), ui.ErrorCard("Rollback Conflict", "Naudia could not safely roll back every file.", "Conflict details were written to .naudia/conflicts/."))
					}
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Rolled back proposal %d (%d files).\n", id, len(result.RolledBack))
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "restore previous content even if files drifted")
	return cmd
}

func askCmd() *cobra.Command {
	var showContext bool
	cmd := &cobra.Command{
		Use:   "ask <question>",
		Short: "Ask a read-only question over conservative vault context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				_, _, _ = scanVault(ctx, a, scanOptions{NoEmbeddings: false})
				r, _ := runner(ctx, a)
				result, err := r.Ask(ctx, args[0])
				if err != nil {
					return err
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, result)
				}
				fmt.Fprintln(cmd.OutOrStdout(), result.Answer)
				if showContext {
					fmt.Fprintln(cmd.OutOrStdout(), "\nContext:\n"+contextpack.Table(result.Context))
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&showContext, "show-context", false, "show context pack")
	return cmd
}

func proposalIDs(ctx context.Context, a *app.App, arg string) ([]int64, error) {
	if arg != "all" {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return nil, err
		}
		return []int64{id}, nil
	}
	vaultID, err := a.DB.VaultID(ctx, a.Config.Vault.Path)
	if err != nil {
		return nil, err
	}
	records, err := a.DB.ListProposals(ctx, vaultID, "pending", false)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}
	return ids, nil
}

func vectorStatus(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "unavailable"
}

func format(cmd *cobra.Command) string {
	if rootOpts.output != "" {
		return rootOpts.output
	}
	if rootOpts.json {
		return "json"
	}
	if rootOpts.markdown {
		return "markdown"
	}
	return "pretty"
}

func writeJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func promptDefault(cmd *cobra.Command, label, fallback string) string {
	if !isTerminalStdin() {
		return fallback
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s [%s]: ", label, fallback)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fallback
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return fallback
	}
	return line
}

func isTerminalStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func modelAvailability(models []string, chatModel, embeddingModel string) string {
	available := map[string]bool{}
	for _, model := range models {
		available[model] = true
	}
	chat := "missing"
	if available[chatModel] {
		chat = "available"
	}
	embedding := "missing"
	if available[embeddingModel] {
		embedding = "available"
	}
	return fmt.Sprintf("%s: %s, %s: %s", chatModel, chat, embeddingModel, embedding)
}

func enabledDisabled(v bool) string {
	if v {
		return "enabled"
	}
	return "disabled"
}

func riskMessage(p *proposals.Proposal, yes bool) string {
	if p.RiskLevel == proposals.RiskHigh {
		return "This proposal may significantly change your vault."
	}
	if yes {
		return "This proposal requires confirmation."
	}
	return "Review this proposal before applying."
}
