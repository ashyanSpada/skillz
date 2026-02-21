package skillz

import (
	"path"
	"regexp"
	"sort"
	"strings"
)

var slugPattern = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func slugify(value string) string {
	cleaned := strings.TrimSpace(strings.ToLower(value))
	cleaned = slugPattern.ReplaceAllString(cleaned, "-")
	cleaned = strings.Trim(cleaned, "-")
	if cleaned == "" {
		return "skill"
	}
	return cleaned
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func normalizeRelPath(relPath string) string {
	normalized := path.Clean(strings.ReplaceAll(relPath, "\\", "/"))
	normalized = strings.TrimPrefix(normalized, "./")
	return normalized
}
