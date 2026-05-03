package obsidian

import (
	"os"
	"path/filepath"
)

func DeriveVaultName(path string) string {
	clean := filepath.Clean(path)
	name := filepath.Base(clean)
	if name == "." || name == string(filepath.Separator) {
		if wd, err := os.Getwd(); err == nil {
			return filepath.Base(wd)
		}
		return "Vault"
	}
	return name
}

func LooksLikeVault(path string) bool {
	found := false
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".naudia" || d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(d.Name()) == ".md" {
			found = true
		}
		return nil
	})
	return found
}
