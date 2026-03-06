package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type failingListFormataStore struct {
	*MemoryStore
	err error
}

func (s *failingListFormataStore) ListFormataBuilderStreams(_ context.Context) ([]FormataBuilderStream, error) {
	return nil, s.err
}

type failingSaveFormataStore struct {
	*MemoryStore
	err error
}

func (s *failingSaveFormataStore) SaveFormataBuilderStream(_ context.Context, stream FormataBuilderStream) (FormataBuilderStream, error) {
	return FormataBuilderStream{}, s.err
}

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

func TestWorkflowCatalogUsesFormataBuilderStreamsWhenAvailable(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "From file", "string")

	streamPath := filepath.Join(tempDir, "db-seed.yaml")
	writeWorkflowConfig(t, streamPath, "From DB", "string")
	streamData, err := os.ReadFile(streamPath)
	if err != nil {
		t.Fatalf("read stream config: %v", err)
	}

	store := NewMemoryStore()
	saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Key:       "from-db",
		Stream:    string(streamData),
		UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}

	server := &Server{store: store, configDir: tempDir}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(): %v", err)
	}
	if len(catalog) != 1 {
		t.Fatalf("catalog size = %d, want 1", len(catalog))
	}
	cfg, ok := catalog[saved.ID.Hex()]
	if !ok {
		t.Fatalf("expected workflow key %q in catalog", saved.ID.Hex())
	}
	if cfg.Workflow.Name != "From DB" {
		t.Fatalf("workflow name = %q, want %q", cfg.Workflow.Name, "From DB")
	}
}

func TestBootstrapFormataBuilderStreamsSeedsFromConfigWhenEmpty(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	writeWorkflowConfig(t, filepath.Join(tempDir, "secondary.yaml"), "Secondary workflow", "string")

	store := NewMemoryStore()
	err := bootstrapFormataBuilderStreams(t.Context(), store, tempDir, func() time.Time {
		return time.Date(2026, 3, 6, 17, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("bootstrapFormataBuilderStreams: %v", err)
	}

	streams, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil {
		t.Fatalf("ListFormataBuilderStreams: %v", err)
	}
	if len(streams) != 2 {
		t.Fatalf("stream count = %d, want 2", len(streams))
	}

	server := &Server{store: store}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(): %v", err)
	}
	if len(catalog) != 2 {
		t.Fatalf("catalog size = %d, want 2", len(catalog))
	}
}

func TestBootstrapFormataBuilderStreamsNoopWhenAlreadySeeded(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

	store := NewMemoryStore()
	original, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Key:       "existing",
		Stream:    "workflow:\n  name: \"Existing\"\n  steps:\n    - id: \"1\"\n      title: \"Step\"\n      order: 1\n      organization: \"org1\"\n      substeps:\n        - id: \"1.1\"\n          title: \"Input\"\n          order: 1\n          roles: [\"dep1\"]\n          inputKey: \"value\"\n          inputType: \"string\"\norganizations:\n  - slug: \"org1\"\n    name: \"Org\"\nroles:\n  - orgSlug: \"org1\"\n    slug: \"dep1\"\n    name: \"Dep\"\n",
		UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}

	if err := bootstrapFormataBuilderStreams(t.Context(), store, tempDir, time.Now); err != nil {
		t.Fatalf("bootstrapFormataBuilderStreams: %v", err)
	}
	streams, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil {
		t.Fatalf("ListFormataBuilderStreams: %v", err)
	}
	if len(streams) != 1 {
		t.Fatalf("stream count = %d, want 1", len(streams))
	}
	if streams[0].ID != original.ID {
		t.Fatalf("stream ID changed: got %s want %s", streams[0].ID.Hex(), original.ID.Hex())
	}
}

func TestBootstrapFormataBuilderStreamsEdgeCases(t *testing.T) {
	t.Run("nil store", func(t *testing.T) {
		if err := bootstrapFormataBuilderStreams(t.Context(), nil, t.TempDir(), nil); err != nil {
			t.Fatalf("bootstrapFormataBuilderStreams nil store: %v", err)
		}
	})

	t.Run("missing config dir", func(t *testing.T) {
		store := NewMemoryStore()
		err := bootstrapFormataBuilderStreams(t.Context(), store, filepath.Join(t.TempDir(), "missing"), nil)
		if err == nil {
			t.Fatal("expected missing config dir error")
		}
		if !strings.Contains(err.Error(), "config dir not found") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("no yaml files", func(t *testing.T) {
		tempDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tempDir, "note.txt"), []byte("hello"), 0o644); err != nil {
			t.Fatalf("write note.txt: %v", err)
		}
		store := NewMemoryStore()
		err := bootstrapFormataBuilderStreams(t.Context(), store, tempDir, nil)
		if err == nil {
			t.Fatal("expected empty workflow catalog error")
		}
		if !strings.Contains(err.Error(), "workflow config catalog is empty") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("save error", func(t *testing.T) {
		tempDir := t.TempDir()
		writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
		store := &failingSaveFormataStore{
			MemoryStore: NewMemoryStore(),
			err:         errors.New("save failed"),
		}
		err := bootstrapFormataBuilderStreams(t.Context(), store, tempDir, nil)
		if err == nil {
			t.Fatal("expected save error")
		}
		if !strings.Contains(err.Error(), "seed formata stream") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty key from filename", func(t *testing.T) {
		tempDir := t.TempDir()
		content := "workflow:\n  name: \"Workflow\"\n  steps: []\n"
		if err := os.WriteFile(filepath.Join(tempDir, ".yaml"), []byte(content), 0o644); err != nil {
			t.Fatalf("write .yaml: %v", err)
		}
		store := NewMemoryStore()
		err := bootstrapFormataBuilderStreams(t.Context(), store, tempDir, nil)
		if err == nil {
			t.Fatal("expected empty key error")
		}
		if !strings.Contains(err.Error(), "workflow key is empty") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestWorkflowCatalogReturnsListFormataError(t *testing.T) {
	server := &Server{
		store: &failingListFormataStore{
			MemoryStore: NewMemoryStore(),
			err:         errors.New("boom"),
		},
		configDir: t.TempDir(),
	}
	_, err := server.workflowCatalog()
	if err == nil {
		t.Fatal("expected workflowCatalog error")
	}
	if !strings.Contains(err.Error(), "list formata streams") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowCatalogFallsBackToFilesWhenStoreHasNoStreams(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "From file fallback", "string")
	server := &Server{
		store:     NewMemoryStore(),
		configDir: tempDir,
	}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(): %v", err)
	}
	if len(catalog) != 1 {
		t.Fatalf("catalog size = %d, want 1", len(catalog))
	}
	cfg, ok := catalog["workflow"]
	if !ok {
		t.Fatal("expected workflow key from filesystem fallback")
	}
	if cfg.Workflow.Name != "From file fallback" {
		t.Fatalf("workflow name = %q, want %q", cfg.Workflow.Name, "From file fallback")
	}
}

func TestWorkflowCatalogStreamErrorBranches(t *testing.T) {
	t.Run("empty stream id", func(t *testing.T) {
		store := NewMemoryStore()
		store.formataStreams["broken"] = FormataBuilderStream{
			Key:    "broken",
			Stream: "workflow:\n  name: \"Broken\"\n  steps: []\n",
		}
		server := &Server{store: store, configDir: t.TempDir()}
		_, err := server.workflowCatalog()
		if err == nil {
			t.Fatal("expected empty stream id error")
		}
		if !strings.Contains(err.Error(), "formata stream id is empty") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid stream yaml", func(t *testing.T) {
		store := NewMemoryStore()
		store.formataStreams["broken"] = FormataBuilderStream{
			ID:     primitive.NewObjectID(),
			Key:    "broken",
			Stream: "workflow: [",
		}
		server := &Server{store: store, configDir: t.TempDir()}
		_, err := server.workflowCatalog()
		if err == nil {
			t.Fatal("expected invalid stream parse error")
		}
		if !strings.Contains(err.Error(), "parse config stream") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("uses cached db catalog", func(t *testing.T) {
		tempDir := t.TempDir()
		streamPath := filepath.Join(tempDir, "stream.yaml")
		writeWorkflowConfig(t, streamPath, "DB Cached", "string")
		content, err := os.ReadFile(streamPath)
		if err != nil {
			t.Fatalf("read stream: %v", err)
		}
		store := NewMemoryStore()
		if _, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
			Key:       "cached",
			Stream:    string(content),
			UpdatedAt: time.Date(2026, 3, 6, 19, 0, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("SaveFormataBuilderStream: %v", err)
		}
		server := &Server{store: store, configDir: tempDir}
		first, err := server.workflowCatalog()
		if err != nil {
			t.Fatalf("workflowCatalog(first): %v", err)
		}
		second, err := server.workflowCatalog()
		if err != nil {
			t.Fatalf("workflowCatalog(second): %v", err)
		}
		if len(first) != len(second) || len(second) != 1 {
			t.Fatalf("unexpected catalog sizes: first=%d second=%d", len(first), len(second))
		}
	})
}

func TestWorkflowCatalogModTimeFallbacks(t *testing.T) {
	now := time.Date(2026, 3, 6, 18, 0, 0, 0, time.UTC)
	if got := workflowCatalogModTime(FormataBuilderStream{UpdatedAt: now}); !got.Equal(now) {
		t.Fatalf("mod time = %s, want %s", got, now)
	}
	id := primitive.NewObjectID()
	if got := workflowCatalogModTime(FormataBuilderStream{ID: id}); !got.Equal(id.Timestamp()) {
		t.Fatalf("mod time = %s, want id timestamp %s", got, id.Timestamp())
	}
	if got := workflowCatalogModTime(FormataBuilderStream{}); !got.IsZero() {
		t.Fatalf("mod time = %s, want zero", got)
	}
}

func TestParseRuntimeConfigDataErrors(t *testing.T) {
	invalidYAML := []byte("workflow: [")
	if _, err := parseRuntimeConfigData("invalid.yaml", invalidYAML); err == nil {
		t.Fatal("expected yaml parse error")
	}

	emptyWorkflow := []byte("workflow:\n  name: \"\"\n  steps: []\n")
	if _, err := parseRuntimeConfigData("empty.yaml", emptyWorkflow); err == nil {
		t.Fatal("expected empty workflow error")
	}

	invalidInputType := []byte(`
workflow:
  name: "Workflow"
  steps:
    - id: "1"
      title: "Step 1"
      order: 1
      organization: "org1"
      substeps:
        - id: "1.1"
          title: "Input"
          order: 1
          roles: ["dep1"]
          inputKey: "value"
          inputType: "unsupported"
organizations:
  - slug: "org1"
    name: "Organization 1"
roles:
  - orgSlug: "org1"
    slug: "dep1"
    name: "Department 1"
`)
	if _, err := parseRuntimeConfigData("bad-input.yaml", invalidInputType); err == nil {
		t.Fatal("expected invalid input type error")
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

func TestValidateWorkflowRefsMissingOrganization(t *testing.T) {
	cfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{
			{Slug: "org1", Name: "Organization 1"},
		},
		Roles: []WorkflowRole{
			{OrgSlug: "org1", Slug: "dep1", Name: "Department 1"},
		},
		Workflow: WorkflowDef{
			Name: "Workflow",
			Steps: []WorkflowStep{
				{
					StepID:           "1",
					Title:            "Step 1",
					Order:            1,
					OrganizationSlug: "org1",
					Substep: []WorkflowSub{
						{SubstepID: "1.1", Title: "Sub 1", Order: 1, Roles: []string{"dep1"}, InputKey: "value", InputType: "string"},
					},
				},
			},
		},
	}
	server := &Server{store: NewMemoryStore(), enforceAuth: true}

	err := server.validateWorkflowRefs(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected missing organization validation error")
	}
	if !strings.Contains(err.Error(), "missing organization slug org1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkflowRefsMissingRole(t *testing.T) {
	store := NewMemoryStore()
	if _, err := store.CreateOrganization(t.Context(), Organization{Name: "Organization 1"}); err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}

	cfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{
			{Slug: "organization-1", Name: "Organization 1"},
		},
		Roles: []WorkflowRole{
			{OrgSlug: "organization-1", Slug: "dep1", Name: "Department 1"},
		},
		Workflow: WorkflowDef{
			Name: "Workflow",
			Steps: []WorkflowStep{
				{
					StepID:           "1",
					Title:            "Step 1",
					Order:            1,
					OrganizationSlug: "organization-1",
					Substep: []WorkflowSub{
						{SubstepID: "1.1", Title: "Sub 1", Order: 1, Roles: []string{"dep1"}, InputKey: "value", InputType: "string"},
					},
				},
			},
		},
	}
	server := &Server{store: store, enforceAuth: true}

	err := server.validateWorkflowRefs(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected missing role validation error")
	}
	if !strings.Contains(err.Error(), "missing role slug organization-1/dep1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkflowRefsAllowsDuplicateRoleSlugAcrossOrganizations(t *testing.T) {
	store := NewMemoryStore()
	closingOrg, err := store.CreateOrganization(t.Context(), Organization{Name: "Closing the loop"})
	if err != nil {
		t.Fatalf("CreateOrganization(closing): %v", err)
	}
	fivrecOrg, err := store.CreateOrganization(t.Context(), Organization{Name: "Fivrec"})
	if err != nil {
		t.Fatalf("CreateOrganization(fivrec): %v", err)
	}
	if _, err := store.CreateRole(t.Context(), Role{
		OrgID:   closingOrg.ID,
		OrgSlug: closingOrg.Slug,
		Name:    "Operator",
	}); err != nil {
		t.Fatalf("CreateRole(closing/operator): %v", err)
	}
	if _, err := store.CreateRole(t.Context(), Role{
		OrgID:   fivrecOrg.ID,
		OrgSlug: fivrecOrg.Slug,
		Name:    "Operator",
	}); err != nil {
		t.Fatalf("CreateRole(fivrec/operator): %v", err)
	}

	cfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{
			{Slug: "closing-the-loop", Name: "Closing the loop"},
			{Slug: "fivrec", Name: "Fivrec"},
		},
		Roles: []WorkflowRole{
			{OrgSlug: "closing-the-loop", Slug: "operator", Name: "Operator"},
			{OrgSlug: "fivrec", Slug: "operator", Name: "Operator"},
		},
		Workflow: WorkflowDef{
			Name: "Workflow",
			Steps: []WorkflowStep{
				{
					StepID:           "1",
					Title:            "Step 1",
					Order:            1,
					OrganizationSlug: "closing-the-loop",
					Substep: []WorkflowSub{
						{
							SubstepID: "1.1",
							Title:     "Sub 1",
							Order:     1,
							Roles:     []string{"operator"},
							InputKey:  "value",
							InputType: "string",
						},
					},
				},
			},
		},
	}
	server := &Server{store: store, enforceAuth: true}

	if err := server.validateWorkflowRefs(t.Context(), cfg); err != nil {
		t.Fatalf("validateWorkflowRefs returned error: %v", err)
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
		"      organization: \"org1\"\n" +
		"      substeps:\n" +
		"        - id: \"1.1\"\n" +
		"          title: \"Input\"\n" +
		"          order: 1\n" +
		"          roles: [\"dep1\"]\n" +
		"          inputKey: \"value\"\n" +
		"          inputType: \"" + inputType + "\"\n" +
		"organizations:\n" +
		"  - slug: \"org1\"\n" +
		"    name: \"Organization 1\"\n" +
		"roles:\n" +
		"  - orgSlug: \"org1\"\n" +
		"    slug: \"dep1\"\n" +
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
		"      organization: \"org1\"\n" +
		"      substeps:\n" +
		"        - id: \"1.1\"\n" +
		"          title: \"Input\"\n" +
		"          order: 1\n" +
		"          roles: [\"dep1\"]\n" +
		"          inputKey: \"value\"\n" +
		"          inputType: \"string\"\n" +
		"organizations:\n" +
		"  - slug: \"org1\"\n" +
		"    name: \"Organization 1\"\n" +
		"roles:\n" +
		"  - orgSlug: \"org1\"\n" +
		"    slug: \"dep1\"\n" +
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
