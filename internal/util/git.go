package util

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
)

type GitStatus struct {
	InRepo  bool
	Dirty   bool
	Summary string
}

func GitStatusForPath(path string) GitStatus {
	root := findGitRoot(path)
	if root == "" {
		return GitStatus{}
	}
	cmd := exec.Command("git", "-C", root, "status", "--short")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return GitStatus{InRepo: true, Dirty: true, Summary: "git status unavailable"}
	}
	summary := out.String()
	return GitStatus{InRepo: true, Dirty: len(summary) > 0, Summary: summary}
}

func findGitRoot(start string) string {
	abs, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for {
		if info, err := os.Stat(filepath.Join(abs, ".git")); err == nil && info.IsDir() {
			return abs
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
}
