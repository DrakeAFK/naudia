package contextpack

type Item struct {
	NotePath        string  `json:"note_path"`
	ObsidianURI     string  `json:"obsidian_uri"`
	Heading         string  `json:"heading"`
	Excerpt         string  `json:"excerpt"`
	RetrievalMethod string  `json:"retrieval_method"`
	Reason          string  `json:"reason"`
	Score           float64 `json:"score"`
	CharCount       int     `json:"char_count"`
}

type Budget struct {
	MaxNotes               int
	MaxChunks              int
	MaxCharsTotal          int
	MaxCharsPerNote        int
	MaxSemanticMatches     int
	MinSimilarityThreshold float64
	IncludeFullNotes       bool
	PreferHeadings         bool
}

type Pack struct {
	Items          []Item `json:"items"`
	Dropped        int    `json:"dropped"`
	SelectedChars  int    `json:"selected_chars"`
	BudgetExceeded bool   `json:"budget_exceeded"`
}
