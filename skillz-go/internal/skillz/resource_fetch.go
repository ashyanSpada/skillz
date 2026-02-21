package skillz

import (
	"encoding/base64"
	"mime"
	"path"
	"strings"
	"unicode/utf8"

	"net/url"
)

func BuildResourceURI(skill Skill, relPath string) string {
	slug := url.PathEscape(skill.Slug)
	relPath = normalizeRelPath(relPath)
	parts := strings.Split(relPath, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return "resource://skillz/" + slug + "/" + strings.Join(parts, "/")
}

func makeErrorResource(resourceURI string, message string) map[string]any {
	name := "invalid resource"
	if strings.HasPrefix(resourceURI, "resource://skillz/") {
		pathPart := strings.TrimPrefix(resourceURI, "resource://skillz/")
		if pathPart != "" {
			name = pathPart
		}
	}
	return map[string]any{
		"uri":       resourceURI,
		"name":      name,
		"mime_type": "text/plain",
		"content":   "Error: " + message,
		"encoding":  "utf-8",
	}
}

func FetchResourceJSON(registry *Registry, resourceURI string) map[string]any {
	const prefix = "resource://skillz/"
	if !strings.HasPrefix(resourceURI, prefix) {
		return makeErrorResource(resourceURI, "unsupported URI prefix. Expected resource://skillz/{skill-slug}/{path}")
	}

	remainder := strings.TrimPrefix(resourceURI, prefix)
	if remainder == "" {
		return makeErrorResource(resourceURI, "invalid resource URI format")
	}

	parts := strings.SplitN(remainder, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return makeErrorResource(resourceURI, "invalid resource URI format")
	}

	slug, err := url.PathUnescape(parts[0])
	if err != nil {
		return makeErrorResource(resourceURI, "invalid skill slug encoding")
	}
	relPath, err := url.PathUnescape(parts[1])
	if err != nil {
		return makeErrorResource(resourceURI, "invalid resource path encoding")
	}
	relPath = normalizeRelPath(relPath)
	if strings.HasPrefix(relPath, "/") || strings.Contains(relPath, "..") {
		return makeErrorResource(resourceURI, "invalid path: path traversal not allowed")
	}

	skill, err := registry.Get(slug)
	if err != nil {
		return makeErrorResource(resourceURI, "skill not found: "+slug)
	}
	if !skill.HasResource(relPath) {
		return makeErrorResource(resourceURI, "resource not found: "+relPath)
	}

	data, err := skill.OpenBytes(relPath)
	if err != nil {
		return makeErrorResource(resourceURI, "failed to read resource: "+err.Error())
	}

	mimeType := detectMimeType(relPath)
	content := ""
	encoding := "utf-8"
	if utf8.Valid(data) {
		content = string(data)
	} else {
		content = base64.StdEncoding.EncodeToString(data)
		encoding = "base64"
	}

	return map[string]any{
		"uri":       resourceURI,
		"name":      skill.ResourceName(relPath),
		"mime_type": mimeType,
		"content":   content,
		"encoding":  encoding,
	}
}

func detectMimeType(relPath string) any {
	ext := strings.ToLower(path.Ext(relPath))
	mimeType := mime.TypeByExtension(ext)
	if idx := strings.Index(mimeType, ";"); idx > -1 {
		mimeType = mimeType[:idx]
	}
	if mimeType != "" {
		return mimeType
	}

	switch ext {
	case ".py":
		return "text/x-python"
	case ".md":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	default:
		return nil
	}
}
