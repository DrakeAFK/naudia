package obsidian

import (
	"net/url"
	"os"
	"strings"
)

func BuildOpenNoteURI(vaultName string, filePath string) string {
	q := url.Values{}
	q.Set("vault", vaultName)
	q.Set("file", strings.TrimPrefix(filePath, "/"))
	return "obsidian://open?" + q.Encode()
}

func BuildSearchURI(vaultName string, query string) string {
	q := url.Values{}
	q.Set("vault", vaultName)
	q.Set("query", query)
	return "obsidian://search?" + q.Encode()
}

func TerminalLink(label string, uri string, enabled bool) string {
	if !enabled || uri == "" || os.Getenv("NO_COLOR") != "" {
		return label
	}
	return "\033]8;;" + uri + "\033\\" + label + "\033]8;;\033\\"
}
