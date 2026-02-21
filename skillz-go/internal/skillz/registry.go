package skillz

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type Registry struct {
	Root         string
	skillsBySlug map[string]Skill
	skillsByName map[string]Skill
}

func NewRegistry(root string) *Registry {
	return &Registry{
		Root:         root,
		skillsBySlug: map[string]Skill{},
		skillsByName: map[string]Skill{},
	}
}

func (r *Registry) Skills() []Skill {
	skills := make([]Skill, 0, len(r.skillsBySlug))
	for _, skill := range r.skillsBySlug {
		skills = append(skills, skill)
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Slug < skills[j].Slug
	})
	return skills
}

func (r *Registry) Get(slug string) (Skill, error) {
	skill, ok := r.skillsBySlug[slug]
	if !ok {
		return Skill{}, SkillError{Code: "skill_error", Message: fmt.Sprintf("unknown skill '%s'", slug)}
	}
	return skill, nil
}

func (r *Registry) Load() error {
	stat, err := os.Stat(r.Root)
	if err != nil || !stat.IsDir() {
		return SkillError{Code: "skill_error", Message: fmt.Sprintf("skills root %s does not exist or is not a directory", r.Root)}
	}

	r.skillsBySlug = map[string]Skill{}
	r.skillsByName = map[string]Skill{}

	absRoot, err := filepath.Abs(r.Root)
	if err != nil {
		return err
	}
	return r.scanDirectory(absRoot)
}

func (r *Registry) scanDirectory(directory string) error {
	skillMD := filepath.Join(directory, SkillMarkdown)
	if stat, err := os.Stat(skillMD); err == nil && !stat.IsDir() {
		r.registerDirSkill(directory, skillMD)
		return nil
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		if entry.IsDir() {
			nextDir := filepath.Join(directory, entry.Name())
			_ = r.scanDirectory(nextDir)
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".zip" || ext == ".skill" {
			zipPath := filepath.Join(directory, entry.Name())
			r.tryRegisterZipSkill(zipPath)
		}
	}
	return nil
}

func (r *Registry) registerDirSkill(directory string, skillMD string) {
	raw, err := os.ReadFile(skillMD)
	if err != nil {
		return
	}
	metadata, body, err := parseSkillMarkdown(string(raw), skillMD)
	if err != nil {
		return
	}

	slug := slugify(metadata.Name)
	if _, exists := r.skillsBySlug[slug]; exists {
		return
	}
	if _, exists := r.skillsByName[metadata.Name]; exists {
		return
	}

	resources := map[string]string{}
	_ = filepath.WalkDir(directory, func(current string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		if filepath.Clean(current) == filepath.Clean(skillMD) {
			return nil
		}
		rel, err := filepath.Rel(directory, current)
		if err != nil {
			return nil
		}
		resources[filepath.ToSlash(rel)] = current
		return nil
	})

	skill := Skill{
		Slug:         slug,
		Directory:    directory,
		Instructions: body,
		Metadata:     metadata,
		Resources:    resources,
	}
	r.skillsBySlug[slug] = skill
	r.skillsByName[metadata.Name] = skill
}

func (r *Registry) tryRegisterZipSkill(zipPath string) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return
	}
	defer reader.Close()

	members := map[string]*zip.File{}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		members[file.Name] = file
	}

	skillMDPath := ""
	zipRootPrefix := ""
	if _, ok := members[SkillMarkdown]; ok {
		skillMDPath = SkillMarkdown
	} else {
		topDirs := map[string]struct{}{}
		for _, file := range reader.File {
			if strings.Contains(file.Name, "/") {
				topDir := strings.SplitN(file.Name, "/", 2)[0]
				topDirs[topDir] = struct{}{}
			}
		}
		if len(topDirs) == 1 {
			for topDir := range topDirs {
				candidate := path.Join(topDir, SkillMarkdown)
				if _, ok := members[candidate]; ok {
					skillMDPath = candidate
					zipRootPrefix = topDir + "/"
				}
			}
		}
	}

	if skillMDPath == "" {
		return
	}

	skillMDFile := members[skillMDPath]
	rc, err := skillMDFile.Open()
	if err != nil {
		return
	}
	skillMDBytes, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return
	}

	metadata, body, err := parseSkillMarkdown(string(skillMDBytes), zipPath+":"+skillMDPath)
	if err != nil {
		return
	}

	slug := slugify(metadata.Name)
	if _, exists := r.skillsBySlug[slug]; exists {
		return
	}
	if _, exists := r.skillsByName[metadata.Name]; exists {
		return
	}

	zipMembers := map[string]struct{}{}
	resources := map[string]string{}
	for name := range members {
		normalizedName := name
		if zipRootPrefix != "" {
			if !strings.HasPrefix(name, zipRootPrefix) {
				continue
			}
			normalizedName = strings.TrimPrefix(name, zipRootPrefix)
		}
		zipMembers[normalizedName] = struct{}{}
		if normalizedName == SkillMarkdown {
			continue
		}
		if strings.Contains(normalizedName, "__MACOSX/") || strings.HasSuffix(normalizedName, ".DS_Store") {
			continue
		}
		resources[normalizedName] = normalizedName
	}

	skill := Skill{
		Slug:          slug,
		Directory:     filepath.Dir(zipPath),
		Instructions:  body,
		Metadata:      metadata,
		Resources:     resources,
		ZipPath:       zipPath,
		ZipRootPrefix: zipRootPrefix,
		zipMembers:    zipMembers,
	}
	r.skillsBySlug[slug] = skill
	r.skillsByName[metadata.Name] = skill
}

func (s Skill) OpenBytes(relPath string) ([]byte, error) {
	relPath = normalizeRelPath(relPath)
	if s.IsZip() {
		reader, err := zip.OpenReader(s.ZipPath)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		memberPath := s.ZipRootPrefix + relPath
		for _, file := range reader.File {
			if file.Name != memberPath {
				continue
			}
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
		return nil, os.ErrNotExist
	}

	fullPath, ok := s.Resources[relPath]
	if !ok {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(fullPath)
}

func (s Skill) HasResource(relPath string) bool {
	relPath = normalizeRelPath(relPath)
	_, ok := s.Resources[relPath]
	return ok
}
