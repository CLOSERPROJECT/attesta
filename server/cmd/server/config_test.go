package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		{name: "formata", input: "formata", want: "formata"},
		{name: "schema alias", input: "schema", want: "formata"},
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
					{SubstepID: "1.4", InputType: "formata", Schema: map[string]interface{}{"type": "object"}},
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
	if workflow.Steps[0].Substep[3].InputType != "formata" {
		t.Fatalf("substep 1.4 type = %q, want %q", workflow.Steps[0].Substep[3].InputType, "formata")
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

	missingSchema := WorkflowDef{
		Steps: []WorkflowStep{
			{
				Substep: []WorkflowSub{
					{SubstepID: "3.1", InputType: "formata"},
				},
			},
		},
	}
	err = normalizeInputTypes(&missingSchema)
	if err == nil {
		t.Fatal("normalizeInputTypes(formata without schema): expected error")
	}
	if !strings.Contains(err.Error(), "schema is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowCatalogLoadsMultipleFiles(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string", "Main workflow description")
	writeWorkflowConfig(t, filepath.Join(tempDir, "secondary.yaml"), "Secondary workflow", "number")

	server := &Server{configDir: tempDir}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(): %v", err)
	}
	if len(catalog) != 2 {
		t.Fatalf("catalog size = %d, want 2", len(catalog))
	}
	if catalog["workflow"].Workflow.Name != "Main workflow" {
		t.Fatalf("workflow key mismatch: got %q", catalog["workflow"].Workflow.Name)
	}
	if catalog["workflow"].Workflow.Description != "Main workflow description" {
		t.Fatalf("workflow description mismatch: got %q", catalog["workflow"].Workflow.Description)
	}
	if catalog["secondary"].Workflow.Name != "Secondary workflow" {
		t.Fatalf("secondary key mismatch: got %q", catalog["secondary"].Workflow.Name)
	}
	if catalog["secondary"].Workflow.Description != "" {
		t.Fatalf("secondary workflow description = %q, want empty", catalog["secondary"].Workflow.Description)
	}
}

func TestWorkflowCatalogRejectsInvalidFile(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	writeWorkflowConfig(t, filepath.Join(tempDir, "bad.yaml"), "Bad workflow", "unsupported")

	server := &Server{configDir: tempDir}
	_, err := server.workflowCatalog()
	if err == nil {
		t.Fatal("expected invalid inputType error")
	}
	if !strings.Contains(err.Error(), "bad.yaml") || !strings.Contains(err.Error(), "invalid inputType") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowCatalogRejectsDuplicateWorkflowKeys(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "alpha.yaml"), "Alpha", "string")
	writeWorkflowConfig(t, filepath.Join(tempDir, "alpha.yml"), "Alpha duplicate", "string")

	server := &Server{configDir: tempDir}
	_, err := server.workflowCatalog()
	if err == nil {
		t.Fatal("expected duplicate workflow key error")
	}
	if !strings.Contains(err.Error(), "duplicate workflow key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowCatalogInvalidatesCacheWhenFileChanges(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "workflow.yaml")
	writeWorkflowConfig(t, path, "First", "string")

	server := &Server{configDir: tempDir}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(first): %v", err)
	}
	if catalog["workflow"].Workflow.Name != "First" {
		t.Fatalf("name = %q, want First", catalog["workflow"].Workflow.Name)
	}

	writeWorkflowConfig(t, path, "Second", "string")
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	updated, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(updated): %v", err)
	}
	if updated["workflow"].Workflow.Name != "Second" {
		t.Fatalf("name = %q, want Second", updated["workflow"].Workflow.Name)
	}
}

func TestWorkflowByKeyAndRuntimeConfigUseCatalog(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	writeWorkflowConfig(t, filepath.Join(tempDir, "zeta.yaml"), "Zeta workflow", "string")

	server := &Server{configDir: tempDir}
	cfg, err := server.workflowByKey("zeta")
	if err != nil {
		t.Fatalf("workflowByKey(zeta): %v", err)
	}
	if cfg.Workflow.Name != "Zeta workflow" {
		t.Fatalf("workflowByKey(zeta) name = %q", cfg.Workflow.Name)
	}

	defaultCfg, err := server.runtimeConfig()
	if err != nil {
		t.Fatalf("runtimeConfig(): %v", err)
	}
	if defaultCfg.Workflow.Name != "Main workflow" {
		t.Fatalf("runtimeConfig default = %q, want Main workflow", defaultCfg.Workflow.Name)
	}
}

func TestWorkflowCatalogConfigWithoutDPPRemainsValid(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

	server := &Server{configDir: tempDir}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(): %v", err)
	}
	cfg := catalog["workflow"]
	if cfg.DPP.Enabled {
		t.Fatal("dpp.enabled = true, want false")
	}
	if cfg.DPP.GTIN != "" {
		t.Fatalf("dpp.gtin = %q, want empty", cfg.DPP.GTIN)
	}
}

func TestWorkflowCatalogRejectsEnabledDPPWithoutGTIN(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfigWithDPP(t, filepath.Join(tempDir, "workflow.yaml"), "  enabled: true\n")

	server := &Server{configDir: tempDir}
	_, err := server.workflowCatalog()
	if err == nil {
		t.Fatal("expected dpp.gtin validation error")
	}
	if !strings.Contains(err.Error(), "dpp.gtin is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowCatalogRejectsEnabledDPPWithInvalidGTIN(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfigWithDPP(t, filepath.Join(tempDir, "workflow.yaml"), "  enabled: true\n  gtin: \"abc123\"\n")

	server := &Server{configDir: tempDir}
	_, err := server.workflowCatalog()
	if err == nil {
		t.Fatal("expected dpp.gtin validation error")
	}
	if !strings.Contains(err.Error(), "dpp.gtin must contain only digits") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowCatalogNormalizesEnabledDPPDefaults(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfigWithDPP(t, filepath.Join(tempDir, "workflow.yaml"), "  enabled: true\n  gtin: \"9506000134352\"\n")

	server := &Server{configDir: tempDir}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(): %v", err)
	}
	cfg := catalog["workflow"]
	if cfg.DPP.GTIN != "09506000134352" {
		t.Fatalf("dpp.gtin = %q, want %q", cfg.DPP.GTIN, "09506000134352")
	}
	if cfg.DPP.LotInputKey != "batchId" {
		t.Fatalf("dpp.lotInputKey = %q, want %q", cfg.DPP.LotInputKey, "batchId")
	}
	if cfg.DPP.LotDefault != "defaultProduct" {
		t.Fatalf("dpp.lotDefault = %q, want %q", cfg.DPP.LotDefault, "defaultProduct")
	}
	if cfg.DPP.SerialStrategy != "process_id_hex" {
		t.Fatalf("dpp.serialStrategy = %q, want %q", cfg.DPP.SerialStrategy, "process_id_hex")
	}
}

func writeWorkflowConfig(t *testing.T, path, name, inputType string, description ...string) {
	t.Helper()
	content := "workflow:\n" +
		"  name: \"" + name + "\"\n" +
		func() string {
			if len(description) == 0 || strings.TrimSpace(description[0]) == "" {
				return ""
			}
			return "  description: \"" + description[0] + "\"\n"
		}() +
		"  steps:\n" +
		"    - id: \"1\"\n" +
		"      title: \"Step 1\"\n" +
		"      order: 1\n" +
		"      substeps:\n" +
		"        - id: \"1.1\"\n" +
		"          title: \"Input\"\n" +
		"          order: 1\n" +
		"          role: \"dep1\"\n" +
		"          inputKey: \"value\"\n" +
		"          inputType: \"" + inputType + "\"\n" +
		"departments:\n" +
		"  - id: \"dep1\"\n" +
		"    name: \"Department 1\"\n" +
		"users:\n" +
		"  - id: \"u1\"\n" +
		"    name: \"User 1\"\n" +
		"    departmentId: \"dep1\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config %s: %v", path, err)
	}
}

func writeWorkflowConfigWithDPP(t *testing.T, path, dppBlock string) {
	t.Helper()
	content := "workflow:\n" +
		"  name: \"Workflow\"\n" +
		"  steps:\n" +
		"    - id: \"1\"\n" +
		"      title: \"Step 1\"\n" +
		"      order: 1\n" +
		"      substeps:\n" +
		"        - id: \"1.1\"\n" +
		"          title: \"Input\"\n" +
		"          order: 1\n" +
		"          role: \"dep1\"\n" +
		"          inputKey: \"value\"\n" +
		"          inputType: \"string\"\n" +
		"departments:\n" +
		"  - id: \"dep1\"\n" +
		"    name: \"Department 1\"\n" +
		"users:\n" +
		"  - id: \"u1\"\n" +
		"    name: \"User 1\"\n" +
		"    departmentId: \"dep1\"\n" +
		"dpp:\n" + dppBlock
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config %s: %v", path, err)
	}
}
