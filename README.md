# Naudia

Naudia is your local AI steward for Obsidian.

Most AI note tools let you chat with your notes. Naudia helps you operate your vault.

It scans Markdown deterministically, builds a local SQLite index, optionally stores local embeddings for semantic search, prepares reviewable proposals, shows diffs, applies only approved changes, and records rollback data so edits can be reversed safely whenever possible.

Your notes stay local. Ollama runs local. SQLite and sqlite-vec run local. Naudia stores metadata under `.naudia/` inside the vault and does not include telemetry.

## Why Naudia is not just another RAG chatbot

A typical RAG chatbot embeds notes, retrieves chunks, and answers a question.

Naudia does more:

- Scans the vault deterministically.
- Builds a local structural index.
- Uses local embeddings for similarity when useful.
- Selects conservative, source-grounded context.
- Generates concrete proposals.
- Shows reviewable diffs.
- Requires approval before writing.
- Records rollback information.
- Detects rollback conflicts instead of overwriting drifted files.
- Opens notes through Obsidian URI links.

Naudia is built around stewardship over chat, diffs over magic, rollbacks over blind edits, and deterministic structure before AI judgment.

## Features

- Vault review
- Daily note distillation
- Project memory compiler
- Link suggestions
- Task extraction
- Structure and template analysis
- Proposal list/show/apply/reject flow
- Drift-aware rollback with conflict artifacts
- SQLite index with sqlite-vec migration and Go cosine fallback
- Ollama chat and embedding clients
- Conservative context budgeting with `--show-context`
- Obsidian URI helpers and optional CLI detection
- Bubble Tea and Lip Gloss terminal interface

## Installation

From source:

```sh
go install github.com/drakeafk/naudia/cmd/naudia@latest
```

Homebrew target:

```sh
brew install drakeafk/naudia/naudia
```

Curl installer target:

```sh
curl -fsSL https://raw.githubusercontent.com/drakeafk/naudia/main/scripts/install.sh | sh
```

## Quick Start

```sh
ollama pull llama3.1:8b
ollama pull nomic-embed-text
naudia init --vault /path/to/Obsidian/Main --vault-name Main
naudia scan
naudia status
naudia doctor
naudia review
naudia proposals
naudia show 1
naudia apply 1 --yes
naudia rollback 1
```

Expected first-run notes:

- `naudia scan` indexes Markdown and, when Ollama is online, stores embeddings locally.
- `sqlite-vec` is optional. If `status` says `Vector search  Go cosine fallback`, Naudia can still use stored Ollama embeddings for semantic candidates. It only means native sqlite-vec KNN is unavailable in the local SQLite runtime.
- If `doctor` says no embeddings are stored, run `naudia scan` again after confirming Ollama is online with `naudia status`.
- For a faster keyword-only scan, use `naudia scan --no-embeddings`.
- Naudia creates proposals only; it does not mutate notes until `naudia apply`.

Recommended first workflow:

```sh
naudia status
naudia scan
naudia doctor
naudia review --no-ai
naudia review
naudia proposals
```

Use `review --no-ai` first if you want to inspect the deterministic findings before Ollama-assisted suggestions.

## Commands

- `naudia init`
- `naudia status`
- `naudia scan`
- `naudia review`
- `naudia daily`
- `naudia project "<name>"`
- `naudia links`
- `naudia tasks`
- `naudia decisions`
- `naudia questions`
- `naudia structure`
- `naudia templates`
- `naudia doctor`
- `naudia proposals`
- `naudia show <proposal-id>`
- `naudia apply <proposal-id|all>`
- `naudia reject <proposal-id|all>`
- `naudia rollback <proposal-id>`
- `naudia undo <proposal-id>`
- `naudia ask "<question>"`

All output supports the global `--json`, `--markdown`, and `--output` flags where the command has structured data to emit.

## Safety Model

Naudia writes through proposals. Proposal actions are structured, stale-hash checked, and recorded as change rows with previous content, applied content, forward patch, inverse patch, affected ranges, and anchors.

Before mutating a file, Naudia also writes an apply journal entry to `.naudia/changes/` and `apply_journal` in SQLite. If a file write succeeds but later bookkeeping fails, rollback metadata still exists outside the changed note.

Rollback order:

1. Clean rollback if the current file still matches Naudia's applied hash.
2. Strict inverse patch or anchor replacement when the file drifted but Naudia's edited section is still exact.
3. Conservative three-way rollback when the applied block is still present exactly inside a drifted file.
4. Conflict artifact when rollback is ambiguous.
5. Full restore only with `--force` or when the file is unchanged since apply.

Conflict artifacts are written to `.naudia/conflicts/`.

## Context Model

Naudia treats local model context as expensive. Exact path, title, link, folder, tag, and text matches outrank semantic-only candidates. Semantic search is useful evidence, not proof.

Semantic retrieval has two local modes:

- `sqlite-vec native KNN`: used when the sqlite-vec virtual table is available.
- `Go cosine fallback`: used when sqlite-vec is unavailable but embeddings are stored in SQLite.

Both modes stay local. If there are no stored embeddings, Naudia still works with deterministic structure, links, tags, folders, and keyword search.

Use:

```sh
naudia project "Naudia" --show-context
naudia daily --show-context
naudia ask "What did I decide about naming?" --show-context
```

## Configuration

Vault-local config lives at:

```text
<vault>/.naudia/config.toml
```

Global config can live at:

```text
~/.config/naudia/config.toml
```

Vault-local values override global values.

## Architecture

Core packages:

- `internal/vault`: scanning, parsing, chunking, ignore rules.
- `internal/db`: SQLite schema, migrations, indexing, proposals, changes, conflicts, embeddings.
- `internal/ai`: Ollama chat and embeddings.
- `internal/contextpack`: conservative context ranking and packing.
- `internal/proposals`: structured actions, diffs, apply, rollback, conflicts.
- `internal/engines`: review, daily, links, tasks, project, structure, templates, ask.
- `internal/ui`: Bubble Tea, Lip Gloss, Glamour terminal UI.
- `internal/cli`: Cobra command wiring.

## Privacy

Naudia reads Markdown files from the configured vault, writes only after approval, stores local metadata in `.naudia/`, and uses local Ollama models. It does not upload notes and does not include telemetry.
