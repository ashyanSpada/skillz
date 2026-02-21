package skillz

import "path/filepath"

const SkillMarkdown = "SKILL.md"

type SkillMetadata struct {
	Name         string
	Description  string
	License      string
	AllowedTools []string
	Extra        map[string]any
}

type Skill struct {
	Slug          string
	Directory     string
	Instructions  string
	Metadata      SkillMetadata
	Resources     map[string]string
	ZipPath       string
	ZipRootPrefix string
	zipMembers    map[string]struct{}
}

type ResourceMetadata struct {
	URI      string `json:"uri"`
	Name     string `json:"name"`
	MIMEType any    `json:"mime_type"`
}

func (s Skill) IsZip() bool {
	return s.ZipPath != ""
}

func (s Skill) ResourceName(relPath string) string {
	return filepath.ToSlash(s.Slug + "/" + relPath)
}
