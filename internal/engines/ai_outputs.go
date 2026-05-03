package engines

type aiReviewOutput struct {
	Summary string `json:"summary"`
	Issues  []struct {
		Category        string   `json:"category"`
		Severity        string   `json:"severity"`
		Description     string   `json:"description"`
		SourceNotes     []string `json:"source_notes"`
		SuggestedAction string   `json:"suggested_action"`
	} `json:"issues"`
	Proposals []struct {
		Title       string   `json:"title"`
		Type        string   `json:"type"`
		Summary     string   `json:"summary"`
		SourceNotes []string `json:"source_notes"`
		RiskLevel   string   `json:"risk_level"`
	} `json:"proposals"`
}

type aiDailyOutput struct {
	Date       string   `json:"date"`
	SourceNote string   `json:"source_note"`
	Summary    string   `json:"summary"`
	Decisions  []string `json:"decisions"`
	Tasks      []struct {
		Text         string `json:"text"`
		Explicit     bool   `json:"explicit"`
		ProjectGuess string `json:"project_guess"`
	} `json:"tasks"`
	ProjectUpdates []struct {
		Project string `json:"project"`
		Update  string `json:"update"`
	} `json:"project_updates"`
	IdeasWorthKeeping []string `json:"ideas_worth_keeping"`
	NotesToCreate     []struct {
		Title        string `json:"title"`
		Reason       string `json:"reason"`
		DraftContent string `json:"draft_content"`
	} `json:"notes_to_create"`
	CarryForward []string `json:"carry_forward"`
}

type aiProjectOutput struct {
	ProjectName          string   `json:"project_name"`
	SourceNotes          []string `json:"source_notes"`
	CurrentUnderstanding string   `json:"current_understanding"`
	Goals                []string `json:"goals"`
	Decisions            []string `json:"decisions"`
	OpenQuestions        []string `json:"open_questions"`
	Tasks                []string `json:"tasks"`
	Risks                []string `json:"risks"`
	SuggestedFiles       []struct {
		Path    string `json:"path"`
		Purpose string `json:"purpose"`
		Content string `json:"content"`
	} `json:"suggested_files"`
}

type aiLinkOutput struct {
	Suggestions []struct {
		SourceNote    string `json:"source_note"`
		TargetNote    string `json:"target_note"`
		Reason        string `json:"reason"`
		Confidence    string `json:"confidence"`
		SuggestedEdit string `json:"suggested_edit"`
	} `json:"suggestions"`
}

type aiStructureOutput struct {
	Summary              string   `json:"summary"`
	CurrentIssues        []string `json:"current_issues"`
	RecommendedStructure []struct {
		Path    string `json:"path"`
		Purpose string `json:"purpose"`
	} `json:"recommended_structure"`
	MigrationSuggestions []struct {
		From      string `json:"from"`
		To        string `json:"to"`
		Reason    string `json:"reason"`
		RiskLevel string `json:"risk_level"`
	} `json:"migration_suggestions"`
}
