ALTER TABLE links ADD COLUMN target_heading TEXT;

CREATE TABLE IF NOT EXISTS apply_journal (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  proposal_id INTEGER NOT NULL,
  action_id TEXT NOT NULL,
  note_path TEXT NOT NULL,
  action_kind TEXT NOT NULL,
  planned_change_json TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'planned',
  created_at TEXT NOT NULL,
  completed_at TEXT,
  error TEXT,
  FOREIGN KEY (proposal_id) REFERENCES proposals(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_apply_journal_proposal ON apply_journal(proposal_id);
