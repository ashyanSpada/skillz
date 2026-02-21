package skillz

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeSkill(t *testing.T, root string, name string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "---\nname: " + name + "\ndescription: Test skill\n---\nBody\n"
	if err := os.WriteFile(filepath.Join(dir, SkillMarkdown), []byte(content), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	return dir
}

func createZipSkill(t *testing.T, zipPath string, name string) {
	t.Helper()
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip file: %v", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	skillMD, err := writer.Create(SkillMarkdown)
	if err != nil {
		t.Fatalf("create SKILL.md: %v", err)
	}
	content := "---\nname: " + name + "\ndescription: Test skill from zip\n---\nZip Body\n"
	if _, err := skillMD.Write([]byte(content)); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	resource, err := writer.Create("text/hello.txt")
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}
	if _, err := resource.Write([]byte("hello")); err != nil {
		t.Fatalf("write resource: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
}

func TestRegistryDiscoversDirectorySkill(t *testing.T) {
	temp := t.TempDir()
	writeSkill(t, temp, "echo")

	registry := NewRegistry(temp)
	if err := registry.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	skill, err := registry.Get("echo")
	if err != nil {
		t.Fatalf("get skill: %v", err)
	}
	if skill.Metadata.Name != "echo" {
		t.Fatalf("unexpected name: %s", skill.Metadata.Name)
	}
}

func TestRegistryDiscoversZipSkill(t *testing.T) {
	temp := t.TempDir()
	zipPath := filepath.Join(temp, "my-skill.zip")
	createZipSkill(t, zipPath, "MySkill")

	registry := NewRegistry(temp)
	if err := registry.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	skill, err := registry.Get("myskill")
	if err != nil {
		t.Fatalf("get skill: %v", err)
	}
	if !skill.IsZip() {
		t.Fatalf("expected zip skill")
	}
	if !skill.HasResource("text/hello.txt") {
		t.Fatalf("expected text resource")
	}
}
