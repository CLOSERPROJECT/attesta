package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeInputType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "number", input: "number", want: "number"},
		{name: "string", input: "string", want: "string"},
		{name: "text alias", input: "text", want: "string"},
		{name: "file", input: "file", want: "file"},
		{name: "trim and case", input: "  TeXt  ", want: "string"},
		{name: "unsupported", input: "unsupported", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeInputType(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("normalizeInputType(%q): expected error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeInputType(%q): %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("normalizeInputType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeInputTypes(t *testing.T) {
	workflow := WorkflowDef{
		Steps: []WorkflowStep{
			{
				Substep: []WorkflowSub{
					{SubstepID: "1.1", InputType: " text "},
					{SubstepID: "1.2", InputType: "NUMBER"},
					{SubstepID: "1.3", InputType: "file"},
				},
			},
		},
	}

	if err := normalizeInputTypes(&workflow); err != nil {
		t.Fatalf("normalizeInputTypes(valid): %v", err)
	}
	if workflow.Steps[0].Substep[0].InputType != "string" {
		t.Fatalf("substep 1.1 type = %q, want %q", workflow.Steps[0].Substep[0].InputType, "string")
	}
	if workflow.Steps[0].Substep[1].InputType != "number" {
		t.Fatalf("substep 1.2 type = %q, want %q", workflow.Steps[0].Substep[1].InputType, "number")
	}
	if workflow.Steps[0].Substep[2].InputType != "file" {
		t.Fatalf("substep 1.3 type = %q, want %q", workflow.Steps[0].Substep[2].InputType, "file")
	}

	invalid := WorkflowDef{
		Steps: []WorkflowStep{
			{
				Substep: []WorkflowSub{
					{SubstepID: "2.1", InputType: "unsupported"},
				},
			},
		},
	}
	err := normalizeInputTypes(&invalid)
	if err == nil {
		t.Fatal("normalizeInputTypes(invalid): expected error")
	}
	if !strings.Contains(err.Error(), "invalid inputType for substep 2.1") {
		t.Fatalf("unexpected error: %v", err)
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
