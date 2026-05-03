package vault

import "testing"

func TestParseNoteSkipsCodeBlockLinksAndTasks(t *testing.T) {
	content := `---
title: Test Note
aliases: [Alias One]
---

# Heading

[[Real Link]]
#real/tag
- [ ] Real task

` + "```" + `
[[Ignored Link]]
#ignored/tag
- [ ] ignored task
` + "```" + `
`
	note := ParseNote("Test Note.md", "/tmp/Test Note.md", content)
	if note.Title != "Test Note" {
		t.Fatalf("title = %q", note.Title)
	}
	if len(note.Aliases) != 1 || note.Aliases[0] != "Alias One" {
		t.Fatalf("aliases = %#v", note.Aliases)
	}
	if len(note.Links) != 1 || note.Links[0].TargetRaw != "Real Link" {
		t.Fatalf("links = %#v", note.Links)
	}
	if len(note.Tags) != 1 || note.Tags[0].Name != "real/tag" {
		t.Fatalf("tags = %#v", note.Tags)
	}
	if len(note.Tasks) != 1 || note.Tasks[0].Text != "Real task" {
		t.Fatalf("tasks = %#v", note.Tasks)
	}
}

func TestResolveLinksUsesAlias(t *testing.T) {
	notes := []Note{
		ParseNote("A.md", "/tmp/A.md", "[[Alias B]]"),
		ParseNote("B.md", "/tmp/B.md", "---\naliases: [Alias B]\n---\n# Bee"),
	}
	notes = ResolveLinks(notes)
	if !notes[0].Links[0].Resolved || notes[0].Links[0].TargetPath != "B.md" {
		t.Fatalf("link was not resolved through alias: %#v", notes[0].Links[0])
	}
}

func TestResolveLinksRequiresExistingHeading(t *testing.T) {
	notes := []Note{
		ParseNote("A.md", "/tmp/A.md", "[[B#Real Heading]]\n[[B#Missing Heading]]"),
		ParseNote("B.md", "/tmp/B.md", "# B\n\n## Real Heading\n"),
	}
	notes = ResolveLinks(notes)
	if !notes[0].Links[0].Resolved || notes[0].Links[0].TargetHeading != "Real Heading" {
		t.Fatalf("heading link should resolve: %#v", notes[0].Links[0])
	}
	if notes[0].Links[1].Resolved {
		t.Fatalf("missing heading link should not resolve: %#v", notes[0].Links[1])
	}
}
