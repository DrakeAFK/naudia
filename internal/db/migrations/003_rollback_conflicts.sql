CREATE TABLE IF NOT EXISTS rollback_conflicts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  proposal_id INTEGER NOT NULL,
  change_id INTEGER NOT NULL,
  note_path TEXT NOT NULL,
  reason TEXT NOT NULL,
  conflict_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (proposal_id) REFERENCES proposals(id) ON DELETE CASCADE,
  FOREIGN KEY (change_id) REFERENCES changes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rollback_conflicts_proposal ON rollback_conflicts(proposal_id);

