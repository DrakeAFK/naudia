CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS vaults (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  path TEXT NOT NULL UNIQUE,
  obsidian_cli_enabled INTEGER NOT NULL DEFAULT 0,
  obsidian_uri_enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  vault_id INTEGER NOT NULL,
  path TEXT NOT NULL,
  title TEXT,
  aliases_json TEXT NOT NULL DEFAULT '[]',
  frontmatter_json TEXT,
  content_hash TEXT NOT NULL,
  word_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT,
  modified_at TEXT,
  indexed_at TEXT NOT NULL,
  FOREIGN KEY (vault_id) REFERENCES vaults(id) ON DELETE CASCADE,
  UNIQUE(vault_id, path)
);

CREATE TABLE IF NOT EXISTS headings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  note_id INTEGER NOT NULL,
  level INTEGER NOT NULL,
  text TEXT NOT NULL,
  line_number INTEGER,
  FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS links (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source_note_id INTEGER NOT NULL,
  target_raw TEXT NOT NULL,
  target_note_id INTEGER,
  link_text TEXT,
  line_number INTEGER,
  resolved INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY (source_note_id) REFERENCES notes(id) ON DELETE CASCADE,
  FOREIGN KEY (target_note_id) REFERENCES notes(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS tags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  note_id INTEGER NOT NULL,
  tag TEXT NOT NULL,
  FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  note_id INTEGER NOT NULL,
  text TEXT NOT NULL,
  completed INTEGER NOT NULL DEFAULT 0,
  line_number INTEGER,
  inferred INTEGER NOT NULL DEFAULT 0,
  project_guess TEXT,
  due_date_guess TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS chunks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  note_id INTEGER NOT NULL,
  chunk_index INTEGER NOT NULL,
  content TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  heading_context TEXT,
  token_estimate INTEGER,
  created_at TEXT NOT NULL,
  FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE,
  UNIQUE(note_id, chunk_index)
);

CREATE TABLE IF NOT EXISTS embeddings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  note_id INTEGER NOT NULL,
  chunk_id INTEGER NOT NULL,
  model TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  dimensions INTEGER NOT NULL,
  embedding_json TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE,
  FOREIGN KEY (chunk_id) REFERENCES chunks(id) ON DELETE CASCADE,
  UNIQUE(chunk_id, model)
);

CREATE TABLE IF NOT EXISTS vector_chunks (
  rowid INTEGER PRIMARY KEY,
  chunk_id INTEGER NOT NULL,
  note_id INTEGER NOT NULL,
  model TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  FOREIGN KEY (chunk_id) REFERENCES chunks(id) ON DELETE CASCADE,
  FOREIGN KEY (note_id) REFERENCES notes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS proposals (
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
  FOREIGN KEY (vault_id) REFERENCES vaults(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS changes (
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
  FOREIGN KEY (proposal_id) REFERENCES proposals(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_notes_vault_path ON notes(vault_id, path);
CREATE INDEX IF NOT EXISTS idx_links_source ON links(source_note_id);
CREATE INDEX IF NOT EXISTS idx_tags_note ON tags(note_id);
CREATE INDEX IF NOT EXISTS idx_tasks_note ON tasks(note_id);
CREATE INDEX IF NOT EXISTS idx_chunks_note ON chunks(note_id);
CREATE INDEX IF NOT EXISTS idx_proposals_vault_status ON proposals(vault_id, status);

