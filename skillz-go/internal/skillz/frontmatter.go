package skillz

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var frontMatterPattern = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)$`)

type SkillError struct {
	Code    string
	Message string
}

func (e SkillError) Error() string {
	return e.Message
}

func parseSkillMarkdown(raw string, source string) (SkillMetadata, string, error) {
	match := frontMatterPattern.FindStringSubmatch(raw)
	if len(match) != 3 {
		return SkillMetadata{}, "", SkillError{
			Code:    "validation_error",
			Message: fmt.Sprintf("%s must begin with YAML front matter delimited by '---'.", source),
		}
	}

	frontMatter := match[1]
	body := strings.TrimLeft(match[2], " \t\r\n")

	data := map[string]any{}
	if err := yaml.Unmarshal([]byte(frontMatter), &data); err != nil {
		return SkillMetadata{}, "", SkillError{
			Code:    "validation_error",
			Message: fmt.Sprintf("unable to parse YAML in %s: %v", source, err),
		}
	}

	name := strings.TrimSpace(toString(data["name"]))
	description := strings.TrimSpace(toString(data["description"]))
	if name == "" {
		return SkillMetadata{}, "", SkillError{Code: "validation_error", Message: fmt.Sprintf("front matter in %s is missing 'name'", source)}
	}
	if description == "" {
		return SkillMetadata{}, "", SkillError{Code: "validation_error", Message: fmt.Sprintf("front matter in %s is missing 'description'", source)}
	}

	allowedRaw := data["allowed-tools"]
	if allowedRaw == nil {
		allowedRaw = data["allowed_tools"]
	}

	allowedTools := []string{}
	switch value := allowedRaw.(type) {
	case string:
		for _, part := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				allowedTools = append(allowedTools, trimmed)
			}
		}
	case []any:
		for _, item := range value {
			trimmed := strings.TrimSpace(toString(item))
			if trimmed != "" {
				allowedTools = append(allowedTools, trimmed)
			}
		}
	}

	extra := map[string]any{}
	for key, value := range data {
		if key == "name" || key == "description" || key == "license" || key == "allowed-tools" || key == "allowed_tools" {
			continue
		}
		extra[key] = value
	}

	metadata := SkillMetadata{
		Name:         name,
		Description:  description,
		License:      strings.TrimSpace(toString(data["license"])),
		AllowedTools: allowedTools,
		Extra:        extra,
	}

	return metadata, body, nil
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}
