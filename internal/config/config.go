package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/drakeafk/naudia/internal/obsidian"
	"github.com/spf13/viper"
)

type Config struct {
	Ollama   OllamaConfig   `mapstructure:"ollama" json:"ollama"`
	Vault    VaultConfig    `mapstructure:"vault" json:"vault"`
	Obsidian ObsidianConfig `mapstructure:"obsidian" json:"obsidian"`
	Behavior BehaviorConfig `mapstructure:"behavior" json:"behavior"`
	Index    IndexConfig    `mapstructure:"index" json:"index"`
	Context  ContextConfig  `mapstructure:"context" json:"context"`
	Daily    DailyConfig    `mapstructure:"daily" json:"daily"`
	Output   OutputConfig   `mapstructure:"output" json:"output"`
	Ignore   IgnoreConfig   `mapstructure:"ignore" json:"ignore"`
}

type OllamaConfig struct {
	Host           string `mapstructure:"host" json:"host"`
	ChatModel      string `mapstructure:"chat_model" json:"chat_model"`
	EmbeddingModel string `mapstructure:"embedding_model" json:"embedding_model"`
}

type VaultConfig struct {
	Path string `mapstructure:"path" json:"path"`
	Name string `mapstructure:"name" json:"name"`
}

type ObsidianConfig struct {
	UseURI     bool   `mapstructure:"use_uri" json:"use_uri"`
	UseCLI     bool   `mapstructure:"use_cli" json:"use_cli"`
	CLICommand string `mapstructure:"cli_command" json:"cli_command"`
}

type BehaviorConfig struct {
	ApprovalRequired        bool   `mapstructure:"approval_required" json:"approval_required"`
	WriteMode               string `mapstructure:"write_mode" json:"write_mode"`
	MaxFilesPerProposal     int    `mapstructure:"max_files_per_proposal" json:"max_files_per_proposal"`
	SourceCitations         bool   `mapstructure:"source_citations" json:"source_citations"`
	AllowDestructiveChanges bool   `mapstructure:"allow_destructive_changes" json:"allow_destructive_changes"`
}

type IndexConfig struct {
	DatabasePath  string `mapstructure:"database_path" json:"database_path"`
	ChunkSize     int    `mapstructure:"chunk_size" json:"chunk_size"`
	ChunkOverlap  int    `mapstructure:"chunk_overlap" json:"chunk_overlap"`
	UseEmbeddings bool   `mapstructure:"use_embeddings" json:"use_embeddings"`
	VectorBackend string `mapstructure:"vector_backend" json:"vector_backend"`
}

type ContextConfig struct {
	MaxNotes               int     `mapstructure:"max_notes" json:"max_notes"`
	MaxChunks              int     `mapstructure:"max_chunks" json:"max_chunks"`
	MaxCharsTotal          int     `mapstructure:"max_chars_total" json:"max_chars_total"`
	MaxCharsPerNote        int     `mapstructure:"max_chars_per_note" json:"max_chars_per_note"`
	MaxSemanticMatches     int     `mapstructure:"max_semantic_matches" json:"max_semantic_matches"`
	MinSimilarityThreshold float64 `mapstructure:"min_similarity_threshold" json:"min_similarity_threshold"`
	IncludeFullNotes       bool    `mapstructure:"include_full_notes" json:"include_full_notes"`
	PreferHeadings         bool    `mapstructure:"prefer_headings" json:"prefer_headings"`
	RecentDailyNoteDays    int     `mapstructure:"recent_daily_note_days" json:"recent_daily_note_days"`
}

type DailyConfig struct {
	Folder     string `mapstructure:"folder" json:"folder"`
	DateFormat string `mapstructure:"date_format" json:"date_format"`
}

type OutputConfig struct {
	DefaultFormat string `mapstructure:"default_format" json:"default_format"`
	UseColor      bool   `mapstructure:"use_color" json:"use_color"`
	UseUnicode    bool   `mapstructure:"use_unicode" json:"use_unicode"`
	TerminalLinks bool   `mapstructure:"terminal_links" json:"terminal_links"`
}

type IgnoreConfig struct {
	Patterns []string `mapstructure:"patterns" json:"patterns"`
}

type LoadOptions struct {
	VaultPath  string
	ConfigPath string
}

func Default() Config {
	return Config{
		Ollama: OllamaConfig{
			Host:           "http://localhost:11434",
			ChatModel:      "llama3.1:8b",
			EmbeddingModel: "nomic-embed-text",
		},
		Obsidian: ObsidianConfig{
			UseURI:     true,
			UseCLI:     false,
			CLICommand: "obsidian",
		},
		Behavior: BehaviorConfig{
			ApprovalRequired:        true,
			WriteMode:               "proposal",
			MaxFilesPerProposal:     20,
			SourceCitations:         true,
			AllowDestructiveChanges: false,
		},
		Index: IndexConfig{
			DatabasePath:  ".naudia/naudia.sqlite",
			ChunkSize:     1200,
			ChunkOverlap:  150,
			UseEmbeddings: true,
			VectorBackend: "sqlite-vec",
		},
		Context: ContextConfig{
			MaxNotes:               8,
			MaxChunks:              16,
			MaxCharsTotal:          24000,
			MaxCharsPerNote:        6000,
			MaxSemanticMatches:     6,
			MinSimilarityThreshold: 0.68,
			IncludeFullNotes:       false,
			PreferHeadings:         true,
			RecentDailyNoteDays:    14,
		},
		Daily: DailyConfig{
			Folder:     "Daily",
			DateFormat: "2006-01-02",
		},
		Output: OutputConfig{
			DefaultFormat: "pretty",
			UseColor:      true,
			UseUnicode:    true,
			TerminalLinks: true,
		},
		Ignore: IgnoreConfig{
			Patterns: []string{
				".naudia/**",
				".git/**",
				"node_modules/**",
				".obsidian/workspace*",
				".obsidian/cache/**",
				".DS_Store",
			},
		},
	}
}

func Load(opts LoadOptions) (Config, string, error) {
	cfg := Default()
	v := viper.New()
	v.SetConfigType("toml")
	setDefaults(v, cfg)
	v.SetEnvPrefix("NAUDIA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	globalPath := filepath.Join(userConfigDir(), "config.toml")
	if _, err := os.Stat(globalPath); err == nil {
		if err := mergeFile(v, globalPath); err != nil {
			return cfg, "", err
		}
	}

	localPath := opts.ConfigPath
	if localPath == "" {
		searchRoot := opts.VaultPath
		if searchRoot == "" {
			if wd, err := os.Getwd(); err == nil {
				searchRoot = wd
			}
		}
		localPath = FindLocalConfig(searchRoot)
	}
	if localPath != "" {
		if err := mergeFile(v, localPath); err != nil {
			return cfg, "", err
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, "", err
	}
	if opts.VaultPath != "" {
		abs, err := filepath.Abs(opts.VaultPath)
		if err != nil {
			return cfg, "", err
		}
		cfg.Vault.Path = abs
	}
	if cfg.Vault.Path == "" {
		if localPath != "" {
			cfg.Vault.Path = filepath.Dir(filepath.Dir(localPath))
		} else {
			wd, _ := os.Getwd()
			cfg.Vault.Path = wd
		}
	}
	abs, err := filepath.Abs(cfg.Vault.Path)
	if err != nil {
		return cfg, "", err
	}
	cfg.Vault.Path = abs
	if cfg.Vault.Name == "" {
		cfg.Vault.Name = obsidian.DeriveVaultName(cfg.Vault.Path)
	}
	if localPath == "" {
		localPath = filepath.Join(cfg.Vault.Path, ".naudia", "config.toml")
	}
	return cfg, localPath, nil
}

func FindLocalConfig(start string) string {
	if start == "" {
		return ""
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	stat, err := os.Stat(abs)
	if err == nil && !stat.IsDir() {
		abs = filepath.Dir(abs)
	}
	for {
		candidate := filepath.Join(abs, ".naudia", "config.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(RenderTOML(cfg)), 0o644)
}

func Validate(cfg Config) error {
	if cfg.Vault.Path == "" {
		return errors.New("vault path is required")
	}
	stat, err := os.Stat(cfg.Vault.Path)
	if err != nil {
		return fmt.Errorf("vault path is not available: %w", err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("vault path is not a directory: %s", cfg.Vault.Path)
	}
	if cfg.Index.ChunkSize <= 0 {
		return errors.New("index.chunk_size must be positive")
	}
	if cfg.Context.MaxCharsTotal <= 0 {
		return errors.New("context.max_chars_total must be positive")
	}
	return nil
}

func RenderTOML(cfg Config) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "[ollama]\n")
	fmt.Fprintf(&b, "host = %q\nchat_model = %q\nembedding_model = %q\n\n", cfg.Ollama.Host, cfg.Ollama.ChatModel, cfg.Ollama.EmbeddingModel)
	fmt.Fprintf(&b, "[vault]\n")
	fmt.Fprintf(&b, "path = %q\nname = %q\n\n", cfg.Vault.Path, cfg.Vault.Name)
	fmt.Fprintf(&b, "[obsidian]\n")
	fmt.Fprintf(&b, "use_uri = %t\nuse_cli = %t\ncli_command = %q\n\n", cfg.Obsidian.UseURI, cfg.Obsidian.UseCLI, cfg.Obsidian.CLICommand)
	fmt.Fprintf(&b, "[behavior]\n")
	fmt.Fprintf(&b, "approval_required = %t\nwrite_mode = %q\nmax_files_per_proposal = %d\nsource_citations = %t\nallow_destructive_changes = %t\n\n",
		cfg.Behavior.ApprovalRequired, cfg.Behavior.WriteMode, cfg.Behavior.MaxFilesPerProposal, cfg.Behavior.SourceCitations, cfg.Behavior.AllowDestructiveChanges)
	fmt.Fprintf(&b, "[index]\n")
	fmt.Fprintf(&b, "database_path = %q\nchunk_size = %d\nchunk_overlap = %d\nuse_embeddings = %t\nvector_backend = %q\n\n",
		cfg.Index.DatabasePath, cfg.Index.ChunkSize, cfg.Index.ChunkOverlap, cfg.Index.UseEmbeddings, cfg.Index.VectorBackend)
	fmt.Fprintf(&b, "[context]\n")
	fmt.Fprintf(&b, "max_notes = %d\nmax_chunks = %d\nmax_chars_total = %d\nmax_chars_per_note = %d\nmax_semantic_matches = %d\nmin_similarity_threshold = %.2f\ninclude_full_notes = %t\nprefer_headings = %t\nrecent_daily_note_days = %d\n\n",
		cfg.Context.MaxNotes, cfg.Context.MaxChunks, cfg.Context.MaxCharsTotal, cfg.Context.MaxCharsPerNote, cfg.Context.MaxSemanticMatches, cfg.Context.MinSimilarityThreshold, cfg.Context.IncludeFullNotes, cfg.Context.PreferHeadings, cfg.Context.RecentDailyNoteDays)
	fmt.Fprintf(&b, "[daily]\nfolder = %q\ndate_format = %q\n\n", cfg.Daily.Folder, cfg.Daily.DateFormat)
	fmt.Fprintf(&b, "[output]\ndefault_format = %q\nuse_color = %t\nuse_unicode = %t\nterminal_links = %t\n\n",
		cfg.Output.DefaultFormat, cfg.Output.UseColor, cfg.Output.UseUnicode, cfg.Output.TerminalLinks)
	fmt.Fprintf(&b, "[ignore]\npatterns = [\n")
	for _, p := range cfg.Ignore.Patterns {
		fmt.Fprintf(&b, "  %q,\n", p)
	}
	fmt.Fprintf(&b, "]\n")
	return b.String()
}

func mergeFile(v *viper.Viper, path string) error {
	v.SetConfigFile(path)
	return v.MergeInConfig()
}

func setDefaults(v *viper.Viper, cfg Config) {
	v.SetDefault("ollama.host", cfg.Ollama.Host)
	v.SetDefault("ollama.chat_model", cfg.Ollama.ChatModel)
	v.SetDefault("ollama.embedding_model", cfg.Ollama.EmbeddingModel)
	v.SetDefault("obsidian.use_uri", cfg.Obsidian.UseURI)
	v.SetDefault("obsidian.use_cli", cfg.Obsidian.UseCLI)
	v.SetDefault("obsidian.cli_command", cfg.Obsidian.CLICommand)
	v.SetDefault("behavior.approval_required", cfg.Behavior.ApprovalRequired)
	v.SetDefault("behavior.write_mode", cfg.Behavior.WriteMode)
	v.SetDefault("behavior.max_files_per_proposal", cfg.Behavior.MaxFilesPerProposal)
	v.SetDefault("behavior.source_citations", cfg.Behavior.SourceCitations)
	v.SetDefault("behavior.allow_destructive_changes", cfg.Behavior.AllowDestructiveChanges)
	v.SetDefault("index.database_path", cfg.Index.DatabasePath)
	v.SetDefault("index.chunk_size", cfg.Index.ChunkSize)
	v.SetDefault("index.chunk_overlap", cfg.Index.ChunkOverlap)
	v.SetDefault("index.use_embeddings", cfg.Index.UseEmbeddings)
	v.SetDefault("index.vector_backend", cfg.Index.VectorBackend)
	v.SetDefault("context.max_notes", cfg.Context.MaxNotes)
	v.SetDefault("context.max_chunks", cfg.Context.MaxChunks)
	v.SetDefault("context.max_chars_total", cfg.Context.MaxCharsTotal)
	v.SetDefault("context.max_chars_per_note", cfg.Context.MaxCharsPerNote)
	v.SetDefault("context.max_semantic_matches", cfg.Context.MaxSemanticMatches)
	v.SetDefault("context.min_similarity_threshold", cfg.Context.MinSimilarityThreshold)
	v.SetDefault("context.include_full_notes", cfg.Context.IncludeFullNotes)
	v.SetDefault("context.prefer_headings", cfg.Context.PreferHeadings)
	v.SetDefault("context.recent_daily_note_days", cfg.Context.RecentDailyNoteDays)
	v.SetDefault("daily.folder", cfg.Daily.Folder)
	v.SetDefault("daily.date_format", cfg.Daily.DateFormat)
	v.SetDefault("output.default_format", cfg.Output.DefaultFormat)
	v.SetDefault("output.use_color", cfg.Output.UseColor)
	v.SetDefault("output.use_unicode", cfg.Output.UseUnicode)
	v.SetDefault("output.terminal_links", cfg.Output.TerminalLinks)
	v.SetDefault("ignore.patterns", cfg.Ignore.Patterns)
}

func userConfigDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "naudia")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config", "naudia")
}
