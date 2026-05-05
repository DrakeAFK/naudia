package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DrakeAFK/naudia/internal/ai"
	"github.com/DrakeAFK/naudia/internal/app"
	"github.com/DrakeAFK/naudia/internal/config"
	"github.com/DrakeAFK/naudia/internal/contextpack"
	"github.com/DrakeAFK/naudia/internal/db"
	"github.com/DrakeAFK/naudia/internal/engines"
	"github.com/DrakeAFK/naudia/internal/obsidian"
	"github.com/DrakeAFK/naudia/internal/proposals"
	"github.com/DrakeAFK/naudia/internal/ui"
	"github.com/DrakeAFK/naudia/internal/util"
	"github.com/DrakeAFK/naudia/internal/vault"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	vaultPath  string
	configPath string
	output     string
	model      string
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
	cmd.PersistentFlags().StringVar(&rootOpts.model, "model", "", "Ollama chat model override for this run")
	cmd.PersistentFlags().BoolVar(&rootOpts.debug, "debug", false, "enable debug logging")
	cmd.PersistentFlags().BoolVar(&rootOpts.json, "json", false, "write JSON output")
	cmd.PersistentFlags().BoolVar(&rootOpts.markdown, "markdown", false, "write Markdown output")
	cmd.AddCommand(
		initCmd(),
		statusCmd(),
		modelsCmd(),
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
		versionCmd(),
		proposalsCmd(),
		showCmd(),
		applyCmd(),
		rejectCmd(),
		rollbackCmd("rollback"),
		rollbackCmd("undo"),
		askCmd(),
		chatCmd(),
		noteCmd(),
	)
	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show Naudia version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := map[string]string{
				"name":    app.Name,
				"version": app.Version,
				"commit":  app.Commit,
				"date":    app.Date,
			}
			if format(cmd) == "json" {
				return writeJSON(cmd, info)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\ncommit %s\nbuilt %s\n", app.Name, app.Version, app.Commit, app.Date)
			return nil
		},
	}
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
	if rootOpts.model != "" {
		a.Config.Ollama.ChatModel = rootOpts.model
		a.AI.ChatModel = rootOpts.model
		a.AI.ChatFallbackModels = nil
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
			client := ai.NewOllamaClient(cfg.Ollama.Host, cfg.Ollama.ChatModel, cfg.Ollama.EmbeddingModel, cfg.Ollama.ChatFallbackModels...)
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
				{"Fallbacks", emptyString(strings.Join(cfg.Ollama.ChatFallbackModels, ", "), "-")},
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
					return writeJSON(cmd, map[string]any{"status": st, "ollama": ollamaStatus, "chat_model": a.Config.Ollama.ChatModel, "chat_fallback_models": a.Config.Ollama.ChatFallbackModels, "obsidian_uri": a.Config.Obsidian.UseURI, "obsidian_cli": cliStatus})
				}
				uriStatus := "disabled"
				if a.Config.Obsidian.UseURI {
					uriStatus = "enabled"
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.StatusCard(st, ollamaStatus, a.Config.Ollama.ChatModel, a.Config.Ollama.ChatFallbackModels, a.Config.Ollama.EmbeddingModel, uriStatus, cliStatus))
				return nil
			})
		},
	}
}

func modelsCmd() *cobra.Command {
	var setChat, setEmbedding, addFallback string
	var clearFallbacks bool
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Show or update local Ollama model settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				changed := false
				if setChat = strings.TrimSpace(setChat); setChat != "" {
					a.Config.Ollama.ChatModel = setChat
					changed = true
				}
				if setEmbedding = strings.TrimSpace(setEmbedding); setEmbedding != "" {
					a.Config.Ollama.EmbeddingModel = setEmbedding
					changed = true
				}
				if clearFallbacks {
					a.Config.Ollama.ChatFallbackModels = nil
					changed = true
				}
				if addFallback = strings.TrimSpace(addFallback); addFallback != "" {
					if !stringIn(addFallback, a.Config.Ollama.ChatFallbackModels) {
						a.Config.Ollama.ChatFallbackModels = append(a.Config.Ollama.ChatFallbackModels, addFallback)
						changed = true
					}
				}
				if changed {
					if err := config.Save(a.ConfigPath, a.Config); err != nil {
						return err
					}
					a.AI.ChatModel = a.Config.Ollama.ChatModel
					a.AI.ChatFallbackModels = a.Config.Ollama.ChatFallbackModels
					a.AI.EmbeddingModel = a.Config.Ollama.EmbeddingModel
				}
				ollamaStatus := "offline"
				var models []string
				if err := a.AI.HealthCheck(ctx); err == nil {
					ollamaStatus = "online"
					models, _ = a.AI.ListModels(ctx)
				}
				payload := map[string]any{
					"ollama":               ollamaStatus,
					"chat_model":           a.Config.Ollama.ChatModel,
					"chat_fallback_models": a.Config.Ollama.ChatFallbackModels,
					"embedding_model":      a.Config.Ollama.EmbeddingModel,
					"installed_models":     models,
					"config":               a.ConfigPath,
					"changed":              changed,
				}
				if format(cmd) == "json" {
					return writeJSON(cmd, payload)
				}
				rows := [][2]string{
					{"Ollama", ollamaStatus},
					{"Chat model", a.Config.Ollama.ChatModel},
					{"Fallbacks", emptyString(strings.Join(a.Config.Ollama.ChatFallbackModels, ", "), "-")},
					{"Embedding", a.Config.Ollama.EmbeddingModel},
					{"Config", a.ConfigPath},
				}
				if len(models) > 0 {
					rows = append(rows, [2]string{"Installed", strings.Join(models, ", ")})
				}
				if changed {
					rows = append(rows, [2]string{"Updated", "saved model settings"})
				}
				fmt.Fprintln(cmd.OutOrStdout(), ui.Card("Naudia Models", rows))
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&setChat, "set-chat", "", "persist the primary Ollama chat model")
	cmd.Flags().StringVar(&setEmbedding, "set-embedding", "", "persist the Ollama embedding model")
	cmd.Flags().StringVar(&addFallback, "add-fallback", "", "append a chat fallback model")
	cmd.Flags().BoolVar(&clearFallbacks, "clear-fallbacks", false, "remove configured chat fallback models")
	return cmd
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
					{"Embeddings updated", strconv.Itoa(stats.EmbeddingsStored)},
					{"Total embeddings", strconv.Itoa(stats.TotalEmbeddings)},
					{"Vector chunks", strconv.Itoa(stats.VectorIndexed)},
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
	if st, err := a.DB.Status(ctx, a.Config.Vault.Path); err == nil {
		stats.TotalEmbeddings = st.EmbeddingsStored
		stats.VectorIndexed = st.VectorIndexed
	}
	return stats, result.Warnings, nil
}

func reviewCmd() *cobra.Command {
	var noAI, interactive bool
	var folder string
	var today, week, templatesFocus, orphansFocus, tasksFocus, structureFocus bool
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
				reviewFolder := folder
				if today || week {
					reviewFolder = strings.Trim(a.Config.Daily.Folder, "/")
				}
				if templatesFocus {
					reviewFolder = "Templates"
				}
				report, props, err := r.Review(ctx, noAI, reviewFolder)
				if err != nil {
					return err
				}
				if orphansFocus {
					orphans, err := r.OrphanNotes(ctx)
					if err != nil {
						return err
					}
					report.Title = "Orphan Notes Review"
					report.Lines = append(report.Lines, orphans...)
				}
				if tasksFocus {
					taskReport, _, err := r.TasksWithOptions(ctx, engines.TaskOptions{Week: week, Today: today})
					if err != nil {
						return err
					}
					report.Issues = append(report.Issues, taskReport.Issues...)
					report.Lines = append(report.Lines, taskReport.Lines...)
				}
				if structureFocus {
					structureReport, structureProp, err := r.Structure(ctx, true)
					if err != nil {
						return err
					}
					report.Issues = append(report.Issues, structureReport.Issues...)
					report.Lines = append(report.Lines, structureReport.Lines...)
					if structureProp != nil {
						props = append(props, structureProp)
					}
				}
				if templatesFocus {
					templateReport, templateProp, err := r.Templates(ctx, "Templates")
					if err != nil {
						return err
					}
					report.Issues = append(report.Issues, templateReport.Issues...)
					report.Lines = append(report.Lines, templateReport.Lines...)
					if templateProp != nil {
						props = append(props, templateProp)
					}
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
	cmd.Flags().BoolVar(&today, "today", false, "focus today's notes")
	cmd.Flags().BoolVar(&week, "week", false, "focus this week")
	cmd.Flags().BoolVar(&templatesFocus, "templates", false, "focus templates")
	cmd.Flags().BoolVar(&orphansFocus, "orphans", false, "focus orphan notes")
	cmd.Flags().BoolVar(&tasksFocus, "tasks", false, "focus tasks")
	cmd.Flags().BoolVar(&structureFocus, "structure", false, "focus structure")
	return cmd
}

func dailyCmd() *cobra.Command {
	var dateText string
	var apply, showContext, interactive, week, createPermanentNotes, moveTasks bool
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
				result, err := r.DailyWithOptions(ctx, engines.DailyOptions{
					Date:                 dateText,
					Week:                 week,
					CreatePermanentNotes: createPermanentNotes,
					MoveTasks:            moveTasks,
					ShowContext:          showContext,
				})
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
	cmd.Flags().BoolVar(&week, "week", false, "distill the current week")
	cmd.Flags().BoolVar(&createPermanentNotes, "create-permanent-notes", false, "propose permanent notes")
	cmd.Flags().BoolVar(&moveTasks, "move-tasks", false, "propose task moves")
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
				result, err := r.ProjectWithOptions(ctx, args[0], engines.ProjectOptions{
					Folder:   folder,
					Generate: splitCSV(generate),
				})
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
	var apply, today, week, includeInferred bool
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
				report, prop, err := r.TasksWithOptions(ctx, engines.TaskOptions{
					Project:         project,
					Today:           today,
					Week:            week,
					IncludeInferred: includeInferred,
				})
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
	cmd.Flags().BoolVar(&today, "today", false, "focus today")
	cmd.Flags().BoolVar(&week, "week", false, "focus week")
	cmd.Flags().BoolVar(&includeInferred, "include-inferred", false, "include inferred tasks")
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
	cmd.Flags().BoolVar(&propose, "propose", false, "prepare move proposals")
	cmd.Flags().BoolVar(&apply, "apply", false, "prepare apply flow")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "open interactive UI")
	return cmd
}

func templatesCmd() *cobra.Command {
	var folder string
	var apply, projectOnly, dailyOnly bool
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Analyze and improve templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				r, _ := runner(ctx, a)
				report, prop, err := r.TemplatesWithOptions(ctx, engines.TemplateOptions{
					Folder:      folder,
					ProjectOnly: projectOnly,
					DailyOnly:   dailyOnly,
				})
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
	cmd.Flags().StringVar(&folder, "folder", "Templates", "template folder")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply generated proposal")
	cmd.Flags().BoolVar(&projectOnly, "project", false, "focus project template")
	cmd.Flags().BoolVar(&dailyOnly, "daily", false, "focus daily template")
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
			if args[0] == "all" {
				return fmt.Errorf("show requires a specific proposal ID. Use `naudia proposals` to list all proposals")
			}
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
				if len(ids) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No pending proposals to apply.")
					return nil
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
						fmt.Fprintf(cmd.ErrOrStderr(), "Failed to apply proposal %d: %v\n", id, err)
						continue
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
				if len(ids) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No pending proposals to reject.")
					return nil
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

func chatCmd() *cobra.Command {
	var showContext, apply bool
	cmd := &cobra.Command{
		Use:   "chat [message]",
		Short: "Chat with Naudia and prepare note edits",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(cmd, func(ctx context.Context, a *app.App) error {
				_, _, _ = scanVault(ctx, a, scanOptions{NoEmbeddings: false})
				r, err := runner(ctx, a)
				if err != nil {
					return err
				}
				pm, err := proposalManager(ctx, a)
				if err != nil {
					return err
				}
				if len(args) > 0 {
					message := strings.Join(args, " ")
					result, proposalID, applied, err := runAssistantTurn(cmd, ctx, r, pm, message, nil, apply)
					if err != nil {
						return err
					}
					if format(cmd) == "json" {
						return writeJSON(cmd, map[string]any{"result": result, "proposal_id": proposalID, "applied": applied})
					}
					writeAssistantTurn(cmd, result, proposalID, applied, showContext)
					return nil
				}
				if format(cmd) == "json" {
					return errors.New("chat --json requires a message argument")
				}
				return runInteractiveChat(cmd, ctx, r, pm, apply, showContext)
			})
		},
	}
	cmd.Flags().BoolVar(&showContext, "show-context", false, "show context after each assistant turn")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply generated low-risk note proposals immediately")
	return cmd
}

func runInteractiveChat(cmd *cobra.Command, ctx context.Context, r engines.Runner, pm proposals.Manager, apply bool, showContext bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Naudia chat. Type /help for commands, /quit to exit.")
	scanner := bufio.NewScanner(os.Stdin)
	var history []ai.Message
	var lastContext contextpack.Pack
	for {
		fmt.Fprint(out, "\n> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		handled, exit, err := handleChatCommand(cmd, ctx, r, pm, line, apply, &lastContext)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%v\n", err)
			continue
		}
		if exit {
			return nil
		}
		if handled {
			continue
		}
		result, proposalID, applied, err := runAssistantTurn(cmd, ctx, r, pm, line, history, apply)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%v\n", err)
			continue
		}
		lastContext = result.Context
		writeAssistantTurn(cmd, result, proposalID, applied, showContext)
		history = append(history,
			ai.Message{Role: "user", Content: line},
			ai.Message{Role: "assistant", Content: result.Answer},
		)
	}
	return scanner.Err()
}

func handleChatCommand(cmd *cobra.Command, ctx context.Context, r engines.Runner, pm proposals.Manager, line string, apply bool, lastContext *contextpack.Pack) (bool, bool, error) {
	out := cmd.OutOrStdout()
	switch {
	case line == "/quit" || line == "/exit" || line == "/q":
		return true, true, nil
	case line == "/help":
		fmt.Fprintln(out, chatHelp())
		return true, false, nil
	case line == "/context":
		if lastContext == nil || len(lastContext.Items) == 0 {
			fmt.Fprintln(out, "No context from a previous turn yet.")
			return true, false, nil
		}
		fmt.Fprintln(out, contextpack.Table(*lastContext))
		return true, false, nil
	case line == "/proposals" || strings.HasPrefix(line, "/proposals "):
		extra := strings.TrimSpace(strings.TrimPrefix(line, "/proposals"))
		answer, err := pendingProposalAnswer(ctx, pm)
		if err != nil {
			return true, false, err
		}
		if extra != "" {
			explanation, ok, explainErr := explainOnlyPendingProposal(ctx, pm)
			if explainErr != nil {
				return true, false, explainErr
			}
			if ok {
				answer += "\n\n" + explanation
			}
		}
		fmt.Fprintln(out, answer)
		return true, false, nil
	case strings.HasPrefix(line, "/set-path "):
		parts := strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, "/set-path ")))
		if len(parts) < 2 {
			return true, false, errors.New("/set-path requires a proposal ID and path")
		}
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return true, false, errors.New("/set-path requires a numeric proposal ID")
		}
		answer, err := updateProposalActionPath(ctx, r, pm, id, strings.Join(parts[1:], " "))
		if err != nil {
			return true, false, err
		}
		fmt.Fprintln(out, answer)
		return true, false, nil
	case strings.HasPrefix(line, "/show "):
		id, err := parseChatID(line, "/show ")
		if err != nil {
			return true, false, err
		}
		p, rec, err := pm.Load(ctx, id)
		if err != nil {
			return true, false, err
		}
		fmt.Fprintln(out, proposals.RenderDetails(*p, rec))
		return true, false, nil
	case strings.HasPrefix(line, "/apply "):
		id, err := parseChatID(line, "/apply ")
		if err != nil {
			return true, false, err
		}
		return true, false, applyProposalFromChat(cmd, ctx, pm, id)
	case strings.HasPrefix(line, "/reject "):
		id, err := parseChatID(line, "/reject ")
		if err != nil {
			return true, false, err
		}
		if err := pm.Store.UpdateProposalStatus(ctx, id, "rejected"); err != nil {
			return true, false, err
		}
		fmt.Fprintf(out, "Rejected proposal %d.\n", id)
		return true, false, nil
	case strings.HasPrefix(line, "/create "):
		return true, false, prepareDirectChatNote(cmd, ctx, r, pm, proposals.ActionCreateNote, strings.TrimSpace(strings.TrimPrefix(line, "/create ")), apply)
	case strings.HasPrefix(line, "/append "):
		return true, false, prepareDirectChatNote(cmd, ctx, r, pm, proposals.ActionAppendToNote, strings.TrimSpace(strings.TrimPrefix(line, "/append ")), apply)
	case strings.HasPrefix(line, "/"):
		return true, false, fmt.Errorf("unknown chat command %q", strings.Fields(line)[0])
	default:
		return false, false, nil
	}
}

func chatHelp() string {
	return strings.TrimSpace(`Commands:
/create path/to/Note.md | Markdown content
/append path/to/Note.md | Markdown content
/show <proposal-id>
/apply <proposal-id>
/reject <proposal-id>
/proposals
/set-path <proposal-id> <path>
/context
/quit`)
}

func runAssistantTurn(cmd *cobra.Command, ctx context.Context, r engines.Runner, pm proposals.Manager, message string, history []ai.Message, apply bool) (engines.AssistResult, int64, bool, error) {
	if result, handled, err := runNaudiaOperation(ctx, r, pm, message); handled {
		return result, 0, false, err
	}
	result, err := r.Assist(ctx, message, history)
	if err != nil {
		return result, 0, false, err
	}
	var proposalID int64
	var applied bool
	if result.Proposal != nil {
		proposalID, err = pm.Save(ctx, result.Proposal)
		if err != nil {
			return result, 0, false, err
		}
		result.Proposal.ID = proposalID
		if apply {
			if _, err := pm.Apply(ctx, proposalID); err != nil {
				return result, proposalID, false, err
			}
			applied = true
		}
	}
	return result, proposalID, applied, nil
}

func writeAssistantTurn(cmd *cobra.Command, result engines.AssistResult, proposalID int64, applied bool, showContext bool) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, result.Answer)
	for _, question := range result.FollowUpQuestions {
		if strings.TrimSpace(question) != "" && question != result.Answer {
			fmt.Fprintf(out, "\nFollow-up: %s\n", question)
		}
	}
	if proposalID > 0 {
		if applied {
			fmt.Fprintf(out, "\nApplied proposal %d.\n", proposalID)
		} else {
			fmt.Fprintf(out, "\nProposal %d prepared. Review with `/show %d`, apply with `/apply %d`, or run `naudia show %d`.\n", proposalID, proposalID, proposalID, proposalID)
		}
	}
	if showContext {
		fmt.Fprintln(out, "\nContext:\n"+contextpack.Table(result.Context))
	}
}

func parseChatID(line, prefix string) (int64, error) {
	value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s requires a proposal ID", strings.TrimSpace(prefix))
	}
	return id, nil
}

func applyProposalFromChat(cmd *cobra.Command, ctx context.Context, pm proposals.Manager, id int64) error {
	p, _, err := pm.Load(ctx, id)
	if err != nil {
		return err
	}
	if p.RequiresConfirmation || p.RiskLevel == proposals.RiskHigh {
		ok, err := ui.ConfirmID(os.Stdin, cmd.OutOrStdout(), id, riskMessage(p, false))
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("proposal %d was not confirmed", id)
		}
	}
	result, err := pm.Apply(ctx, id)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Applied proposal %d (%d actions). Roll back with `naudia rollback %d`.\n", id, len(result.Applied), id)
	return nil
}

func prepareDirectChatNote(cmd *cobra.Command, ctx context.Context, r engines.Runner, pm proposals.Manager, kind proposals.ActionKind, payload string, apply bool) error {
	path, content, ok := strings.Cut(payload, "|")
	if !ok {
		return errors.New("use `|` between the note path and content")
	}
	prop, err := r.NoteChangeProposal(kind, strings.TrimSpace(path), strings.TrimSpace(content))
	if err != nil {
		return err
	}
	id, applied, err := saveMaybeApplyProposal(ctx, pm, prop, apply)
	if err != nil {
		return err
	}
	if applied {
		fmt.Fprintf(cmd.OutOrStdout(), "Applied proposal %d.\n", id)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Proposal %d prepared. Review with `/show %d`, apply with `/apply %d`.\n", id, id, id)
	}
	return nil
}

func noteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "note",
		Short: "Create or append to Obsidian notes",
	}
	cmd.AddCommand(noteCreateCmd(), noteAppendCmd())
	return cmd
}

func noteCreateCmd() *cobra.Command {
	var content, contentFile string
	var apply bool
	cmd := &cobra.Command{
		Use:   "create <path>",
		Short: "Prepare a proposal that creates a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDirectNoteCommand(cmd, args[0], proposals.ActionCreateNote, content, contentFile, apply)
		},
	}
	cmd.Flags().StringVar(&content, "content", "", "Markdown content")
	cmd.Flags().StringVar(&contentFile, "content-file", "", "read Markdown content from a file")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply the proposal immediately")
	return cmd
}

func noteAppendCmd() *cobra.Command {
	var content, contentFile, heading string
	var apply bool
	cmd := &cobra.Command{
		Use:   "append <path>",
		Short: "Prepare a proposal that appends to a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readNoteCommandContent(content, contentFile)
			if err != nil {
				return err
			}
			if heading != "" {
				body = "## " + strings.TrimSpace(heading) + "\n\n" + strings.TrimSpace(body)
			}
			return runDirectNoteCommandWithContent(cmd, args[0], proposals.ActionAppendToNote, body, apply)
		},
	}
	cmd.Flags().StringVar(&content, "content", "", "Markdown content")
	cmd.Flags().StringVar(&contentFile, "content-file", "", "read Markdown content from a file")
	cmd.Flags().StringVar(&heading, "heading", "", "prepend a heading before appended content")
	cmd.Flags().BoolVar(&apply, "apply", false, "apply the proposal immediately")
	return cmd
}

func runDirectNoteCommand(cmd *cobra.Command, path string, kind proposals.ActionKind, content, contentFile string, apply bool) error {
	body, err := readNoteCommandContent(content, contentFile)
	if err != nil {
		return err
	}
	return runDirectNoteCommandWithContent(cmd, path, kind, body, apply)
}

func runDirectNoteCommandWithContent(cmd *cobra.Command, path string, kind proposals.ActionKind, content string, apply bool) error {
	return withApp(cmd, func(ctx context.Context, a *app.App) error {
		r, err := runner(ctx, a)
		if err != nil {
			return err
		}
		prop, err := r.NoteChangeProposal(kind, path, content)
		if err != nil {
			return err
		}
		pm, err := proposalManager(ctx, a)
		if err != nil {
			return err
		}
		id, applied, err := saveMaybeApplyProposal(ctx, pm, prop, apply)
		if err != nil {
			return err
		}
		if format(cmd) == "json" {
			return writeJSON(cmd, map[string]any{"proposal": prop, "proposal_id": id, "applied": applied})
		}
		if applied {
			fmt.Fprintf(cmd.OutOrStdout(), "Applied proposal %d.\n", id)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Proposal %d prepared. Review with `naudia show %d`, apply with `naudia apply %d`.\n", id, id, id)
		}
		return nil
	})
}

func readNoteCommandContent(content, contentFile string) (string, error) {
	if contentFile != "" {
		data, err := os.ReadFile(contentFile)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	if strings.TrimSpace(content) != "" {
		return content, nil
	}
	if !isTerminalStdin() {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(string(data)) != "" {
			return string(data), nil
		}
	}
	return "", errors.New("note content is required; use --content, --content-file, or pipe Markdown on stdin")
}

func saveMaybeApplyProposal(ctx context.Context, pm proposals.Manager, prop *proposals.Proposal, apply bool) (int64, bool, error) {
	id, err := pm.Save(ctx, prop)
	if err != nil {
		return 0, false, err
	}
	prop.ID = id
	if !apply {
		return id, false, nil
	}
	if _, err := pm.Apply(ctx, id); err != nil {
		return id, false, err
	}
	return id, true, nil
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
	hasModel := func(target string) bool {
		for _, m := range models {
			if m == target || m == target+":latest" || m+":latest" == target {
				return true
			}
		}
		return false
	}
	chat := "missing"
	if hasModel(chatModel) {
		chat = "available"
	}
	embedding := "missing"
	if hasModel(embeddingModel) {
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

func emptyString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func stringIn(value string, values []string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
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
