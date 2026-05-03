package obsidian

import (
	"context"
	"os/exec"
)

func CLIAvailable(command string) bool {
	if command == "" {
		return false
	}
	_, err := exec.LookPath(command)
	return err == nil
}

func OpenNoteWithCLI(ctx context.Context, command string, path string) error {
	cmd := exec.CommandContext(ctx, command, path)
	return cmd.Start()
}
