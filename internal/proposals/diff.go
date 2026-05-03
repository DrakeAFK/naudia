package proposals

import (
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func UnifiedPatch(path, before, after string) string {
	dmp := diffmatchpatch.New()
	patches := dmp.PatchMake(before, after)
	text := dmp.PatchToText(patches)
	if text == "" && before == after {
		return ""
	}
	var b strings.Builder
	b.WriteString("--- ")
	b.WriteString(path)
	b.WriteString("\n+++ ")
	b.WriteString(path)
	b.WriteString("\n")
	b.WriteString(text)
	return b.String()
}

func TextPatch(before, after string) string {
	dmp := diffmatchpatch.New()
	return dmp.PatchToText(dmp.PatchMake(before, after))
}

func ApplyTextPatch(patchText, current string) (string, bool) {
	if strings.TrimSpace(patchText) == "" {
		return current, true
	}
	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(patchText)
	if err != nil {
		return current, false
	}
	result, applied := dmp.PatchApply(patches, current)
	for _, ok := range applied {
		if !ok {
			return current, false
		}
	}
	return result, true
}
