package vault

import (
	"os"
	"path/filepath"
	"strings"
)

type Scanner struct {
	Root   string
	Ignore IgnoreMatcher
	Folder string
}

func NewScanner(root string, patterns []string) Scanner {
	return Scanner{Root: root, Ignore: NewIgnoreMatcher(patterns)}
}

func (s Scanner) WithFolder(folder string) Scanner {
	s.Folder = strings.Trim(strings.ReplaceAll(folder, "\\", "/"), "/")
	return s
}

func (s Scanner) Scan() (ScanResult, error) {
	root := s.Root
	if s.Folder != "" {
		root = filepath.Join(root, filepath.FromSlash(s.Folder))
	}
	result := ScanResult{VaultPath: s.Root}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			result.Warnings = append(result.Warnings, err.Error())
			return nil
		}
		rel, err := filepath.Rel(s.Root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if s.Ignore.Ignored(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			result.Skipped++
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(d.Name())) != ".md" {
			result.Skipped++
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			result.Warnings = append(result.Warnings, err.Error())
			return nil
		}
		note := ParseNote(rel, path, string(b))
		if info, err := d.Info(); err == nil {
			note.ModifiedAt = info.ModTime()
			note.CreatedAt = info.ModTime()
		}
		result.Notes = append(result.Notes, note)
		return nil
	})
	if err != nil {
		return result, err
	}
	result.Notes = ResolveLinks(result.Notes)
	return result, nil
}
