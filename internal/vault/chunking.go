package vault

import (
	"strings"

	"github.com/drakeafk/naudia/internal/util"
)

func ChunkNote(note Note, chunkSize int, overlap int) []Chunk {
	if chunkSize <= 0 {
		chunkSize = 1200
	}
	if overlap < 0 {
		overlap = 0
	}
	sections := splitByHeadings(note.Content)
	var chunks []Chunk
	for _, section := range sections {
		text := strings.TrimSpace(section.Text)
		if text == "" {
			continue
		}
		if len(text) <= chunkSize {
			chunks = append(chunks, Chunk{
				NotePath:       note.Path,
				ChunkIndex:     len(chunks),
				Content:        text,
				ContentHash:    util.SHA256String(text),
				HeadingContext: section.Heading,
				TokenEstimate:  estimateTokens(text),
			})
			continue
		}
		start := 0
		for start < len(text) {
			end := start + chunkSize
			if end > len(text) {
				end = len(text)
			} else if idx := strings.LastIndex(text[start:end], "\n"); idx > chunkSize/2 {
				end = start + idx
			}
			part := strings.TrimSpace(text[start:end])
			if part != "" {
				chunks = append(chunks, Chunk{
					NotePath:       note.Path,
					ChunkIndex:     len(chunks),
					Content:        part,
					ContentHash:    util.SHA256String(part),
					HeadingContext: section.Heading,
					TokenEstimate:  estimateTokens(part),
				})
			}
			if end == len(text) {
				break
			}
			start = end - overlap
			if start < 0 {
				start = end
			}
		}
	}
	return chunks
}

type section struct {
	Heading string
	Text    string
}

func splitByHeadings(content string) []section {
	lines := strings.Split(content, "\n")
	var sections []section
	var current section
	for _, line := range lines {
		if m := headingRE.FindStringSubmatch(line); len(m) == 3 {
			if strings.TrimSpace(current.Text) != "" {
				sections = append(sections, current)
			}
			current = section{Heading: strings.TrimSpace(m[2]), Text: line + "\n"}
			continue
		}
		current.Text += line + "\n"
	}
	if strings.TrimSpace(current.Text) != "" {
		sections = append(sections, current)
	}
	if len(sections) == 0 && strings.TrimSpace(content) != "" {
		sections = append(sections, section{Text: content})
	}
	return sections
}

func estimateTokens(s string) int {
	words := len(strings.Fields(s))
	if words == 0 {
		return 0
	}
	return int(float64(words) * 1.3)
}
