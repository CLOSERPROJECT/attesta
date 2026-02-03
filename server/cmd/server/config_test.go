package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeInputType(t *testing.T) {
	got, err := normalizeInputType("text")
	if err != nil {
		t.Fatalf("normalizeInputType(text): %v", err)
	}
	if got != "string" {
		t.Fatalf("expected text to normalize to string, got %q", got)
	}

	if _, err := normalizeInputType("unsupported"); err == nil {
		t.Fatal("expected unsupported input type to fail")
	}
}

func TestGetConfigRejectsInvalidInputType(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "workflow.yaml")
	content := `workflow:
  name: "Demo"
  steps:
    - id: "1"
      title: "Step 1"
      order: 1
      substeps:
        - id: "1.1"
          title: "Invalid"
          order: 1
          role: "dep1"
          inputKey: "value"
          inputType: "bad"
departments:
  - id: "dep1"
    name: "Department 1"
users:
  - id: "u1"
    name: "User 1"
    departmentId: "dep1"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	server := &Server{configPath: configPath}
	_, err := server.getConfig()
	if err == nil {
		t.Fatal("expected invalid inputType error")
	}
	if !strings.Contains(err.Error(), "invalid inputType") {
		t.Fatalf("expected invalid inputType in error, got %v", err)
	}
}
