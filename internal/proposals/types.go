package proposals

import "time"

type ProposalType string
type ActionKind string
type RiskLevel string

const (
	TypeDailyDistillation   ProposalType = "daily_distillation"
	TypeProjectCompile      ProposalType = "project_compile"
	TypeLinkSuggestions     ProposalType = "link_suggestions"
	TypeTaskExtraction      ProposalType = "task_extraction"
	TypeStructureChange     ProposalType = "structure_change"
	TypeTemplateImprovement ProposalType = "template_improvement"
	TypeVaultReview         ProposalType = "vault_review"
	TypeNoteUpdate          ProposalType = "note_update"
	TypeNoteCreate          ProposalType = "note_create"
	TypeNoteMove            ProposalType = "note_move"
	TypeNoteRename          ProposalType = "note_rename"
	TypeNoteDelete          ProposalType = "note_delete"
)

const (
	ActionCreateNote          ActionKind = "create_note"
	ActionUpdateNote          ActionKind = "update_note"
	ActionRenameNote          ActionKind = "rename_note"
	ActionMoveNote            ActionKind = "move_note"
	ActionDeleteNote          ActionKind = "delete_note"
	ActionAppendToNote        ActionKind = "append_to_note"
	ActionReplaceSection      ActionKind = "replace_section"
	ActionInsertAfterHeading  ActionKind = "insert_after_heading"
	ActionInsertBeforeHeading ActionKind = "insert_before_heading"
	ActionUpdateLines         ActionKind = "update_lines"
	ActionRemoveLines         ActionKind = "remove_lines"
	ActionUpdateTaskStatus    ActionKind = "update_task_status"
	ActionAddFrontmatter      ActionKind = "add_frontmatter"
	ActionUpdateFrontmatter   ActionKind = "update_frontmatter"
)

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type SourceNote struct {
	Path        string `json:"path"`
	ObsidianURI string `json:"obsidian_uri,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

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

type ProposalAction struct {
	ID             string            `json:"id"`
	Kind           ActionKind        `json:"kind"`
	Path           string            `json:"path"`
	NewPath        string            `json:"new_path,omitempty"`
	Content        string            `json:"content,omitempty"`
	Heading        string            `json:"heading,omitempty"`
	HeadingLevel   int               `json:"heading_level,omitempty"`
	ExpectedHash   string            `json:"expected_hash,omitempty"`
	LineEdits      []LineEdit        `json:"line_edits,omitempty"`
	StartLine      int               `json:"start_line,omitempty"`
	EndLine        int               `json:"end_line,omitempty"`
	TaskCompleted  *bool             `json:"task_completed,omitempty"`
	Frontmatter    map[string]string `json:"frontmatter,omitempty"`
	AllowOverwrite bool              `json:"allow_overwrite,omitempty"`
}

type LineEdit struct {
	LineNumber int    `json:"line_number"`
	OldText    string `json:"old_text"`
	NewText    string `json:"new_text"`
}

type Change struct {
	ID              int64           `json:"id"`
	ProposalID      int64           `json:"proposal_id"`
	ActionID        string          `json:"action_id"`
	NotePath        string          `json:"note_path"`
	ActionKind      string          `json:"action_kind"`
	BeforeHash      string          `json:"before_hash"`
	AfterHash       string          `json:"after_hash"`
	PreviousContent string          `json:"previous_content"`
	AppliedContent  string          `json:"applied_content"`
	ForwardPatch    string          `json:"forward_patch"`
	InversePatch    string          `json:"inverse_patch"`
	AffectedRanges  []AffectedRange `json:"affected_ranges"`
	Anchors         []PatchAnchor   `json:"anchors"`
	AppliedAt       time.Time       `json:"applied_at"`
}

type AffectedRange struct {
	StartLineBefore int    `json:"start_line_before"`
	EndLineBefore   int    `json:"end_line_before"`
	StartLineAfter  int    `json:"start_line_after"`
	EndLineAfter    int    `json:"end_line_after"`
	BeforeTextHash  string `json:"before_text_hash"`
	AfterTextHash   string `json:"after_text_hash"`
}

type PatchAnchor struct {
	BeforeContext string `json:"before_context"`
	AfterContext  string `json:"after_context"`
	Heading       string `json:"heading"`
	SectionID     string `json:"section_id"`
}

type Conflict struct {
	ProposalID          int64  `json:"proposal_id"`
	ChangeID            int64  `json:"change_id"`
	NotePath            string `json:"note_path"`
	Reason              string `json:"reason"`
	OriginalChange      string `json:"original_change"`
	CurrentFileHash     string `json:"current_file_hash"`
	ExpectedAfterHash   string `json:"expected_after_hash"`
	SuggestedResolution string `json:"suggested_resolution"`
}
