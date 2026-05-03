# Architecture

Naudia is organized around local, reversible operations.

## Flow

1. `internal/vault` scans Markdown and extracts deterministic structure.
2. `internal/db` stores notes, links, tags, tasks, chunks, embeddings, proposals, changes, and conflicts.
3. `internal/contextpack` ranks and caps context before any model call.
4. `internal/ai` talks to Ollama through HTTP interfaces.
5. `internal/engines` generate reports and proposals.
6. `internal/proposals` validates, diffs, applies, and rolls back file actions.
7. `internal/ui` renders premium CLI and TUI surfaces.
8. `internal/cli` wires Cobra commands.

## Apply And Rollback

Naudia journals planned changes before mutating files. The journal is stored both in SQLite and under `.naudia/changes/`, so rollback metadata survives a bookkeeping failure after a file write.

Rollback never blindly restores a full file after drift. If the current file still matches the applied hash, Naudia can safely restore the previous content. If the file drifted, Naudia attempts strict inverse patch, anchor-based replacement, and conservative three-way rollback only when the applied block still exists exactly. If that is ambiguous, it writes a conflict artifact and leaves the file untouched.

## sqlite-vec

The migration `002_vec.sql` attempts to create a `vec0` table. If the extension is not available in the local SQLite runtime, Naudia skips the virtual table and continues with keyword search plus Go cosine search over locally stored embedding JSON. Core indexing and proposals do not depend on sqlite-vec availability.
