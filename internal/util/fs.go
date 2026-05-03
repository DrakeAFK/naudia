package util

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func EnsureNaudiaDirs(vaultPath string) error {
	for _, rel := range []string{
		".naudia",
		".naudia/proposals",
		".naudia/changes",
		".naudia/conflicts",
		".naudia/reports",
		".naudia/logs",
	} {
		if err := os.MkdirAll(filepath.Join(vaultPath, rel), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".naudia-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func ResolveInside(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", errors.New("absolute paths are not allowed")
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == string(filepath.Separator) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", errors.New("path escapes vault")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	full, err := filepath.Abs(filepath.Join(rootAbs, clean))
	if err != nil {
		return "", err
	}
	relToRoot, err := filepath.Rel(rootAbs, full)
	if err != nil {
		return "", err
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes vault")
	}
	return full, nil
}

func SlashPath(path string) string {
	return filepath.ToSlash(path)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
