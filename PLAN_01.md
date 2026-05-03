# Project Name

**Naudia**

Pronunciation: **nod-ee-uh**

Naudia is a local-first AI operator for Obsidian.

# One-Line Description

Naudia is a local-first AI steward for Obsidian that uses Ollama, SQLite, sqlite-vec, Obsidian URI links, optional Obsidian CLI integration, conservative context budgeting, and safe reviewable diffs to inspect, organize, connect, distill, and update a user’s vault.

# Product Vision

Naudia should feel like a calm, capable local operator for an Obsidian vault.

Not a chatbot.

Not a generic RAG wrapper.

Not another “ask questions about your notes” plugin.

Naudia is a CLI-first AI operator that can inspect a vault, understand its structure, identify useful improvements, generate safe proposals, present those proposals beautifully, and apply approved changes with robust rollback support.

The user should be able to run commands like:

```
naudia review
naudia daily
naudia project "My App"
naudia links
naudia structure
naudia tasks
naudia templates
naudia apply 3
naudia rollback 3
```

And Naudia should inspect the vault, reason over the contents locally, propose improvements, and mutate the vault only after approval.

The long-term goal is for Naudia to become a local AI command center for Obsidian. Since Obsidian is open, Markdown-based, and extensible, Naudia can be as small or as powerful as the user wants.

# Core Product Promise

**Naudia helps your Obsidian vault maintain itself.**

# Preferred Tagline

**Naudia — your local AI steward for Obsidian.**

Other acceptable tagline options:

**Naudia — a local-first AI operator for your Obsidian vault.**

**Naudia — a private AI assistant that helps your Obsidian vault stay clean, connected, and useful.**

# Core Differentiation

Most AI note tools focus on this:

> Chat with your notes.

Naudia should focus on this:

> Do useful work inside your vault.

Naudia should be positioned as an **operator**, not a chatbot.

It should be able to:

* Review vault health
* Clean up daily notes
* Extract tasks and decisions
* Suggest missing links
* Generate project memory
* Improve templates
* Propose better folder structures
* Create reviewable diffs
* Apply approved changes
* Roll back applied changes safely
* Preserve unrelated user edits during rollback whenever possible
* Open relevant notes directly in Obsidian
* Run fully local through Ollama
* Use semantic search without overstuffing local model prompts

# Operator Pattern

The README should explicitly explain this.

Naudia is not a standard RAG chatbot.

A RAG chatbot usually does this:

1. Embed notes
2. Retrieve chunks
3. Answer a question

Naudia does more:

1. Scans the vault deterministically
2. Builds a local structural index
3. Uses local embeddings for similarity when useful
4. Conservatively selects high-signal context
5. Reviews the actual vault state
6. Generates proposed changes
7. Shows diffs
8. Requires approval
9. Applies changes
10. Records rollback information
11. Detects conflicts
12. Opens affected notes in Obsidian when useful

Core words to emphasize:

* Steward
* Operator
* Diffs
* Rollbacks
* Determinism
* Local-first
* Private
* Reviewable changes
* Conservative context
* Conflict-safe editing

Suggested README section:

# Why Naudia is not just another RAG chatbot

Most AI note tools let you ask questions about your notes.

Naudia helps you operate your vault.

Instead of only retrieving context and answering questions, Naudia scans your vault, understands its structure, identifies maintenance issues, proposes concrete edits, shows reviewable diffs, applies approved changes, and records rollback information.

Your notes stay local. Your model runs local. Your changes are deterministic, reviewable, and reversible whenever safe.

Naudia is built around four core ideas:

* Stewardship over chat
* Diffs over magic
* Rollbacks over blind edits
* Deterministic structure before AI judgment

# Design Principles

# 1. Local-first

Notes stay local.

AI runs locally through Ollama.

Vector search runs locally through SQLite and sqlite-vec.

No cloud dependency.

No external API keys required.

No telemetry in the initial version.

# 2. Obsidian remains the source of truth

Naudia does not replace Obsidian.

Naudia operates on the user’s existing vault.

The vault remains normal Markdown.

# 3. Proposal-first editing

Naudia should not silently mutate notes.

All meaningful changes must be represented as proposals.

The user can review, apply, reject, and roll back changes.

# 4. Diffs over magic

The primary trust mechanism is the diff.

Naudia should not say “I improved your vault” in a vague way.

It should show exactly what it wants to change.

# 5. Rollbacks are mandatory

Every applied proposal should create rollback information.

The user should feel safe letting Naudia operate.

Rollback must be robust enough to preserve unrelated manual edits whenever possible.

# 6. Deterministic first, AI second

Naudia should use deterministic parsing and rules wherever possible.

AI should enhance judgment, summarization, classification, and drafting.

AI should not be required for basic vault scanning, indexing, or structural analysis.

# 7. Conservative context by default

Local models can degrade when overloaded with loosely related context.

Naudia should retrieve less context by default and ask the model to reason over better context.

Semantic search should produce candidates, not final truth.

Exact matches, explicit links, tags, and folder proximity should outrank vector similarity by default.

# 8. Premium terminal experience

The terminal output cannot look like a raw script.

Naudia should feel premium.

Think:

* Linear
* Supabase
* Vercel
* Raycast
* Arc
* Modern developer tools

The CLI should have a minimalist, structural aesthetic.

Use the Charmbracelet ecosystem.

# 9. No AI slop

Do not create generic filler notes.

Do not invent nonexistent context.

Preserve the user’s voice unless asked to rewrite.

Source important suggestions from actual vault content.

# 10. Seamless Obsidian bridge

Naudia should not feel disconnected from Obsidian.

When Naudia references a note, it should provide an Obsidian URI link when possible.

Example:

```
obsidian://open?vault=Main&file=Projects/Naudia.md
```

In supported terminals, this should appear as a clickable link.

# 11. Frictionless installation

The project should be easy to install.

Target install methods:

```
brew install drakeafk/naudia/naudia
```

And:

```
curl -fsSL https://raw.githubusercontent.com/drakeafk/naudia/main/scripts/install.sh | sh
```

Also provide GitHub Releases with binaries for:

* macOS arm64
* macOS amd64
* Linux amd64
* Linux arm64
* Windows amd64

# Intended Audience

Naudia should be useful for everyone who uses Obsidian, including:

* Developers
* Writers
* Researchers
* Students
* Knowledge workers
* Builders
* Indie hackers
* People with messy vaults
* People who use daily notes heavily
* People who want private AI workflows

The first implementation should be especially strong for technical users because GitHub stars will likely come from developers first.

# Non-Goals for Initial Version

Do not start with:

* Full Obsidian plugin
* Cloud sync
* Hosted AI
* Mobile support
* Voice assistant
* Browser extension
* Multi-user collaboration
* Real-time background daemon
* Complex GUI
* Complex plugin marketplace integrations

Naudia should begin as a polished CLI/TUI tool.

# Recommended Tech Stack

Use **Go**.

Recommended stack:

* Language: Go
* CLI framework: Cobra
* Config: Viper
* TUI framework: Bubble Tea
* Styling: Lip Gloss
* Components: Bubbles
* Markdown rendering: Glamour
* Terminal tables/lists: Charmbracelet components or custom Lip Gloss layouts
* SQLite driver: modernc.org/sqlite or mattn/go-sqlite3
* Vector search: sqlite-vec
* LLM runtime: Ollama HTTP API
* Embeddings: Ollama embeddings API
* Diffs: sergi/go-diff or similar
* Patch application: robust unified diff / three-way merge helper
* File watching later: fsnotify
* Testing: Go testing package + testify
* Packaging: GoReleaser
* Distribution: Homebrew tap, install script, GitHub Releases

# Why Go

Naudia should use Go because:

* Single static binaries are ideal for CLI distribution.
* It avoids Node/npm friction.
* It feels more serious and native as a developer tool.
* Go is excellent for filesystem traversal, SQLite, HTTP clients, and CLIs.
* Charmbracelet makes Go the best choice for premium terminal interfaces.
* GoReleaser makes multi-platform releases and Homebrew taps straightforward.
* A fast, local, boring-but-effective CLI is exactly the kind of tool developers like to star.

# Repository Structure

Use this structure:

```
naudia/
  README.md
  PLAN.md
  LICENSE
  go.mod
  go.sum
  .goreleaser.yaml

  cmd/
    naudia/
      main.go

  internal/
    app/
      app.go
      version.go

    cli/
      root.go
      init.go
      status.go
      scan.go
      review.go
      daily.go
      project.go
      links.go
      tasks.go
      structure.go
      templates.go
      proposals.go
      show.go
      apply.go
      reject.go
      rollback.go
      ask.go

    tui/
      theme.go
      layout.go
      components.go
      spinner.go
      status.go
      review.go
      proposals.go
      diff.go
      confirm.go
      context.go
      conflict.go

    config/
      config.go
      defaults.go
      validate.go

    obsidian/
      vault.go
      uri.go
      cli.go
      paths.go

    vault/
      scanner.go
      parser.go
      markdown.go
      frontmatter.go
      links.go
      tags.go
      tasks.go
      headings.go
      files.go
      ignore.go

    db/
      db.go
      migrate.go
      migrations/
        001_init.sql
        002_vec.sql
        003_rollback_conflicts.sql
      repo/
        vaults.go
        notes.go
        links.go
        tags.go
        tasks.go
        chunks.go
        embeddings.go
        proposals.go
        changes.go
        conflicts.go

    ai/
      ollama.go
      chat.go
      embeddings.go
      prompts/
        system.go
        review.go
        daily.go
        project.go
        links.go
        tasks.go
        structure.go
        templates.go
      context.go
      chunking.go

    context/
      budget.go
      item.go
      rank.go
      dedupe.go
      pack.go
      render.go

    vector/
      sqlite_vec.go
      similarity.go
      search.go

    engines/
      review.go
      daily.go
      project.go
      links.go
      tasks.go
      structure.go
      templates.go

    proposals/
      proposal.go
      action.go
      diff.go
      apply.go
      rollback.go
      merge.go
      conflict.go
      render.go
      validate.go

    output/
      console.go
      markdown.go
      json.go
      table.go

    util/
      hash.go
      dates.go
      logger.go
      errors.go
      fs.go

  scripts/
    install.sh

  testdata/
    sample-vault/
      Daily/
      Projects/
      Templates/
      Ideas/
      Resources/

  tests/
    integration/
```

# Premium Terminal Interface Requirements

Naudia should not output plain walls of text.

The terminal UI is a core product feature.

Use Charmbracelet:

* Bubble Tea for interactive screens
* Lip Gloss for layout and styling
* Bubbles for lists, spinners, text inputs, progress, and viewport
* Glamour for Markdown rendering

# Visual Style

Aim for:

* Minimal
* Structured
* Calm
* High contrast but not loud
* Modern
* Premium
* Developer-native

Avoid:

* Excessive emojis
* Rainbow colors
* Noisy ASCII art
* Giant banners
* Raw JSON unless requested
* Dense unformatted logs

# Suggested Theme

Use a restrained palette:

* Primary: muted violet, blue, or cyan
* Success: green
* Warning: amber
* Danger: red
* Muted text: gray
* Borders: subtle gray
* Background: terminal default

The product should look good in both dark and light terminals.

# Layout Examples

Status card:

```
╭─ Naudia Status ─────────────────────────────╮
│ Vault          Main                         │
│ Notes          1,284                        │
│ Last scan      2026-05-03 22:14             │
│ Ollama         online                       │
│ Model          llama3.1:8b                  │
│ Embeddings     nomic-embed-text             │
│ Vector search  sqlite-vec enabled           │
│ Proposals      4 pending                    │
╰─────────────────────────────────────────────╯
```

Review summary:

```
╭─ Vault Review ──────────────────────────────╮
│ Naudia reviewed 1,284 notes.                │
│                                             │
│ Health       Good, but cluttered            │
│ Risk         Low                            │
│ Proposals    12 prepared                    │
╰─────────────────────────────────────────────╯

Structure
  • 47 notes are uncategorized
  • 12 project notes appear outside Projects/
  • 8 templates are inconsistent

Daily Notes
  • 21 daily notes contain unresolved tasks
  • 9 daily notes contain reusable project knowledge
```

Proposal list:

```
╭─ Pending Proposals ─────────────────────────╮
│ 1  Create Projects/Naudia/PLAN.md     low   │
│ 2  Add missing links to 8 notes       low   │
│ 3  Move 14 notes into Projects/       high  │
│ 4  Improve Templates/Project.md       med   │
╰─────────────────────────────────────────────╯
```

Rollback conflict:

```
╭─ Rollback Conflict ─────────────────────────╮
│ Naudia could not safely roll back proposal 3│
│ for this file:                              │
│                                             │
│ Projects/App/PLAN.md                        │
│                                             │
│ The file was manually edited after Naudia   │
│ applied the proposal, and the affected      │
│ section could not be matched confidently.   │
│                                             │
│ Conflict details were written to:           │
│ .naudia/conflicts/proposal-3.json           │
╰─────────────────────────────────────────────╯
```

Diff viewer should be clean and scrollable in interactive mode.

# Output Modes

Default:

```
pretty
```

Supported:

```
pretty
json
markdown
quiet
```

Examples:

```
naudia review
naudia review --json
naudia review --markdown
naudia scan --quiet
```

# Interactive and Non-Interactive Modes

Naudia should support both scriptable CLI behavior and interactive TUI behavior.

Default commands can print beautiful static output.

For complex review/apply flows, support interactive mode:

```
naudia proposals --interactive
naudia review --interactive
naudia apply 3 --interactive
```

Non-interactive mode must still work:

```
naudia apply 3 --yes
naudia review --json
```

# Configuration

Naudia should support global and vault-local configuration.

Global config:

```
~/.config/naudia/config.toml
```

Vault-local config:

```
<vault>/.naudia/config.toml
```

Vault-local config should override global config where appropriate.

Suggested config:

```
[ollama]
host = "http://localhost:11434"
chat_model = "llama3.1:8b"
embedding_model = "nomic-embed-text"

[vault]
path = "/Users/example/Documents/Obsidian/Main"
name = "Main"

[obsidian]
use_uri = true
use_cli = false
cli_command = "obsidian"

[behavior]
approval_required = true
write_mode = "proposal"
max_files_per_proposal = 20
source_citations = true
allow_destructive_changes = false

[index]
database_path = ".naudia/naudia.sqlite"
chunk_size = 1200
chunk_overlap = 150
use_embeddings = true
vector_backend = "sqlite-vec"

[context]
max_notes = 8
max_chunks = 16
max_chars_total = 24000
max_chars_per_note = 6000
max_semantic_matches = 6
min_similarity_threshold = 0.68
include_full_notes = false
prefer_headings = true
recent_daily_note_days = 14

[daily]
folder = "Daily"
date_format = "2006-01-02"

[output]
default_format = "pretty"
use_color = true
use_unicode = true
terminal_links = true

[ignore]
patterns = [
  ".naudia/**",
  ".git/**",
  "node_modules/**",
  ".obsidian/workspace*"
]
```

# Local Project Folder

Inside each vault, Naudia should create:

```
.naudia/
  config.toml
  naudia.sqlite
  proposals/
  changes/
  conflicts/
  reports/
  logs/
```

Recommended `.gitignore` entry:

```
.naudia/
```

However, Naudia should not assume every vault is in Git.

# Obsidian Integration

Naudia should integrate with Obsidian in two ways:

1. Obsidian URI scheme
2. Optional Obsidian CLI

The URI scheme should be treated as the more universal bridge.

The Obsidian CLI can be used when available, but it should not be the only integration path.

# Obsidian URI Scheme

Naudia should generate Obsidian URI links for notes wherever useful.

Example:

```
obsidian://open?vault=Main&file=Projects/Naudia.md
```

When Naudia proposes a change to a specific note, output a clickable terminal link if terminal hyperlinks are enabled.

Example display:

```
Projects/Naudia.md  open in Obsidian
```

The underlying link should be:

```
obsidian://open?vault=Main&file=Projects%2FNaudia.md
```

Implement URI helpers:

```
BuildOpenNoteURI(vaultName string, filePath string) string
BuildSearchURI(vaultName string, query string) string
```

Naudia should URL-encode values correctly.

Config should include vault name because Obsidian URIs require the vault name:

```
[vault]
name = "Main"
```

If vault name is missing, derive it from the vault folder name but allow override.

# Terminal Hyperlinks

If supported, output OSC 8 hyperlinks.

Example concept:

```
\033]8;;obsidian://open?vault=Main&file=Projects%2FNaudia.md\033\\Projects/Naudia.md\033]8;;\033\\
```

Implement helper:

```
TerminalLink(label string, url string) string
```

Make this configurable:

```
[output]
terminal_links = true
```

If disabled or unsupported, print the raw Obsidian URI below the path.

# Optional Obsidian CLI Integration

Naudia should detect whether the Obsidian CLI is available.

It should support:

* Opening notes
* Appending to notes if reliable
* Triggering Obsidian-aware actions when available

But initial core file operations should use direct filesystem access for reliability and speed.

Commands should not fail completely just because Obsidian CLI is unavailable, unless the specific command requires it.

# Ollama Integration

Naudia should communicate with Ollama through HTTP.

Default host:

```
http://localhost:11434
```

Implement:

```
OllamaClient
  ListModels()
  Chat()
  Generate()
  Embed()
  HealthCheck()
```

Naudia should verify Ollama during:

```
naudia init
naudia status
```

If Ollama is unavailable, show a premium formatted error:

```
╭─ Ollama Unavailable ────────────────────────╮
│ Naudia could not reach Ollama at:           │
│ http://localhost:11434                      │
│                                             │
│ Start Ollama and try again:                 │
│                                             │
│   ollama serve                              │
│   ollama pull llama3.1:8b                   │
│   ollama pull nomic-embed-text              │
╰─────────────────────────────────────────────╯
```

# Recommended Default Models

Initial defaults:

```
Chat model: llama3.1:8b
Embedding model: nomic-embed-text
```

Also support:

```
qwen2.5:7b
mistral-nemo
gemma3:12b
mxbai-embed-large
```

Do not hardcode assumptions that a model exists.

Always check.

# Vector Search

Use **sqlite-vec**.

sqlite-vec should be the preferred vector backend.

Reason:

* Runs locally
* Fast
* Boring and effective
* Fits the SQLite architecture
* Avoids external vector database complexity
* Allows native vector similarity search in SQL
* Strong technical choice for open-source credibility

Naudia should use sqlite-vec for embeddings when available.

If sqlite-vec is unavailable, Naudia should degrade gracefully:

* Continue with keyword/title/link search
* Show warning that semantic search is unavailable
* Allow user to disable embeddings

Config:

```
[index]
use_embeddings = true
vector_backend = "sqlite-vec"
```

# Data Model

Use SQLite.

# SQLite Extensions

The implementation should support loading sqlite-vec.

Depending on Go SQLite driver choice, the agent should research the cleanest implementation path and document it.

Preferred behavior:

* On startup, verify sqlite-vec availability.
* On `naudia status`, show vector backend status.
* On `naudia scan`, create vector tables if embeddings are enabled.
* If sqlite-vec cannot load, continue without semantic search.

Status example:

```
Vector search  sqlite-vec enabled
```

or:

```
Vector search  unavailable; semantic suggestions disabled
```

# Core Tables

# vaults

```
CREATE TABLE vaults (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  path TEXT NOT NULL UNIQUE,
  obsidian_cli_enabled INTEGER NOT NULL DEFAULT 0,
  obsidian_uri_enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

# notes

```
CREATE TABLE notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  vault_id INTEGER NOT NULL,
  path TEXT NOT NULL,
  title TEXT,
  frontmatter_json TEXT,
  content_hash TEXT NOT NULL,
  word_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT,
  modified_at TEXT,
  indexed_at TEXT NOT NULL,
  FOREIGN KEY (vault_id) REFERENCES vaults(id),
  UNIQUE(vault_id, path)
);
```

# headings

```
CREATE TABLE headings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  note_id INTEGER NOT NULL,
  level INTEGER NOT NULL,
  text TEXT NOT NULL,
  line_number INTEGER,
  FOREIGN KEY (note_id) REFERENCES notes(id)
);
```

# links

```
CREATE TABLE links (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source_note_id INTEGER NOT NULL,
  target_raw TEXT NOT NULL,
  target_note_id INTEGER,
  link_text TEXT,
  line_number INTEGER,
  resolved INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY (source_note_id) REFERENCES notes(id),
  FOREIGN KEY (target_note_id) REFERENCES notes(id)
);
```

# tags

```
CREATE TABLE tags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  note_id INTEGER NOT NULL,
  tag TEXT NOT NULL,
  FOREIGN KEY (note_id) REFERENCES notes(id)
);
```

# tasks

```
CREATE TABLE tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  note_id INTEGER NOT NULL,
  text TEXT NOT NULL,
  completed INTEGER NOT NULL DEFAULT 0,
  line_number INTEGER,
  inferred INTEGER NOT NULL DEFAULT 0,
  project_guess TEXT,
  due_date_guess TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (note_id) REFERENCES notes(id)
);
```

# chunks

```
CREATE TABLE chunks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  note_id INTEGER NOT NULL,
  chunk_index INTEGER NOT NULL,
  content TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  heading_context TEXT,
  token_estimate INTEGER,
  created_at TEXT NOT NULL,
  FOREIGN KEY (note_id) REFERENCES notes(id),
  UNIQUE(note_id, chunk_index)
);
```

# embeddings metadata

Use this table to track embedding metadata even if vector data lives in sqlite-vec virtual tables.

```
CREATE TABLE embeddings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  note_id INTEGER NOT NULL,
  chunk_id INTEGER NOT NULL,
  model TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  dimensions INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (note_id) REFERENCES notes(id),
  FOREIGN KEY (chunk_id) REFERENCES chunks(id),
  UNIQUE(chunk_id, model)
);
```

# vector table

Use sqlite-vec virtual table.

Exact schema may need to be adjusted based on sqlite-vec’s Go integration.

Conceptual target:

```
CREATE VIRTUAL TABLE vec_chunks USING vec0(
  embedding float[768]
);
```

Track mapping between vector row IDs and chunks:

```
CREATE TABLE vector_chunks (
  rowid INTEGER PRIMARY KEY,
  chunk_id INTEGER NOT NULL,
  note_id INTEGER NOT NULL,
  model TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  FOREIGN KEY (chunk_id) REFERENCES chunks(id),
  FOREIGN KEY (note_id) REFERENCES notes(id)
);
```

If sqlite-vec implementation details require a different shape, adapt while preserving the same logical relationship.

# proposals

```
CREATE TABLE proposals (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  vault_id INTEGER NOT NULL,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  summary TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  proposal_json TEXT NOT NULL,
  patch_text TEXT,
  created_at TEXT NOT NULL,
  applied_at TEXT,
  rejected_at TEXT,
  rolled_back_at TEXT,
  FOREIGN KEY (vault_id) REFERENCES vaults(id)
);
```

Allowed proposal statuses:

```
pending
applied
rejected
failed
partially_applied
rolled_back
```

# changes

```
CREATE TABLE changes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  proposal_id INTEGER NOT NULL,
  action_id TEXT NOT NULL,
  note_path TEXT NOT NULL,
  action_kind TEXT NOT NULL,

  before_hash TEXT,
  after_hash TEXT,

  previous_content TEXT,
  applied_content TEXT,

  forward_patch TEXT,
  inverse_patch TEXT,

  affected_ranges_json TEXT,
  anchors_json TEXT,

  applied_at TEXT NOT NULL,

  FOREIGN KEY (proposal_id) REFERENCES proposals(id)
);
```

# rollback_conflicts

```
CREATE TABLE rollback_conflicts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  proposal_id INTEGER NOT NULL,
  change_id INTEGER NOT NULL,
  note_path TEXT NOT NULL,
  reason TEXT NOT NULL,
  conflict_json TEXT NOT NULL,
  created_at TEXT NOT NULL,

  FOREIGN KEY (proposal_id) REFERENCES proposals(id),
  FOREIGN KEY (change_id) REFERENCES changes(id)
);
```

# Context Budgeting and Local Model Discipline

Naudia must be highly conservative with context selection by default.

Local models such as `llama3.1:8b`, `qwen2.5:7b`, and similar models can degrade quickly when overloaded with loosely related context. More context is not automatically better. Naudia should prioritize **high-signal, source-grounded, minimal context** over large prompt stuffing.

# Core Rule

Naudia should retrieve less context by default and ask the model to reason over better context.

The system should prefer:

* Exact note matches
* Direct backlinks and outlinks
* Explicit tags
* Recent project notes
* High-confidence semantic matches
* Small excerpts around relevant headings

The system should avoid:

* Sending entire large notes when only one section matters
* Sending many weak semantic matches
* Sending unrelated daily note fragments
* Sending duplicate or near-duplicate chunks
* Treating vector similarity as proof of relevance
* Overloading the model with loosely related context

# Default Context Limits

```
[context]
max_notes = 8
max_chunks = 16
max_chars_total = 24000
max_chars_per_note = 6000
max_semantic_matches = 6
min_similarity_threshold = 0.68
include_full_notes = false
prefer_headings = true
recent_daily_note_days = 14
```

These values must be configurable.

# Context Selection Strategy

For each AI command, Naudia should build a compact context pack.

A context pack should include:

* Source note path
* Obsidian URI
* Relevant heading
* Short excerpt
* Reason this context was selected
* Retrieval method
* Confidence score

Internal model concept:

```
type ContextItem struct {
    NotePath        string
    ObsidianURI     string
    Heading         string
    Excerpt         string
    RetrievalMethod string
    Reason          string
    Score           float64
    CharCount       int
}
```

Retrieval methods:

```
exact_title
exact_path
tag
backlink
outlink
text_search
semantic
recent_daily
user_supplied
```

# Context Ranking

Naudia should rank context before sending it to the model.

Preferred ranking order:

1. User-specified note or folder
2. Exact title/path match
3. Direct links from known project notes
4. Notes in the same project folder
5. Explicit tags
6. Recent daily notes that mention the topic
7. Text search matches
8. Semantic matches

Semantic matches should be treated as candidates, not authoritative sources.

# Context Deduplication

Before sending context to the model, Naudia should deduplicate:

* Same note selected through multiple methods
* Overlapping chunks
* Near-identical excerpts
* Repeated daily note fragments
* Repeated headings

If two chunks overlap, prefer the one with:

1. Stronger retrieval method
2. Higher score
3. More useful heading context
4. Shorter excerpt

# Command-Specific Context Limits

Different commands should use different context budgets.

# `naudia ask`

Can use slightly broader retrieval because it is read-only.

Defaults:

```
max_notes = 10
max_chunks = 20
max_chars_total = 30000
```

# `naudia project`

Should be conservative but allow enough context for project memory.

Defaults:

```
max_notes = 12
max_chunks = 24
max_chars_total = 36000
```

Priority:

1. Existing project folder
2. Project `README` / `PLAN` / `TODO` / `DECISIONS`
3. Directly linked notes
4. Recent daily mentions
5. Semantic matches

# `naudia daily`

Should usually only use the target daily note plus a small amount of project context.

Defaults:

```
max_notes = 6
max_chunks = 12
max_chars_total = 18000
```

# `naudia links`

Should use deterministic matching first.

Semantic context should be minimal.

Defaults:

```
max_semantic_matches = 5
max_chars_total = 12000
```

# `naudia structure`

Should mostly use metadata, not note bodies.

Defaults:

```
include_full_notes = false
max_chars_total = 12000
```

Use:

* Folder tree
* Note counts
* Title patterns
* Template names
* Tags
* Link graph stats

Avoid sending full vault content.

# Context Preview

Add flags that show which context would be sent to the model:

```
naudia project "Naudia" --show-context
naudia daily --show-context
naudia ask "What did I decide about naming?" --show-context
```

Output should include:

* Note path
* Retrieval method
* Score
* Character count
* Heading
* Obsidian link

This helps users trust the retrieval layer.

# Context Overflow Behavior

If the selected context exceeds the configured budget, Naudia should not blindly stuff the prompt.

It should:

1. Rank items
2. Drop weakest items
3. Compress long excerpts only when safe
4. Prefer headings and relevant sections
5. Show a warning in debug mode

Example:

```
Context budget reached.
Selected 16 of 43 candidate chunks.
Dropped low-confidence semantic matches below 0.71 similarity.
```

# Prompt Instruction

Add this behavior to AI prompts:

```
You are receiving a deliberately small context pack. Do not assume missing information. If the provided context is insufficient, say what is missing. Do not infer facts from weakly related notes.
```

# Context Acceptance Criteria

* Naudia never sends unlimited vault content to the model.
* Naudia has conservative default context limits.
* Context limits are configurable.
* Context selection is explainable with `--show-context`.
* Semantic matches are treated as lower-priority candidates.
* Large daily notes and project folders are excerpted by relevant heading where possible.
* If context is insufficient, Naudia says so rather than hallucinating.

# Proposal Model

Every mutation should be represented as a proposal.

Go model concept:

```
type Proposal struct {
    ID                   int64            `json:"id,omitempty"`
    Type                 ProposalType     `json:"type"`
    Title                string           `json:"title"`
    Summary              string           `json:"summary"`
    SourceNotes          []SourceNote     `json:"source_notes"`
    Actions              []ProposalAction `json:"actions"`
    RiskLevel            RiskLevel        `json:"risk_level"`
    RequiresConfirmation bool             `json:"requires_confirmation"`
    CreatedAt            time.Time        `json:"created_at"`
}
```

Proposal types:

```
daily_distillation
project_compile
link_suggestions
task_extraction
structure_change
template_improvement
vault_review
note_update
note_create
note_move
note_rename
note_delete
```

Proposal action kinds:

```
create_note
update_note
rename_note
move_note
delete_note
append_to_note
replace_section
insert_after_heading
insert_before_heading
update_lines
remove_lines
update_task_status
add_frontmatter
update_frontmatter
```

# Proposal Safety Rules

Naudia must never perform destructive actions silently.

High-risk actions:

* Delete note
* Rename note
* Move note
* Replace entire note
* Remove content
* Modify more than 20 files
* Modify templates
* Modify files outside vault
* Modify hidden folders other than `.naudia`

High-risk proposals should require explicit confirmation even if the user runs:

```
naudia apply all
```

For high-risk proposals, prompt:

```
This proposal may significantly change your vault.

Type the proposal id to confirm:
```

# Rollback System

Use the term **rollback** rather than undo in product messaging.

The command can be:

```
naudia rollback <proposal-id>
```

Also support alias:

```
naudia undo <proposal-id>
```

Naudia must not rely on naive full-file restoration for rollback except in carefully controlled cases.

A user may apply a proposal, then manually edit a different section of the same file, then later ask Naudia to roll back the proposal. In that case, restoring `previous_content` would incorrectly wipe the user’s manual edits.

# Core Rule

Rollback should reverse only the changes Naudia made, while preserving unrelated user edits whenever possible.

Full-file restoration should be a last resort, not the default.

# Change Record Requirements

For every applied action, Naudia should store:

* Proposal ID
* Action ID
* File path
* Action kind
* Before hash
* After hash
* Full previous content
* Full applied content
* Forward patch
* Inverse patch
* Affected ranges
* Stable anchors
* Applied timestamp

Go model concept:

```
type Change struct {
    ID              int64
    ProposalID      int64
    ActionID        string
    NotePath        string
    ActionKind      string
    BeforeHash      string
    AfterHash       string
    PreviousContent string
    AppliedContent  string
    ForwardPatch    string
    InversePatch    string
    AffectedRanges  []AffectedRange
    Anchors         []PatchAnchor
    AppliedAt       time.Time
}

type AffectedRange struct {
    StartLineBefore int
    EndLineBefore   int
    StartLineAfter  int
    EndLineAfter    int
    BeforeTextHash  string
    AfterTextHash   string
}

type PatchAnchor struct {
    BeforeContext string
    AfterContext  string
    Heading       string
    SectionID     string
}
```

# Prefer Structured Actions Over Raw File Rewrites

Whenever possible, proposals should use structured actions:

* `append_to_note`
* `replace_section`
* `insert_after_heading`
* `insert_before_heading`
* `update_lines`
* `remove_lines`
* `update_task_status`
* `add_frontmatter`
* `update_frontmatter`
* `create_note`
* `rename_note`
* `move_note`

Avoid entire-file `update_note` unless necessary.

For example, task extraction should not rewrite a whole daily note. It should use `append_to_note`, `replace_section`, `insert_after_heading`, `update_lines`, or `update_task_status`.

# Patch Application Strategy

When applying a proposal, Naudia should:

1. Read current file content.
2. Verify expected hash if the action requires it.
3. Prefer structured action application.
4. Generate forward patch.
5. Store inverse patch.
6. Store previous and applied content as fallback.
7. Store affected ranges and anchors.
8. Write file.
9. Re-read and verify after hash.
10. Record change.

# Rollback Strategy

When rolling back a proposal, Naudia should not immediately restore `previous_content`.

It should use this order:

1. Try inverse patch against current file.
2. If inverse patch applies cleanly, apply it.
3. If inverse patch fails, try anchor-based rollback.
4. If anchor-based rollback succeeds, apply it and warn that the file had drifted.
5. If rollback is ambiguous, stop and create a conflict artifact.
6. Only use full previous content if the current file still exactly matches the applied hash or the user passes `--force`.

# Clean Rollback

If current file hash equals the stored `after_hash`, rollback is straightforward:

```
current_hash == after_hash
```

Then Naudia may safely restore previous content or apply inverse patch.

# Drifted Rollback

If current file hash does not equal `after_hash`, the file has changed since Naudia applied the proposal:

```
current_hash != after_hash
```

Naudia must not blindly restore previous content.

Instead, it should attempt a three-way rollback.

# Three-Way Rollback

For modified files, Naudia should perform a three-way merge using:

* Base: previous content before Naudia applied the proposal
* Applied: content after Naudia applied the proposal
* Current: current file content

Goal:

Reverse the difference between Base and Applied while preserving unrelated changes in Current.

Conceptually:

```
rollback_patch = diff(Applied, Base)
result = apply rollback_patch to Current
```

If clean, write result.

If conflict, stop and surface conflict.

# Conflict Handling

If rollback cannot safely apply, Naudia should not overwrite the file.

Instead, it should create a conflict artifact:

```
.naudia/conflicts/
  proposal-3-Projects-App-PLAN.md.conflict
```

Or:

```
.naudia/conflicts/
  proposal-3.json
```

Conflict output should show:

* File path
* What Naudia originally changed
* What changed since
* Why rollback could not be applied safely
* Suggested manual resolution

Terminal output example:

```
╭─ Rollback Conflict ─────────────────────────╮
│ Naudia could not safely roll back proposal 3│
│ for this file:                              │
│                                             │
│ Projects/App/PLAN.md                        │
│                                             │
│ The file was manually edited after Naudia   │
│ applied the proposal, and the affected      │
│ section could not be matched confidently.   │
│                                             │
│ Conflict details were written to:           │
│ .naudia/conflicts/proposal-3.json           │
╰─────────────────────────────────────────────╯
```

# Force Rollback

Allow:

```
naudia rollback 3 --force
```

Force rollback may restore previous content and overwrite user edits.

Before doing so, require confirmation:

```
Force rollback may overwrite manual edits made after this proposal was applied.

Type the proposal id to confirm:
```

# Safer Section-Based Operations

For actions like `replace_section`, Naudia should store:

* Heading text
* Heading level
* Original section content
* New section content
* Section hash before
* Section hash after
* Surrounding context
* Line range at apply time

Rollback should search for the section by heading and hash.

If the same heading appears multiple times, require disambiguation or treat as conflict.

# Safer Task Operations

For task extraction or task movement, avoid replacing whole sections when possible.

Prefer line-level operations:

* Mark specific task line as moved
* Append task to TODO file
* Add source reference
* Do not delete original task unless explicitly approved

If removing or moving tasks, store exact task line hashes.

# Rename and Move Rollbacks

For note moves and renames, store:

* Source path
* Destination path
* Source existence before
* Destination existence before
* File hashes
* Any overwritten destination state

Rollback rules:

1. If destination exists and hash matches Naudia-created or Naudia-moved file, move it back.
2. If destination was modified after move, stop and report conflict.
3. If source path now exists, stop and report conflict.
4. If force is used, require explicit confirmation.

# Delete Rollbacks

Deletes are high-risk.

For delete actions:

* Store full deleted content.
* Store metadata.
* Require explicit confirmation before delete.
* Rollback recreates the deleted note only if the path is still empty.
* If a new file exists at that path, stop and report conflict.

# Apply/Rollback Acceptance Criteria

* Naudia does not wipe unrelated manual edits during rollback.
* Full previous content restoration is only used when current hash equals applied hash or `--force` is passed.
* Drifted files use inverse patch or three-way merge.
* Conflicts are detected and surfaced clearly.
* Conflict artifacts are written to `.naudia/conflicts/`.
* Structured actions are preferred over full-file rewrites.
* Rollback behavior is tested with manual edits in unaffected sections.
* Rename, move, and delete rollback cases are tested.
* Force rollback requires explicit confirmation.

# Command Specification

# `naudia init`

Initializes Naudia for a vault.

Behavior:

* Ask for vault path if not supplied.
* Verify path exists.
* Verify path contains Markdown files.
* Ask for vault name or derive from folder.
* Create `.naudia/`.
* Create `.naudia/config.toml`.
* Create SQLite database.
* Run migrations.
* Check sqlite-vec availability.
* Check Ollama availability.
* List available Ollama models.
* Check Obsidian URI setting.
* Check optional Obsidian CLI availability.
* Save config.

Options:

```
naudia init
naudia init --vault /path/to/vault
naudia init --vault-name Main
naudia init --model llama3.1:8b
naudia init --embedding-model nomic-embed-text
```

Acceptance criteria:

* Creates local config.
* Creates database.
* Gives clear success message.
* Gives helpful warnings if Ollama, sqlite-vec, or Obsidian CLI is missing.
* Does not require Obsidian CLI to function.

# `naudia status`

Shows environment status.

Output should include:

* Vault path
* Vault name
* Number of Markdown notes
* Last scan time
* Database path
* Ollama status
* Chat model
* Embedding model
* sqlite-vec status
* Obsidian URI status
* Obsidian CLI status
* Pending proposals
* Applied proposals
* Any warnings

Example:

```
╭─ Naudia Status ─────────────────────────────╮
│ Vault          Main                         │
│ Path           ~/Documents/Obsidian/Main    │
│ Notes          1,284                        │
│ Last scan      2026-05-03 22:14             │
│                                             │
│ Ollama         online                       │
│ Chat model     llama3.1:8b                  │
│ Embeddings     nomic-embed-text             │
│ Vector search  sqlite-vec enabled           │
│                                             │
│ Obsidian URI   enabled                      │
│ Obsidian CLI   unavailable                  │
│ Proposals      4 pending                    │
╰─────────────────────────────────────────────╯
```

# `naudia scan`

Scans and indexes the vault.

Behavior:

* Find Markdown files.
* Ignore `.naudia/`.
* Respect ignore rules.
* Parse frontmatter.
* Parse headings.
* Parse wiki links.
* Parse Markdown links.
* Parse tags.
* Parse tasks.
* Compute content hashes.
* Update SQLite index.
* Generate chunks.
* Generate embeddings if enabled.
* Store vectors in sqlite-vec if available.

Options:

```
naudia scan
naudia scan --no-embeddings
naudia scan --force
naudia scan --folder Projects
naudia scan --json
naudia scan --quiet
```

Acceptance criteria:

* Correctly indexes notes.
* Does not index `.naudia/`.
* Incremental scan skips unchanged files.
* Force scan rebuilds.
* Embeddings are incremental.
* sqlite-vec failure does not break normal indexing.

# `naudia review`

Reviews vault health.

Behavior:

* Run scan if index is stale.
* Analyze structure, note quality, backlinks, tasks, templates, and daily notes.
* Use deterministic rules first.
* Use AI for higher-level judgment.
* Enforce conservative context budgets before calling Ollama.
* Generate a beautiful terminal report.
* Generate proposals where useful.

Options:

```
naudia review
naudia review --today
naudia review --week
naudia review --folder Projects
naudia review --templates
naudia review --orphans
naudia review --tasks
naudia review --structure
naudia review --no-ai
naudia review --json
naudia review --interactive
```

Review categories:

```
Structure
Daily Notes
Projects
Links
Tasks
Templates
Stale Notes
Duplicate Notes
Orphan Notes
Unresolved Questions
```

Acceptance criteria:

* Produces premium formatted terminal output.
* Creates proposal records.
* Does not apply changes automatically.
* Includes Obsidian links where useful.
* Does not overstuff the local model context.

# `naudia daily`

Distills daily notes.

Behavior:

* Find daily note for date.
* Summarize important content.
* Extract tasks.
* Extract decisions.
* Extract project updates.
* Extract ideas worth keeping.
* Suggest permanent notes.
* Suggest carry-forward items.
* Create proposal to update daily note and/or create review note.

Options:

```
naudia daily
naudia daily --date 2026-05-03
naudia daily --week
naudia daily --apply
naudia daily --create-permanent-notes
naudia daily --move-tasks
naudia daily --show-context
```

Default output format:

```
# Daily Review — YYYY-MM-DD

## Summary

## Decisions

## Tasks

## Project Updates

## Ideas Worth Keeping

## Notes to Create

## Carry Forward
```

Acceptance criteria:

* Does not invent content.
* Cites source daily note path.
* Shows Obsidian link to source note.
* Creates safe proposal.
* Can apply after approval.
* Uses a conservative context pack.
* Can show context with `--show-context`.

# `naudia project "<name>"`

Compiles project memory.

Behavior:

* Search notes by title, path, tags, links, text, and semantic similarity.
* Treat semantic similarity as a candidate source, not truth.
* Find daily note mentions.
* Find existing project folder if any.
* Gather related context conservatively.
* Generate or update durable project files.

Possible files:

```
README.md
PLAN.md
TODO.md
DECISIONS.md
QUESTIONS.md
CHANGELOG.md
CONTEXT.md
```

Options:

```
naudia project "Naudia"
naudia project "Naudia" --generate readme,plan,todo,decisions
naudia project "Naudia" --folder Projects/Naudia
naudia project "Naudia" --apply
naudia project "Naudia" --json
naudia project "Naudia" --show-context
```

Generated structure:

```
Projects/<Project Name>/
  README.md
  PLAN.md
  TODO.md
  DECISIONS.md
  QUESTIONS.md
  CHANGELOG.md
  CONTEXT.md
```

Acceptance criteria:

* Finds relevant notes.
* Shows source notes.
* Shows Obsidian links.
* Generates proposal.
* Does not overwrite existing important files without diff.
* Preserves user content where possible.
* Uses conservative context limits.
* Does not stuff all related notes into the prompt.
* Can show context with `--show-context`.

# `naudia links`

Suggests backlinks and graph improvements.

Behavior:

* Detect mentions of existing note titles that are not linked.
* Detect aliases that should be added.
* Detect notes that should have related links.
* Detect orphan notes.
* Detect possible MOC/index notes.
* Use sqlite-vec semantic search when embeddings are available.
* Prefer deterministic matches over semantic guesses.

Options:

```
naudia links
naudia links --note "Ollama"
naudia links --folder Projects
naudia links --orphans
naudia links --apply
```

Acceptance criteria:

* Suggests specific links.
* Avoids excessive link spam.
* Proposals are small and reviewable.
* Each link suggestion includes a reason.
* Each affected note includes an Obsidian open link.
* Semantic suggestions are conservative and confidence-scored.

# `naudia tasks`

Extracts tasks.

Behavior:

* Find Markdown tasks.
* Optionally infer implied tasks from prose.
* Group by project when possible.
* Identify stale tasks.
* Identify completed tasks.
* Identify unassigned tasks.
* Propose task movement or TODO files.
* Prefer line-level task operations over section or full-file rewrites.

Options:

```
naudia tasks
naudia tasks --today
naudia tasks --week
naudia tasks --project "Naudia"
naudia tasks --include-inferred
naudia tasks --apply
```

Acceptance criteria:

* Clearly separates explicit tasks from inferred tasks.
* Does not rewrite task meaning.
* Can generate project TODO proposal.
* Uses safe structured actions for task updates.

# `naudia decisions`

Optional but useful command.

Behavior:

* Extract decisions from project notes and daily notes.
* Group by project or topic.
* Create/update `DECISIONS.md`.

Options:

```
naudia decisions
naudia decisions --project "Naudia"
naudia decisions --apply
```

# `naudia questions`

Optional but useful command.

Behavior:

* Extract unresolved questions.
* Group by project/topic.
* Create/update `QUESTIONS.md`.

Options:

```
naudia questions
naudia questions --project "Naudia"
naudia questions --apply
```

# `naudia structure`

Analyzes vault structure.

Behavior:

* Analyze folders.
* Identify mixed note types.
* Identify messy root notes.
* Identify inconsistent naming.
* Identify lifecycle issues.
* Propose better structure.
* Use metadata-heavy context, not full note bodies.

Options:

```
naudia structure
naudia structure --propose
naudia structure --apply
naudia structure --interactive
```

Possible recommended structure:

```
00 Inbox/
01 Daily/
02 Projects/
03 Areas/
04 Resources/
05 People/
06 Templates/
99 Archive/
```

Important:

* Do not force this structure.
* Treat it as a recommendation.
* Let users configure their preferred structure.

Acceptance criteria:

* Produces useful structure report.
* Generates migration proposals.
* High-risk moves require explicit confirmation.
* Shows Obsidian links for affected notes.
* Avoids sending large note bodies to the model.

# `naudia templates`

Analyzes and improves templates.

Behavior:

* Find template folder.
* Inspect templates.
* Identify inconsistencies.
* Suggest better templates.
* Generate template proposals.

Options:

```
naudia templates
naudia templates --folder Templates
naudia templates --project
naudia templates --daily
naudia templates --apply
```

Initial template types:

```
Daily Note
Project
Meeting
Person
Resource
Decision
Permanent Note
```

Acceptance criteria:

* Does not overwrite templates without proposal.
* Preserves user-specific style where possible.
* Template changes are reviewable.

# `naudia proposals`

Lists pending proposals.

Options:

```
naudia proposals
naudia proposals --all
naudia proposals --pending
naudia proposals --applied
naudia proposals --interactive
naudia proposals --json
```

Output should be premium.

Example:

```
╭─ Pending Proposals ─────────────────────────╮
│ 1  Create Projects/Naudia/PLAN.md     low   │
│ 2  Add missing links to 8 notes       low   │
│ 3  Move 14 notes into Projects/       high  │
│ 4  Improve Templates/Project.md       med   │
╰─────────────────────────────────────────────╯
```

# `naudia show <proposal-id>`

Shows proposal details.

Behavior:

* Show title.
* Show summary.
* Show source notes.
* Show source Obsidian links.
* Show actions.
* Show risk level.
* Show diff.

Options:

```
naudia show 3
naudia show 3 --json
naudia show 3 --patch
```

# `naudia apply <proposal-id>`

Applies a proposal.

Options:

```
naudia apply 3
naudia apply all
naudia apply 3 --yes
```

Behavior:

* Validate proposal.
* Validate target files.
* Check hashes.
* Show warning for risky changes.
* Apply structured actions.
* Generate and store forward patch.
* Generate and store inverse patch.
* Store affected ranges and anchors.
* Record changes.
* Mark proposal as applied.
* Print rollback command.

Acceptance criteria:

* Does not apply stale proposal if file hash changed.
* Records robust rollback information.
* Handles partial failure safely.
* Shows affected notes with Obsidian links.
* Does not rely only on full-file previous content for rollback.

# `naudia reject <proposal-id>`

Rejects proposal.

Options:

```
naudia reject 3
naudia reject all
```

Behavior:

* Mark proposal rejected.
* Do not delete proposal file by default.

# `naudia rollback <proposal-id>`

Roll back an applied proposal.

Alias:

```
naudia undo <proposal-id>
```

Options:

```
naudia rollback 3
naudia rollback 3 --force
```

Behavior:

* Restore previous file states when safe.
* Preserve unrelated manual edits whenever possible.
* Detect drifted files.
* Attempt inverse patch rollback.
* Attempt anchor-based rollback.
* Attempt three-way rollback.
* Stop and create conflict artifacts when unsafe.
* Mark proposal as rolled back or create rollback conflict record.

Acceptance criteria:

* Does not wipe unrelated user edits.
* Blocks unsafe rollback when current file drift conflicts with Naudia’s prior change.
* Creates conflict artifact when needed.
* Requires explicit confirmation for `--force`.

# `naudia ask "<question>"`

Lower priority than operator workflows, but still useful.

Behavior:

* Answer questions using vault context.
* Cite source notes.
* Include Obsidian links.
* Do not mutate files.
* Use conservative context retrieval.

Options:

```
naudia ask "What did I decide about the app name?"
naudia ask "What are my open tasks for Portico?"
naudia ask "What projects mention Ollama?"
naudia ask "What did I decide about naming?" --show-context
```

Acceptance criteria:

* Retrieval is grounded in vault notes.
* Answers cite note paths.
* If uncertain, say so.
* Can show context with `--show-context`.

# Ignore Rules

Naudia should ignore:

```
.naudia/
.obsidian/workspace*
.obsidian/cache
.git/
node_modules/
.DS_Store
```

Configurable ignore patterns:

```
[ignore]
patterns = [
  ".naudia/**",
  ".git/**",
  "node_modules/**"
]
```

Naudia may read `.obsidian/` config later if useful, but should avoid modifying it initially.

# Markdown Parsing Requirements

Parser should extract:

* Frontmatter
* Title
* Headings
* Wiki links: `[[Note]]`, `[[Note|Alias]]`, `[[Note#Heading]]`
* Markdown links
* Tags: `#tag`
* Tasks: `- [ ]`, `- [x]`

Code blocks should be preserved and should not be treated as note links/tasks unless intentionally supported later.

# Title Resolution

Title should be determined by:

1. Frontmatter `title`
2. First H1 heading
3. File basename

# Link Resolution

Naudia should resolve wiki links by:

1. Exact relative path
2. Exact filename without `.md`
3. Case-insensitive filename match
4. Alias match from frontmatter aliases
5. Heading match when link includes `#`

If ambiguous, mark unresolved or ambiguous.

Do not guess silently.

# AI Prompting

Create prompt files by engine.

# Global System Prompt

Use this as the base behavior for all AI calls:

```
You are Naudia, a local AI steward for an Obsidian vault.

You help maintain, organize, connect, distill, and improve the user's notes.

Rules:
1. Never invent notes, facts, or decisions that are not supported by provided vault context.
2. Prefer small, useful edits over large rewrites.
3. Preserve the user's voice unless explicitly asked to rewrite.
4. Do not remove content unless the proposal clearly explains why.
5. Always cite source note paths for summaries and suggestions.
6. Generate reviewable changes.
7. Avoid generic productivity advice.
8. Treat daily notes as raw capture, not final knowledge.
9. Treat project notes as durable memory.
10. If context is insufficient, say what is missing.
11. Never claim you changed the vault unless a proposal has actually been applied.
12. Do not create AI filler content.
13. Prefer deterministic edits and small diffs.
14. Every recommendation should be grounded in provided vault context.
15. You are receiving a deliberately small context pack. Do not assume missing information.
16. If the provided context is insufficient, say what is missing.
17. Do not infer facts from weakly related notes.
```

# Review Prompt Output

The review engine should ask the model to produce structured JSON.

Expected shape:

```
{
  "summary": "string",
  "issues": [
    {
      "category": "string",
      "severity": "low | medium | high",
      "description": "string",
      "source_notes": ["string"],
      "suggested_action": "string"
    }
  ],
  "proposals": [
    {
      "title": "string",
      "type": "string",
      "summary": "string",
      "source_notes": ["string"],
      "risk_level": "low | medium | high"
    }
  ]
}
```

# Daily Prompt Output

Expected shape:

```
{
  "date": "string",
  "source_note": "string",
  "summary": "string",
  "decisions": ["string"],
  "tasks": [
    {
      "text": "string",
      "explicit": true,
      "project_guess": "string"
    }
  ],
  "project_updates": [
    {
      "project": "string",
      "update": "string"
    }
  ],
  "ideas_worth_keeping": ["string"],
  "notes_to_create": [
    {
      "title": "string",
      "reason": "string",
      "draft_content": "string"
    }
  ],
  "carry_forward": ["string"]
}
```

# Project Prompt Output

Expected shape:

```
{
  "project_name": "string",
  "source_notes": ["string"],
  "current_understanding": "string",
  "goals": ["string"],
  "decisions": ["string"],
  "open_questions": ["string"],
  "tasks": ["string"],
  "risks": ["string"],
  "suggested_files": [
    {
      "path": "string",
      "purpose": "string",
      "content": "string"
    }
  ]
}
```

# Link Prompt Output

Expected shape:

```
{
  "suggestions": [
    {
      "source_note": "string",
      "target_note": "string",
      "reason": "string",
      "confidence": "low | medium | high",
      "suggested_edit": "string"
    }
  ]
}
```

# Structure Prompt Output

Expected shape:

```
{
  "summary": "string",
  "current_issues": ["string"],
  "recommended_structure": [
    {
      "path": "string",
      "purpose": "string"
    }
  ],
  "migration_suggestions": [
    {
      "from": "string",
      "to": "string",
      "reason": "string",
      "risk_level": "low | medium | high"
    }
  ]
}
```

# AI Output Validation

All AI JSON output must be validated before use.

If JSON parsing fails:

* Retry once with a repair prompt.
* If still invalid, show useful error.
* Do not create proposal from invalid output.

# Retrieval and Context Selection

For commands that need context:

1. Use exact path/title/tag matches first.
2. Use link graph.
3. Use text search.
4. Use sqlite-vec semantic search if available.
5. Rank and deduplicate candidates.
6. Enforce context budgets.
7. Include source paths and Obsidian URIs in prompt context.

Do not send the entire vault to the model.

Semantic search should produce candidates, not final truth. Exact matches, explicit links, tags, and folder proximity should outrank vector similarity by default.

Naudia must build an explainable context pack before every AI call. The context pack should be inspectable with `--show-context` for commands that use retrieval.

# Embeddings

Embeddings are optional but recommended.

Initial embedding behavior:

* Chunk notes by heading where possible.
* Store chunks in SQLite.
* Generate embeddings for changed chunks.
* Store vectors using sqlite-vec.
* Search with native vector similarity queries.
* Treat vector search results as candidates only.

If embeddings are disabled or sqlite-vec is unavailable, Naudia should still work with keyword/link/title search.

# Output Modes

Default pretty output.

Support JSON:

```
naudia review --json
naudia proposals --json
naudia status --json
```

Support Markdown reports:

```
naudia review --markdown
naudia daily --markdown
```

This helps scripting and future UI integrations.

# Error Handling

Errors should be helpful, specific, and premium formatted.

Example:

```
╭─ Config Missing ────────────────────────────╮
│ No Naudia config was found for this vault.  │
│                                             │
│ Run:                                        │
│   naudia init                               │
╰─────────────────────────────────────────────╯
```

Example:

```
╭─ Vault Not Found ───────────────────────────╮
│ No Obsidian vault was found at:             │
│ /path/to/vault                              │
│                                             │
│ Initialize with:                            │
│   naudia init --vault /path/to/vault        │
╰─────────────────────────────────────────────╯
```

Example:

```
╭─ Stale Proposal ────────────────────────────╮
│ Proposal 3 is stale because this file       │
│ changed after the proposal was created:     │
│                                             │
│ Projects/App/PLAN.md                        │
│                                             │
│ Run `naudia review` to generate a fresh     │
│ proposal.                                   │
╰─────────────────────────────────────────────╯
```

Example:

```
╭─ Context Budget Reached ────────────────────╮
│ Selected 16 of 43 candidate chunks.         │
│ Dropped low-confidence semantic matches     │
│ below 0.71 similarity.                      │
│                                             │
│ Run with --show-context to inspect what     │
│ Naudia sent to the model.                   │
╰─────────────────────────────────────────────╯
```

# Logging

Write logs to:

```
.naudia/logs/naudia.log
```

Do not log full note contents by default.

Debug mode can log more details, but avoid logging sensitive content unless the user opts in.

Command:

```
naudia --debug review
```

# Testing Requirements

Use Go’s testing package.

Use testify if helpful.

# Unit Tests

Test:

* Config loading
* Vault path detection
* Obsidian URI generation
* Terminal hyperlink generation
* Markdown parsing
* Frontmatter parsing
* Wiki link parsing
* Tag parsing
* Task parsing
* Hashing
* Diff generation
* Proposal validation
* Apply logic
* Rollback logic
* sqlite-vec availability fallback
* Context budget enforcement
* Context ranking and deduplication
* `--show-context` output
* Drifted rollback preserving unrelated manual edits
* Rollback conflict detection
* Force rollback confirmation

# Integration Tests

Use:

```
testdata/sample-vault
```

Test:

* `naudia init`
* `naudia scan`
* `naudia review --no-ai`
* proposal creation
* proposal apply
* proposal rollback
* Obsidian URI generation
* context selection for project and daily commands
* rollback after unrelated manual edit
* rollback conflict after same-section manual edit
* force rollback path

# Rollback-Specific Tests

# Clean Rollback Test

Scenario:

1. Apply proposal to append content.
2. Do not modify file.
3. Roll back proposal.

Expected:

* File exactly matches original content.
* Proposal marked rolled back.

# Drifted Unrelated Edit Test

Scenario:

1. Apply proposal replacing a section.
2. User manually edits a different section.
3. Roll back proposal.

Expected:

* Naudia reverses only its own section change.
* User’s manual edit remains.

# Drifted Same Section Conflict Test

Scenario:

1. Apply proposal replacing a section.
2. User manually edits that same section.
3. Roll back proposal.

Expected:

* Naudia detects conflict.
* File is not overwritten.
* Conflict artifact is created.

# Full Restore Safety Test

Scenario:

1. Apply proposal.
2. User manually edits file.
3. Naudia attempts rollback.

Expected:

* Naudia does not restore previous full content unless `--force` is passed.

# Force Rollback Test

Scenario:

1. Apply proposal.
2. User manually edits file.
3. User runs `naudia rollback <id> --force`.

Expected:

* Naudia prompts for explicit confirmation.
* After confirmation, previous content may be restored.
* Output warns that manual edits may have been overwritten.

# Move Rollback Conflict Test

Scenario:

1. Naudia moves a note.
2. User edits moved note.
3. User requests rollback.

Expected:

* Naudia detects changed destination.
* Rollback is blocked unless forced.

# Delete Rollback Conflict Test

Scenario:

1. Naudia deletes a note.
2. User creates a new note at the same path.
3. User requests rollback.

Expected:

* Naudia does not overwrite new note.
* Conflict is reported.

# Context-Specific Tests

# Context Budget Test

Scenario:

1. Project has many related notes.
2. User runs `naudia project "Test App"`.

Expected:

* Context pack does not exceed configured limits.
* Weak semantic matches are dropped.
* Exact matches and linked notes are preferred.

# Show Context Test

Scenario:

1. User runs `naudia project "Test App" --show-context`.

Expected:

* Output shows selected note paths.
* Output shows retrieval methods.
* Output shows scores.
* Output shows character counts.
* Output shows Obsidian links.

# Semantic Candidate Test

Scenario:

1. A semantic match has high similarity but no title/link/tag/folder support.
2. An exact folder project note has lower semantic similarity.

Expected:

* The exact folder project note outranks the semantic-only candidate.

# AI Tests

Do not require Ollama for normal CI.

Create an AI client interface so mock clients can be used.

Example:

```
type ChatClient interface {
    Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

type EmbeddingClient interface {
    Embed(ctx context.Context, input []string) ([]Embedding, error)
}
```

# Sample Vault Fixture

Create a small test vault:

```
testdata/sample-vault/
  Daily/
    2026-05-03.md
  Projects/
    Test App.md
  Templates/
    Daily.md
    Project.md
  Ideas/
    Local AI.md
  Resources/
    Ollama.md
```

Include:

* Missing links
* Tasks
* Frontmatter
* Messy daily note
* Weak template
* Project fragments
* Notes that should be semantically related
* Notes with overlapping chunks
* Notes where rollback conflicts can be tested

# Milestones

# Milestone 1 — Foundation

Goal:

Build the basic Go CLI, config, vault scanner, SQLite index, and environment checks.

Commands:

```
naudia init
naudia status
naudia scan
```

Tasks:

* Create Go project.
* Add Cobra command routing.
* Add Viper config loading.
* Add vault-local `.naudia/` folder.
* Add SQLite database.
* Add migrations.
* Add sqlite-vec detection.
* Add Markdown file scanner.
* Parse frontmatter, headings, links, tags, tasks.
* Add Ollama health check.
* Add Obsidian URI helper.
* Add optional Obsidian CLI health check.
* Add premium terminal output theme using Lip Gloss.
* Add tests.

Acceptance criteria:

* User can initialize Naudia in a vault.
* User can scan vault.
* User can view status.
* Database contains indexed note metadata.
* Obsidian links are generated correctly.
* No AI required yet except health check.
* Output looks polished.

# Milestone 2 — Vault Review

Goal:

Create useful non-mutating vault review.

Commands:

```
naudia review
naudia review --no-ai
```

Tasks:

* Build deterministic review engine.
* Detect orphan notes.
* Detect notes with no tags/links.
* Detect unresolved tasks.
* Detect daily notes with tasks.
* Detect root-level clutter.
* Detect weak titles.
* Detect template inconsistencies.
* Add optional AI review summary.
* Generate premium terminal report.
* Store review report in `.naudia/reports/`.

Acceptance criteria:

* Review produces useful output.
* Review does not mutate vault.
* Review works without embeddings.
* Review works with AI if Ollama is available.
* Review output looks good enough for screenshots.
* Review does not overstuff local model prompts.

# Milestone 3 — Proposal System and Robust Rollbacks

Goal:

Create reviewable proposals, safe apply logic, and robust rollback behavior that preserves unrelated user edits.

Commands:

```
naudia proposals
naudia show <id>
naudia apply <id>
naudia reject <id>
naudia rollback <id>
```

Tasks:

* Implement proposal schema.
* Implement structured proposal actions.
* Store proposals in DB.
* Store proposal JSON/patch files in `.naudia/proposals/`.
* Render proposal summaries.
* Generate forward and inverse diffs.
* Build TUI proposal list.
* Build diff viewer.
* Apply create/update/append/replace/insert/move/rename/delete actions.
* Store rich change records.
* Store affected ranges and anchors.
* Implement clean rollback.
* Implement drifted rollback.
* Implement three-way rollback.
* Implement rollback conflict detection.
* Write conflict artifacts to `.naudia/conflicts/`.
* Add stale hash checks.
* Add high-risk confirmation.
* Add force rollback with explicit confirmation.
* Add rollback tests for manual edits after apply.

Acceptance criteria:

* Proposals can be listed, shown, applied, rejected, and rolled back.
* Changed files can be restored when safe.
* Unrelated manual edits survive rollback.
* Same-section conflicts are detected.
* Stale proposals are blocked.
* High-risk proposals require explicit confirmation.
* Force rollback requires explicit confirmation.
* Diff view is premium.

# Milestone 4 — sqlite-vec Semantic Index

Goal:

Add local vector search.

Commands:

```
naudia scan
naudia status
```

Tasks:

* Add chunking.
* Add Ollama embedding generation.
* Add sqlite-vec vector storage.
* Add vector search helper.
* Add fallback behavior if sqlite-vec unavailable.
* Add vector status display.

Acceptance criteria:

* Changed chunks get embedded.
* Semantic search works locally.
* sqlite-vec status appears in `naudia status`.
* Failure to load sqlite-vec does not break core functionality.
* Semantic search is treated as candidate retrieval, not as final truth.

# Milestone 4.5 — Conservative Context Engine

Goal:

Build a context selection layer that prevents local model overload and reduces hallucinations.

Commands:

```
naudia ask "..."
naudia project "..." --show-context
naudia daily --show-context
```

Tasks:

* Implement context budget config.
* Implement context item model.
* Implement context ranking.
* Implement context deduplication.
* Implement command-specific context limits.
* Implement `--show-context`.
* Prefer exact/link/tag matches before semantic matches.
* Enforce max notes, chunks, and characters.
* Add debug output for dropped context.
* Add prompt instruction for insufficient context.

Acceptance criteria:

* Context is capped by default.
* Context selection is explainable.
* Semantic matches are not overused.
* Large vaults do not cause prompt stuffing.
* Local model prompts remain focused and source-grounded.
* Commands can report when context is insufficient.

# Milestone 5 — Daily Distiller

Goal:

Turn messy daily notes into useful structured memory.

Commands:

```
naudia daily
naudia daily --date YYYY-MM-DD
naudia daily --week
naudia daily --apply
naudia daily --show-context
```

Tasks:

* Detect daily note folder/date format from config.
* Read daily note.
* Extract explicit tasks deterministically.
* Build conservative context pack.
* Use AI to summarize and classify.
* Generate daily review Markdown.
* Create proposal to append or create review note.
* Optional proposal to create permanent notes.
* Optional proposal to move tasks into project TODO files.

Acceptance criteria:

* Daily review includes summary, decisions, tasks, project updates, ideas, notes to create, carry forward.
* Proposal cites source daily note.
* Proposal includes Obsidian link.
* User can apply generated proposal.
* Context can be inspected.
* Context stays within budget.

# Milestone 6 — Project Memory Compiler

Goal:

Create durable project memory from scattered notes.

Commands:

```
naudia project "<name>"
naudia project "<name>" --apply
naudia project "<name>" --show-context
```

Tasks:

* Find relevant notes by title/path/text/tag/link.
* Add sqlite-vec semantic retrieval.
* Rank exact/link/tag/folder matches above semantic-only matches.
* Build conservative context pack.
* Gather project context.
* Generate project summary.
* Generate suggested project files.
* Create proposal to create/update files.
* Avoid overwriting existing files without diff.

Acceptance criteria:

* Given a project name, Naudia finds relevant notes.
* Generates useful README/PLAN/TODO/DECISIONS/QUESTIONS/CONTEXT.
* Shows source notes.
* Shows Obsidian links.
* Creates safe proposal.
* Can apply proposal.
* Does not overstuff the model.
* Can show context.

# Milestone 7 — Link Suggestions

Goal:

Suggest useful graph improvements.

Commands:

```
naudia links
naudia links --note "<name>"
naudia links --folder "<folder>"
naudia links --apply
```

Tasks:

* Resolve note titles and aliases.
* Detect unlinked mentions.
* Detect orphan notes.
* Suggest related notes.
* Add semantic suggestions using sqlite-vec.
* Generate proposal to add links/related sections/frontmatter aliases.
* Keep proposals small and confidence-scored.

Acceptance criteria:

* Suggestions are specific and not spammy.
* Each suggestion includes reason and confidence.
* User can apply proposals.
* Affected notes include Obsidian links.

# Milestone 8 — Task and Decision Extraction

Goal:

Make vault action items and decisions visible.

Commands:

```
naudia tasks
naudia tasks --project "<name>"
naudia tasks --include-inferred
naudia decisions
naudia questions
```

Tasks:

* Extract Markdown tasks.
* Infer possible tasks with AI.
* Group tasks by project.
* Identify stale tasks.
* Extract decisions.
* Extract unresolved questions.
* Generate TODO/DECISIONS/QUESTIONS proposals.
* Prefer line-level and section-level operations over full-file rewrites.

Acceptance criteria:

* Explicit tasks are separated from inferred tasks.
* Decisions are sourced.
* Questions are sourced.
* Project-level files can be generated.
* Task changes can be rolled back safely.

# Milestone 9 — Structure and Template Doctor

Goal:

Help users improve vault organization and templates.

Commands:

```
naudia structure
naudia structure --propose
naudia templates
naudia templates --apply
naudia doctor
```

Tasks:

* Analyze folder structure.
* Identify messy root notes.
* Identify inconsistent templates.
* Propose improved folder structure.
* Propose template updates.
* Generate migration proposals.
* Use metadata-heavy context where possible.

Acceptance criteria:

* Structure recommendations are configurable.
* High-risk file moves require explicit confirmation.
* Template changes are reviewable.
* Terminal output is polished.
* Moves and renames have safe rollback behavior.

# Milestone 10 — Distribution and Release

Goal:

Make project easy to install and attractive on GitHub.

Tasks:

* Add GoReleaser.
* Create GitHub Actions release workflow.
* Build binaries for macOS, Linux, Windows.
* Create Homebrew tap.
* Create install script.
* Add README screenshots or terminal recordings.
* Add example vault.
* Add architecture docs.
* Add contribution guide.
* Add license.
* Add issue templates.
* Add roadmap.

Acceptance criteria:

* User can install with Homebrew.
* User can install with curl script.
* GitHub Releases include binaries.
* README clearly explains value.
* Project feels polished enough to star.

# Installation Requirements

The installation experience must be frictionless.

# Homebrew

Preferred:

```
brew install drakeafk/naudia/naudia
```

This requires:

* GoReleaser config
* Homebrew tap repo
* Formula generation

# Curl Script

Provide:

```
curl -fsSL https://raw.githubusercontent.com/drakeafk/naudia/main/scripts/install.sh | sh
```

Script should:

* Detect OS
* Detect architecture
* Download latest release
* Install binary to `/usr/local/bin` or `~/.local/bin`
* Print next steps

# Manual Install

Also support:

```
go install github.com/drakeafk/naudia/cmd/naudia@latest
```

Assuming public module path.

# GitHub Releases

Each release should include:

* naudia_Darwin_arm64.tar.gz
* naudia_Darwin_x86_64.tar.gz
* naudia_Linux_arm64.tar.gz
* naudia_Linux_x86_64.tar.gz
* naudia_Windows_x86_64.zip
* checksums.txt

# README Structure

README should include:

# Naudia

Naudia is your local AI steward for Obsidian.

## Why Naudia?

Most AI note tools let you chat with your notes.

Naudia helps you operate your vault.

## Why Naudia is not just another RAG chatbot

Explain:

* Steward
* Operator
* Diffs
* Rollbacks
* Determinism
* Local-first
* Conservative context
* Conflict-safe editing

## Features

* Vault review
* Daily note distillation
* Project memory compiler
* Task and decision extraction
* Link suggestions
* Structure and template doctor
* Safe proposal/apply/rollback flow
* Conflict detection
* Local Ollama support
* sqlite-vec semantic search
* Conservative context budgeting
* Obsidian URI links
* Optional Obsidian CLI integration
* Premium terminal UI

## Installation

Include Homebrew, curl, Go install, and manual binaries.

## Quick Start

```
ollama pull llama3.1:8b
ollama pull nomic-embed-text
naudia init
naudia scan
naudia review
```

## Example Workflow

```
naudia review
naudia proposals
naudia show 3
naudia apply 3
naudia rollback 3
```

## Commands

## Configuration

## Safety Model

## Context Model

## Architecture

## Roadmap

## Contributing

# Example README Demo

Include this:

```
naudia review
```

Example output:

```
╭─ Vault Review ──────────────────────────────╮
│ Naudia found 23 opportunities to improve    │
│ your vault.                                 │
│                                             │
│ Proposals    4 prepared                     │
│ Risk         low                            │
│ Next         naudia proposals               │
╰─────────────────────────────────────────────╯

Daily Notes
  • 8 daily notes contain reusable project knowledge
  • 4 project notes have unresolved tasks

Links
  • 11 notes mention existing concepts without links
  • 6 notes appear to be orphaned
```

Then:

```
naudia proposals
```

Example output:

```
╭─ Pending Proposals ─────────────────────────╮
│ 1  Distill this week's daily notes     low   │
│ 2  Add missing links to 8 notes        low   │
│ 3  Create Projects/Naudia/PLAN.md      low   │
│ 4  Improve Templates/Project.md        med   │
╰─────────────────────────────────────────────╯
```

Then:

```
naudia show 3
```

Then:

```
naudia apply 3
```

# CLI Tone

Naudia should sound capable and calm.

Good:

```
Naudia reviewed 324 notes and prepared 5 suggestions.
```

Good:

```
This proposal touches 18 files, so I marked it high risk.
Review carefully before applying.
```

Good:

```
Context budget reached. I selected the strongest 16 chunks and dropped 27 weaker candidates.
```

Avoid overly cute personality.

Avoid:

```
✨ Magical vault vibes activated!
```

The tone should be polished, trustworthy, and slightly assistant-like.

# Security and Privacy

Naudia should state clearly:

* It runs locally.
* It uses local Ollama models.
* It does not upload notes.
* It does not require cloud APIs.
* It reads Markdown files from the configured vault.
* It writes only after approval.
* It stores local metadata in `.naudia/`.
* It uses sqlite-vec locally for semantic search.
* It does not include telemetry in the initial version.

If telemetry is ever added later, it must be opt-in.

# Git Safety

If the vault is inside a Git repo, Naudia should detect this and optionally show Git status before applying changes.

Initial behavior:

* Detect `.git/`.
* Warn if working tree has changes before large proposal.
* Do not require Git.

Future command:

```
naudia apply 3 --git-check
```

# Performance Requirements

Initial targets:

* Scan 1,000 Markdown notes quickly.
* Incremental scan should skip unchanged files using content hashes.
* Embeddings should be incremental.
* Vector search should use sqlite-vec.
* Review and AI-powered commands must enforce conservative context budgets before calling Ollama.
* Large vaults should not crash.
* Rollback conflict detection should avoid destructive overwrites.

# Accessibility of Output

Terminal output should be readable.

Avoid giant walls of text by default.

For large output, show summary first and write full report to:

```
.naudia/reports/
```

Example:

```
Full report written to .naudia/reports/2026-05-03-vault-review.md
```

# Release Naming

Initial release:

```
v0.1.0 — Foundation
```

Suggested roadmap:

```
v0.1.0 scan/status/review
v0.2.0 proposals/apply/rollback
v0.3.0 sqlite-vec semantic index
v0.4.0 conservative context engine
v0.5.0 daily distiller
v0.6.0 project compiler
v0.7.0 links/tasks
v0.8.0 structure/templates
v1.0.0 stable CLI + docs + Homebrew + polished TUI
```

# Package Name

Preferred binary:

```
naudia
```

Preferred repo:

```
github.com/drakeafk/naudia
```

Preferred Homebrew:

```
brew install drakeafk/naudia/naudia
```

# Important Implementation Notes

# 1. Build deterministic functionality first.

Scanning, parsing, indexing, diffs, proposals, apply, and rollback matter more than AI in the foundation.

# 2. Keep AI calls behind interfaces.

This makes tests easy and allows future support for other local model backends.

# 3. Use sqlite-vec for semantic search.

This is a strong technical choice and should be highlighted in the README.

# 4. Keep proposal actions structured.

Do not rely only on raw text diffs.

Store structured actions and render diffs from them.

Structured actions are also required for safe rollback. Prefer append, section replacement, line updates, and task-specific operations over full-file rewrites.

# 5. Use file hashes before applying.

Prevent stale proposals from overwriting user changes.

# 6. Never hide destructive changes.

Moves, renames, deletes, and large rewrites must be explicit.

# 7. Use Obsidian URI links generously.

When a note is referenced, make it easy to open in Obsidian.

# 8. Make the terminal UI beautiful.

Premium terminal output is a product requirement, not polish.

# 9. Make it useful even with no embeddings.

Embeddings are a boost, not a hard dependency.

Semantic matches should never override exact note, link, tag, or folder evidence by default.

# 10. Source everything.

Project summaries, decisions, tasks, and recommendations should include source note paths.

Context should be intentionally small and explainable. Naudia should use `--show-context` to make retrieval transparent and should say when context is insufficient instead of guessing.

# 11. Do not require Obsidian CLI.

Use Obsidian URI scheme as the primary bridge and direct filesystem operations as the reliable core.

# 12. Make installation frictionless.

Homebrew, curl installer, GitHub Releases, and Go install should all be supported.

# 13. Rollback must preserve user trust.

Naudia must not wipe unrelated manual edits during rollback.

Full-file restore should only happen when the current file still matches the applied hash or when the user explicitly forces rollback.

# 14. Treat local model context as expensive.

Do not assume bigger prompts are better.

Prefer smaller, stronger context packs.

# Final Product Definition

Naudia is a local-first CLI AI operator for Obsidian.

It uses:

* Obsidian vault Markdown files
* Obsidian URI links
* Optional Obsidian CLI
* Ollama local models
* SQLite local index
* sqlite-vec semantic search
* Conservative context budgeting
* Charmbracelet terminal UI
* Safe proposal/apply/rollback system
* Conflict detection for drifted files

To provide:

* Vault review
* Daily note distillation
* Project memory compilation
* Link suggestions
* Task extraction
* Decision extraction
* Structure recommendations
* Template improvements
* Reviewable diffs
* Safe rollbacks that preserve unrelated user edits

The first public version should make users feel:

```
My vault is no longer a pile of notes.
It has a local assistant that can help maintain it.
```

That is the goal.
