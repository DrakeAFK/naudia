package db

type ScanStats struct {
	NotesIndexed     int `json:"notes_indexed"`
	NotesSkipped     int `json:"notes_skipped"`
	ChunksIndexed    int `json:"chunks_indexed"`
	EmbeddingsStored int `json:"embeddings_stored"`
}

type Status struct {
	VaultID          int64  `json:"vault_id"`
	VaultName        string `json:"vault_name"`
	VaultPath        string `json:"vault_path"`
	Notes            int    `json:"notes"`
	LastScan         string `json:"last_scan"`
	DatabasePath     string `json:"database_path"`
	VectorAvailable  bool   `json:"vector_available"`
	EmbeddingsStored int    `json:"embeddings_stored"`
	PendingProposals int    `json:"pending_proposals"`
	AppliedProposals int    `json:"applied_proposals"`
}

type NoteRow struct {
	ID          int64
	Path        string
	Title       string
	AliasesJSON string
	ContentHash string
	WordCount   int
	ModifiedAt  string
	IndexedAt   string
}

type ChunkRow struct {
	ID             int64
	NoteID         int64
	NotePath       string
	Title          string
	ChunkIndex     int
	Content        string
	ContentHash    string
	HeadingContext string
	TokenEstimate  int
}

type ProposalRecord struct {
	ID           int64
	VaultID      int64
	Type         string
	Title        string
	Summary      string
	Status       string
	ProposalJSON string
	PatchText    string
	CreatedAt    string
	AppliedAt    string
	RejectedAt   string
	RolledBackAt string
}

type ChangeRecord struct {
	ID                 int64
	ProposalID         int64
	ActionID           string
	NotePath           string
	ActionKind         string
	BeforeHash         string
	AfterHash          string
	PreviousContent    string
	AppliedContent     string
	ForwardPatch       string
	InversePatch       string
	AffectedRangesJSON string
	AnchorsJSON        string
	AppliedAt          string
}

type ConflictRecord struct {
	ID           int64
	ProposalID   int64
	ChangeID     int64
	NotePath     string
	Reason       string
	ConflictJSON string
	CreatedAt    string
}

type ApplyJournalRecord struct {
	ID                int64
	ProposalID        int64
	ActionID          string
	NotePath          string
	ActionKind        string
	PlannedChangeJSON string
	Status            string
	CreatedAt         string
	CompletedAt       string
	Error             string
}
