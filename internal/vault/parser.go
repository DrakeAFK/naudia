package vault

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/drakeafk/naudia/internal/util"
)

var (
	headingRE      = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)
	wikiLinkRE     = regexp.MustCompile(`\[\[([^\]|#]+)(?:#([^\]|]+))?(?:\|([^\]]+))?\]\]`)
	markdownLinkRE = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	tagRE          = regexp.MustCompile(`(^|\s)#([A-Za-z0-9][A-Za-z0-9_/-]*)`)
	taskRE         = regexp.MustCompile(`^\s*[-*+]\s+\[([ xX])\]\s+(.+?)\s*$`)
)

func ParseNote(relPath string, absPath string, content string) Note {
	frontmatter, bodyStart := parseFrontmatter(content)
	lines := strings.Split(content, "\n")
	title := frontmatterString(frontmatter, "title")
	aliases := frontmatterStrings(frontmatter, "aliases")
	var headings []Heading
	var links []Link
	var tags []Tag
	var tasks []Task
	inFence := false
	for i, line := range lines {
		lineNo := i + 1
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence || i < bodyStart {
			continue
		}
		if m := headingRE.FindStringSubmatch(line); len(m) == 3 {
			text := strings.TrimSpace(strings.TrimRight(m[2], "#"))
			h := Heading{Level: len(m[1]), Text: text, LineNumber: lineNo}
			headings = append(headings, h)
			if title == "" && h.Level == 1 {
				title = h.Text
			}
		}
		for _, m := range wikiLinkRE.FindAllStringSubmatch(line, -1) {
			linkText := ""
			targetHeading := ""
			if len(m) > 2 {
				targetHeading = strings.TrimSpace(m[2])
			}
			if len(m) > 3 {
				linkText = strings.TrimSpace(m[3])
			}
			links = append(links, Link{
				Kind:          "wiki",
				TargetRaw:     strings.TrimSpace(m[1]),
				TargetHeading: targetHeading,
				LinkText:      linkText,
				LineNumber:    lineNo,
			})
		}
		for _, m := range markdownLinkRE.FindAllStringSubmatch(line, -1) {
			target := strings.TrimSpace(m[2])
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "obsidian://") || strings.HasPrefix(target, "#") {
				continue
			}
			targetHeading := ""
			if before, after, ok := strings.Cut(target, "#"); ok {
				target = before
				targetHeading = after
			}
			links = append(links, Link{
				Kind:          "markdown",
				TargetRaw:     target,
				TargetHeading: targetHeading,
				LinkText:      strings.TrimSpace(m[1]),
				LineNumber:    lineNo,
			})
		}
		for _, m := range tagRE.FindAllStringSubmatch(line, -1) {
			tag := strings.TrimSpace(m[2])
			if tag != "" {
				tags = append(tags, Tag{Name: tag, LineNumber: lineNo})
			}
		}
		if m := taskRE.FindStringSubmatch(line); len(m) == 3 {
			tasks = append(tasks, Task{
				Text:       strings.TrimSpace(m[2]),
				Completed:  strings.EqualFold(m[1], "x"),
				LineNumber: lineNo,
			})
		}
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	}
	return Note{
		Path:        util.SlashPath(relPath),
		AbsPath:     absPath,
		Title:       title,
		Aliases:     aliases,
		Frontmatter: frontmatter,
		Content:     content,
		ContentHash: util.SHA256String(content),
		WordCount:   len(strings.Fields(content)),
		Headings:    headings,
		Links:       links,
		Tags:        tags,
		Tasks:       tasks,
	}
}

func ResolveLinks(notes []Note) []Note {
	byPath := map[string]string{}
	byStem := map[string][]string{}
	byAlias := map[string][]string{}
	headingsByPath := map[string]map[string]bool{}
	for _, n := range notes {
		key := strings.ToLower(strings.TrimSuffix(n.Path, ".md"))
		byPath[strings.ToLower(n.Path)] = n.Path
		byPath[key] = n.Path
		stem := strings.ToLower(strings.TrimSuffix(filepath.Base(n.Path), ".md"))
		byStem[stem] = append(byStem[stem], n.Path)
		for _, alias := range n.Aliases {
			alias = strings.ToLower(strings.TrimSpace(alias))
			if alias != "" {
				byAlias[alias] = append(byAlias[alias], n.Path)
			}
		}
		headingsByPath[n.Path] = map[string]bool{}
		for _, heading := range n.Headings {
			headingsByPath[n.Path][normalizeHeading(heading.Text)] = true
		}
	}
	for i := range notes {
		for j := range notes[i].Links {
			target := strings.TrimSpace(notes[i].Links[j].TargetRaw)
			target = strings.TrimSuffix(target, ".md")
			targetLower := strings.ToLower(target)
			var resolved string
			if p, ok := byPath[targetLower+".md"]; ok {
				resolved = p
			} else if p, ok := byPath[targetLower]; ok {
				resolved = p
			} else if list := byStem[targetLower]; len(list) == 1 {
				resolved = list[0]
			} else if list := byAlias[targetLower]; len(list) == 1 {
				resolved = list[0]
			}
			if resolved != "" {
				if notes[i].Links[j].TargetHeading != "" && !headingsByPath[resolved][normalizeHeading(notes[i].Links[j].TargetHeading)] {
					continue
				}
				notes[i].Links[j].Resolved = true
				notes[i].Links[j].TargetPath = resolved
			}
		}
	}
	return notes
}

func normalizeHeading(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Trim(s, "#")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func ParseFrontmatterJSON(frontmatter map[string]any) string {
	if len(frontmatter) == 0 {
		return "{}"
	}
	b, err := json.Marshal(frontmatter)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func parseFrontmatter(content string) (map[string]any, int) {
	result := map[string]any{}
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return result, 0
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return result, 0
	}
	var currentKey string
	for i := 1; i < end; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- ") && currentKey != "" {
			existing, _ := result[currentKey].([]string)
			result[currentKey] = append(existing, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		currentKey = key
		if value == "" {
			result[key] = []string{}
			continue
		}
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			inner := strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
			var items []string
			for _, item := range strings.Split(inner, ",") {
				item = strings.Trim(strings.TrimSpace(item), `"'`)
				if item != "" {
					items = append(items, item)
				}
			}
			result[key] = items
			continue
		}
		if b, err := strconv.ParseBool(value); err == nil {
			result[key] = b
			continue
		}
		if n, err := strconv.Atoi(value); err == nil {
			result[key] = n
			continue
		}
		result[key] = strings.Trim(value, `"'`)
	}
	return result, end + 1
}

func frontmatterString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func frontmatterStrings(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		return []string{t}
	default:
		return nil
	}
}
