package obsidian

import "testing"

func TestBuildOpenNoteURI(t *testing.T) {
	got := BuildOpenNoteURI("Main Vault", "Projects/Test App.md")
	want := "obsidian://open?file=Projects%2FTest+App.md&vault=Main+Vault"
	if got != want {
		t.Fatalf("BuildOpenNoteURI() = %q, want %q", got, want)
	}
}

func TestTerminalLinkDisabled(t *testing.T) {
	if got := TerminalLink("Note", "obsidian://open?vault=Main", false); got != "Note" {
		t.Fatalf("TerminalLink disabled = %q", got)
	}
}
