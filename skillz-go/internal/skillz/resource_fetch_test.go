package skillz

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func writeSkillWithResources(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "testskill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	skillMD := "---\nname: TestSkill\ndescription: Test skill with resources\n---\nBody\n"
	if err := os.WriteFile(filepath.Join(dir, SkillMarkdown), []byte(skillMD), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "script.py"), []byte("print('hello')"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data.bin"), []byte{0xff, 0xfe, 0x00, 0x01, 0x80, 0x90}, 0o644); err != nil {
		t.Fatalf("write data: %v", err)
	}
}

func TestFetchTextResource(t *testing.T) {
	temp := t.TempDir()
	writeSkillWithResources(t, temp)

	registry := NewRegistry(temp)
	if err := registry.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	result := FetchResourceJSON(registry, "resource://skillz/testskill/script.py")
	if result["encoding"] != "utf-8" {
		t.Fatalf("expected utf-8 encoding")
	}
	if result["content"] != "print('hello')" {
		t.Fatalf("unexpected content: %v", result["content"])
	}
}

func TestFetchBinaryResource(t *testing.T) {
	temp := t.TempDir()
	writeSkillWithResources(t, temp)

	registry := NewRegistry(temp)
	if err := registry.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	result := FetchResourceJSON(registry, "resource://skillz/testskill/data.bin")
	if result["encoding"] != "base64" {
		t.Fatalf("expected base64 encoding")
	}
	decoded, err := base64.StdEncoding.DecodeString(result["content"].(string))
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	expected := []byte{0xff, 0xfe, 0x00, 0x01, 0x80, 0x90}
	if string(decoded) != string(expected) {
		t.Fatalf("unexpected binary content")
	}
}

func TestFetchRejectsPathTraversal(t *testing.T) {
	temp := t.TempDir()
	writeSkillWithResources(t, temp)

	registry := NewRegistry(temp)
	if err := registry.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}

	result := FetchResourceJSON(registry, "resource://skillz/testskill/../../../etc/passwd")
	content := result["content"].(string)
	if content == "" || content[:6] != "Error:" {
		t.Fatalf("expected error content")
	}
}
