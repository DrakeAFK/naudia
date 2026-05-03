package app

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/DrakeAFK/naudia/internal/ai"
	"github.com/DrakeAFK/naudia/internal/config"
	"github.com/DrakeAFK/naudia/internal/db"
	"github.com/DrakeAFK/naudia/internal/util"
)

type Options struct {
	VaultPath  string
	ConfigPath string
	Debug      bool
}

type App struct {
	Config     config.Config
	ConfigPath string
	DB         *db.DB
	AI         *ai.OllamaClient
	Logger     *log.Logger
}

func New(ctx context.Context, opts Options) (*App, error) {
	cfg, cfgPath, err := config.Load(config.LoadOptions{
		VaultPath:  opts.VaultPath,
		ConfigPath: opts.ConfigPath,
	})
	if err != nil {
		return nil, err
	}
	if err := config.Validate(cfg); err != nil {
		return nil, err
	}
	if err := util.EnsureNaudiaDirs(cfg.Vault.Path); err != nil {
		return nil, err
	}
	logger, err := newLogger(cfg.Vault.Path, opts.Debug)
	if err != nil {
		return nil, err
	}
	databasePath := cfg.Index.DatabasePath
	if !filepath.IsAbs(databasePath) {
		databasePath = filepath.Join(cfg.Vault.Path, databasePath)
	}
	store, err := db.Open(ctx, databasePath, logger)
	if err != nil {
		return nil, err
	}
	if _, err := store.UpsertVault(ctx, cfg.Vault.Name, cfg.Vault.Path, cfg.Obsidian.UseCLI, cfg.Obsidian.UseURI); err != nil {
		_ = store.Close()
		return nil, err
	}
	return &App{
		Config:     cfg,
		ConfigPath: cfgPath,
		DB:         store,
		AI:         ai.NewOllamaClient(cfg.Ollama.Host, cfg.Ollama.ChatModel, cfg.Ollama.EmbeddingModel),
		Logger:     logger,
	}, nil
}

func (a *App) Close() error {
	if a == nil || a.DB == nil {
		return nil
	}
	return a.DB.Close()
}

func newLogger(vaultPath string, debug bool) (*log.Logger, error) {
	logPath := filepath.Join(vaultPath, ".naudia", "logs", "naudia.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	flags := log.LstdFlags
	if debug {
		flags |= log.Lshortfile
	}
	return log.New(file, "", flags), nil
}
