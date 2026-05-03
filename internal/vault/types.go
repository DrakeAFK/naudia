package vault

import "time"

type Note struct {
	Path        string         `json:"path"`
	AbsPath     string         `json:"abs_path"`
	Title       string         `json:"title"`
	Aliases     []string       `json:"aliases"`
	Frontmatter map[string]any `json:"frontmatter"`
	Content     string         `json:"content"`
	ContentHash string         `json:"content_hash"`
	WordCount   int            `json:"word_count"`
	CreatedAt   time.Time      `json:"created_at"`
	ModifiedAt  time.Time      `json:"modified_at"`
	Headings    []Heading      `json:"headings"`
	Links       []Link         `json:"links"`
	Tags        []Tag          `json:"tags"`
	Tasks       []Task         `json:"tasks"`
}

type Heading struct {
	Level      int    `json:"level"`
	Text       string `json:"text"`
	LineNumber int    `json:"line_number"`
}

type Link struct {
	Kind       string `json:"kind"`
	TargetRaw  string `json:"target_raw"`
	LinkText   string `json:"link_text"`
	LineNumber int    `json:"line_number"`
	Resolved   bool   `json:"resolved"`
	TargetPath string `json:"target_path,omitempty"`
}

type Tag struct {
	Name       string `json:"name"`
	LineNumber int    `json:"line_number"`
}

type Task struct {
	Text         string `json:"text"`
	Completed    bool   `json:"completed"`
	LineNumber   int    `json:"line_number"`
	Inferred     bool   `json:"inferred"`
	ProjectGuess string `json:"project_guess,omitempty"`
	DueDateGuess string `json:"due_date_guess,omitempty"`
}

type Chunk struct {
	NotePath       string `json:"note_path"`
	ChunkIndex     int    `json:"chunk_index"`
	Content        string `json:"content"`
	ContentHash    string `json:"content_hash"`
	HeadingContext string `json:"heading_context"`
	TokenEstimate  int    `json:"token_estimate"`
}

type ScanResult struct {
	VaultPath string   `json:"vault_path"`
	Notes     []Note   `json:"notes"`
	Skipped   int      `json:"skipped"`
	Warnings  []string `json:"warnings"`
}
