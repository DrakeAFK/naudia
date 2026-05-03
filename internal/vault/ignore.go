package vault

import (
	"path"
	"strings"
)

type IgnoreMatcher struct {
	patterns []string
}

func NewIgnoreMatcher(patterns []string) IgnoreMatcher {
	if len(patterns) == 0 {
		patterns = []string{".naudia/**", ".git/**", "node_modules/**", ".obsidian/workspace*", ".obsidian/cache/**", ".DS_Store"}
	}
	return IgnoreMatcher{patterns: patterns}
}

func (m IgnoreMatcher) Ignored(rel string) bool {
	rel = strings.TrimPrefix(path.Clean(strings.ReplaceAll(rel, "\\", "/")), "./")
	base := path.Base(rel)
	for _, pattern := range m.patterns {
		pattern = strings.TrimPrefix(strings.ReplaceAll(pattern, "\\", "/"), "./")
		if pattern == "" {
			continue
		}
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "/**")
			if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
				return true
			}
		}
		if strings.HasSuffix(pattern, "*") && !strings.Contains(pattern, "/") {
			if ok, _ := path.Match(pattern, base); ok {
				return true
			}
		}
		if ok, _ := path.Match(pattern, rel); ok {
			return true
		}
		if ok, _ := path.Match(pattern, base); ok {
			return true
		}
	}
	return false
}
