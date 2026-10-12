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
		{name: "number", input: "number", wantErr: true},
		{name: "string", input: "string", wantErr: true},
		{name: "text alias", input: "text", wantErr: true},
		{name: "file", input: "file", wantErr: true},
		{name: "formata", input: "formata", want: "formata"},
		{name: "schema alias", input: "schema", want: "formata"},
		{name: "trim and case", input: "  FoRmAtA  ", want: "formata"},
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
					{SubstepID: "1.1", InputType: " schema ", Schema: map[string]interface{}{"type": "object"}},
					{SubstepID: "1.2", InputType: "JSONSCHEMA", Schema: map[string]interface{}{"type": "object"}},
					{SubstepID: "1.3", InputType: "formata", Schema: map[string]interface{}{"type": "object"}},
				},
			},
		},
	}

	if err := normalizeInputTypes(&workflow); err != nil {
		t.Fatalf("normalizeInputTypes(valid): %v", err)
	}
	for _, substep := range workflow.Steps[0].Substep {
		if substep.InputType != "formata" {
			t.Fatalf("substep %s type = %q, want formata", substep.SubstepID, substep.InputType)
		}
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

	server, _ := newServerWithPackCatalog(t, tempDir)
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(): %v", err)
	}
	if len(catalog) != 2 {
		t.Fatalf("catalog size = %d, want 2", len(catalog))
	}
	_, mainCfg, ok := catalogKeyByWorkflowName(catalog, "Main workflow")
	if !ok {
		t.Fatal("missing Main workflow")
	}
	if mainCfg.Workflow.Description != "Main workflow description" {
		t.Fatalf("workflow description mismatch: got %q", mainCfg.Workflow.Description)
	}
	_, secondaryCfg, ok := catalogKeyByWorkflowName(catalog, "Secondary workflow")
	if !ok {
		t.Fatal("missing Secondary workflow")
	}
	if secondaryCfg.Workflow.Description != "" {
		t.Fatalf("secondary workflow description = %q, want empty", secondaryCfg.Workflow.Description)
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

func TestEnsureEmptyConfigFoundationSeedsFormataFromConfigWhenEmpty(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	writeWorkflowConfig(t, filepath.Join(tempDir, "secondary.yaml"), "Secondary workflow", "string")

	store := NewMemoryStore()
	err := ensureEmptyConfigFoundation(t.Context(), store, tempDir, func() time.Time {
		return time.Date(2026, 3, 6, 17, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("ensureEmptyConfigFoundation: %v", err)
	}

	streams, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil {
		t.Fatalf("ListFormataBuilderStreams: %v", err)
	}
	if len(streams) != 2 {
		t.Fatalf("stream count = %d, want 2", len(streams))
	}
	for _, stream := range streams {
		if stream.CreatedByUserID != platformAdminStreamUserID() {
			t.Fatalf("createdByUserID = %q, want %q", stream.CreatedByUserID, platformAdminStreamUserID())
		}
		if stream.UpdatedByUserID != platformAdminStreamUserID() {
			t.Fatalf("updatedByUserID = %q, want %q", stream.UpdatedByUserID, platformAdminStreamUserID())
		}
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

func TestEnsureEmptyConfigFoundationFormataNoopWhenAlreadySeeded(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

	store := NewMemoryStore()
	original, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{
		Stream:    "workflow:\n  name: \"Existing\"\n  steps:\n    - id: \"1\"\n      title: \"Step\"\n      order: 1\n      organization: \"org1\"\n      substeps:\n        - id: \"1.1\"\n          title: \"Input\"\n          order: 1\n          roles: [\"dep1\"]\n          inputKey: \"value\"\n          inputType: \"string\"\norganizations:\n  - slug: \"org1\"\n    name: \"Org\"\nroles:\n  - orgSlug: \"org1\"\n    slug: \"dep1\"\n    name: \"Dep\"\n",
		UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}

	if err := ensureEmptyConfigFoundation(t.Context(), store, tempDir, time.Now); err != nil {
		t.Fatalf("ensureEmptyConfigFoundation: %v", err)
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

func TestEnsureEmptyConfigFoundationFormataEdgeCases(t *testing.T) {
	t.Run("nil store", func(t *testing.T) {
		if err := ensureEmptyConfigFoundation(t.Context(), nil, t.TempDir(), nil); err != nil {
			t.Fatalf("ensureEmptyConfigFoundation nil store: %v", err)
		}
	})

	t.Run("missing config dir", func(t *testing.T) {
		store := NewMemoryStore()
		err := ensureEmptyConfigFoundation(t.Context(), store, filepath.Join(t.TempDir(), "missing"), nil)
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
		err := ensureEmptyConfigFoundation(t.Context(), store, tempDir, nil)
		if err == nil {
			t.Fatal("expected empty workflow catalog error")
		}
		if !strings.Contains(err.Error(), "catalog template pack is empty") {
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
		err := ensureEmptyConfigFoundation(t.Context(), store, tempDir, nil)
		if err == nil {
			t.Fatal("expected save error")
		}
		if !strings.Contains(err.Error(), "seed formata stream") {
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

func TestWorkflowCatalogFailsClosedWhenFormataEmpty(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "From file fallback", "string")
	server := &Server{
		store:     NewMemoryStore(),
		configDir: tempDir,
	}
	_, err := server.workflowCatalog()
	if err == nil {
		t.Fatal("expected empty Formata catalog error")
	}
	if !strings.Contains(err.Error(), "formata workflow catalog is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowCatalogFailsClosedWithoutStore(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Disk only", "string")
	server := &Server{configDir: tempDir}
	_, err := server.workflowCatalog()
	if err == nil {
		t.Fatal("expected empty Formata catalog error")
	}
	if !strings.Contains(err.Error(), "formata workflow catalog is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowCatalogStreamErrorBranches(t *testing.T) {
	t.Run("empty stream id", func(t *testing.T) {
		store := NewMemoryStore()
		store.formataStreams[primitive.NewObjectID()] = FormataBuilderStream{
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
		streamID := primitive.NewObjectID()
		store.formataStreams[streamID] = FormataBuilderStream{
			ID:     primitive.NewObjectID(),
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

func TestNormalizeOptionalText(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":          "",
		"  ":        "",
		"None":      "",
		"none":      "",
		" null ":    "",
		"nil":       "",
		"undefined": "",
		"Real copy": "Real copy",
		" None yet": "None yet",
	}
	for in, want := range cases {
		if got := normalizeOptionalText(in); got != want {
			t.Fatalf("normalizeOptionalText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeWorkflowConfigClearsNoneDescription(t *testing.T) {
	t.Parallel()
	cfg := RuntimeConfig{Workflow: WorkflowDef{Description: "None"}}
	normalizeWorkflowConfig(&cfg)
	if cfg.Workflow.Description != "" {
		t.Fatalf("description = %q, want empty", cfg.Workflow.Description)
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

func TestWorkflowCatalogRejectsInvalidStream(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	writeWorkflowConfig(t, filepath.Join(tempDir, "bad.yaml"), "Bad workflow", "unsupported")

	server, _ := newServerWithPackCatalog(t, tempDir)
	_, err := server.workflowCatalog()
	if err == nil {
		t.Fatal("expected invalid inputType error")
	}
	if !strings.Contains(err.Error(), "invalid inputType") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCatalogTemplatePackInstallsBothYamlAndYml(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "alpha.yaml"), "Alpha", "string")
	writeWorkflowConfig(t, filepath.Join(tempDir, "alpha.yml"), "Alpha duplicate", "string")

	store := NewMemoryStore()
	installPackIntoFormata(t, store, tempDir)
	streams, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil {
		t.Fatalf("ListFormataBuilderStreams: %v", err)
	}
	if len(streams) != 2 {
		t.Fatalf("stream count = %d, want 2 (pack installs both; Formata keys are ids)", len(streams))
	}
}

func TestWorkflowCatalogInvalidatesCacheWhenFormataStreamChanges(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "First", "string")

	server, store := newServerWithPackCatalog(t, tempDir)
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(first): %v", err)
	}
	key, cfg, ok := catalogKeyByWorkflowName(catalog, "First")
	if !ok {
		t.Fatal("missing First workflow")
	}
	if cfg.Workflow.Name != "First" {
		t.Fatalf("name = %q, want First", cfg.Workflow.Name)
	}

	id, err := primitive.ObjectIDFromHex(key)
	if err != nil {
		t.Fatalf("ObjectIDFromHex: %v", err)
	}
	secondPath := filepath.Join(tempDir, "second.yaml")
	writeWorkflowConfig(t, secondPath, "Second", "string")
	body, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatalf("read second: %v", err)
	}
	existing, err := store.LoadFormataBuilderStreamByID(t.Context(), id)
	if err != nil {
		t.Fatalf("LoadFormataBuilderStreamByID: %v", err)
	}
	existing.Stream = string(body)
	existing.UpdatedAt = time.Now().UTC().Add(2 * time.Second)
	if _, err := store.UpdateFormataBuilderStream(t.Context(), *existing); err != nil {
		t.Fatalf("UpdateFormataBuilderStream: %v", err)
	}

	updated, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(updated): %v", err)
	}
	if updated[key].Workflow.Name != "Second" {
		t.Fatalf("name = %q, want Second", updated[key].Workflow.Name)
	}
}

func TestWorkflowByKeyAndRuntimeConfigUseCatalog(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")
	writeWorkflowConfig(t, filepath.Join(tempDir, "zeta.yaml"), "Zeta workflow", "string")

	server, _ := newServerWithPackCatalog(t, tempDir)
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog: %v", err)
	}
	zetaKey, _, ok := catalogKeyByWorkflowName(catalog, "Zeta workflow")
	if !ok {
		t.Fatal("missing Zeta workflow")
	}
	cfg, err := server.workflowByKey(zetaKey)
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
	if defaultCfg.Workflow.Name == "" {
		t.Fatal("runtimeConfig returned empty workflow name")
	}
}

func TestWorkflowCatalogConfigWithoutDPPRemainsValid(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

	server, _ := newServerWithPackCatalog(t, tempDir)
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(): %v", err)
	}
	_, cfg, ok := catalogKeyByWorkflowName(catalog, "Main workflow")
	if !ok {
		t.Fatal("missing Main workflow")
	}
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

	server, _ := newServerWithPackCatalog(t, tempDir)
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
						{SubstepID: "1.1", Title: "Sub 1", Order: 1, Roles: []string{"dep1"}, InputKey: "value", InputType: "formata"},
					},
				},
			},
		},
	}
	server := &Server{
		store:       NewMemoryStore(),
		identity:    &fakeIdentityStore{listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) { return nil, nil }},
		enforceAuth: true,
	}

	err := server.validateWorkflowRefs(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected missing organization validation error")
	}
	if !strings.Contains(err.Error(), "missing organization slug org1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkflowRefsMissingRole(t *testing.T) {
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
						{SubstepID: "1.1", Title: "Sub 1", Order: 1, Roles: []string{"dep1"}, InputKey: "value", InputType: "formata"},
					},
				},
			},
		},
	}
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
			return []IdentityOrg{{ID: "team-1", Slug: "organization-1", Name: "Organization 1"}}, nil
		}},
		enforceAuth: true,
	}

	err := server.validateWorkflowRefs(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected missing role validation error")
	}
	if !strings.Contains(err.Error(), "missing role slug organization-1/dep1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWorkflowRefsAllowsDuplicateRoleSlugAcrossOrganizations(t *testing.T) {
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
							InputType: "formata",
						},
					},
				},
			},
		},
	}
	server := &Server{
		store: NewMemoryStore(),
		identity: &fakeIdentityStore{listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
			return []IdentityOrg{
				{ID: "team-1", Slug: "closing-the-loop", Name: "Closing the loop", Roles: []IdentityRole{{Slug: "operator", Name: "Operator"}}},
				{ID: "team-2", Slug: "fivrec", Name: "Fivrec", Roles: []IdentityRole{{Slug: "operator", Name: "Operator"}}},
			}, nil
		}},
		enforceAuth: true,
	}

	if err := server.validateWorkflowRefs(t.Context(), cfg); err != nil {
		t.Fatalf("validateWorkflowRefs returned error: %v", err)
	}
}

func TestValidateWorkflowRefsShortCircuitsWithoutIdentityChecks(t *testing.T) {
	cfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{{Slug: "org1", Name: "Organization 1"}},
		Roles:         []WorkflowRole{{OrgSlug: "org1", Slug: "dep1", Name: "Department 1"}},
	}

	if err := (&Server{}).validateWorkflowRefs(t.Context(), cfg); err != nil {
		t.Fatalf("validateWorkflowRefs without identity returned error: %v", err)
	}

	server := &Server{identity: &fakeIdentityStore{}, enforceAuth: false}
	if err := server.validateWorkflowRefs(t.Context(), cfg); err != nil {
		t.Fatalf("validateWorkflowRefs without auth enforcement returned error: %v", err)
	}

	server = &Server{identity: &fakeIdentityStore{}, enforceAuth: true}
	if err := server.validateWorkflowRefs(t.Context(), RuntimeConfig{}); err != nil {
		t.Fatalf("validateWorkflowRefs empty config returned error: %v", err)
	}
}

func TestValidateWorkflowRefsListOrganizationsError(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
			return nil, errors.New("boom")
		}},
		enforceAuth: true,
	}
	cfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{{Slug: "org1", Name: "Organization 1"}},
	}

	err := server.validateWorkflowRefs(t.Context(), cfg)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("validateWorkflowRefs error = %v, want boom", err)
	}
}

func TestValidateWorkflowRefsReportsStepAndRoleMismatches(t *testing.T) {
	cfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{
			{Slug: "org1", Name: "Organization 1"},
			{Slug: "org2", Name: "Organization 2"},
		},
		Roles: []WorkflowRole{
			{OrgSlug: "org1", Slug: "dep1", Name: "Department 1"},
			{OrgSlug: "org2", Slug: "reviewer", Name: "Reviewer"},
		},
		Workflow: WorkflowDef{
			Name: "Workflow",
			Steps: []WorkflowStep{
				{
					StepID:           "1",
					Title:            "Step 1",
					Order:            1,
					OrganizationSlug: "missing-org",
					Substep: []WorkflowSub{
						{SubstepID: "1.1", Title: "Sub 1", Order: 1, InputKey: "value", InputType: "formata"},
					},
				},
				{
					StepID:           "2",
					Title:            "Step 2",
					Order:            2,
					OrganizationSlug: "org1",
					Substep: []WorkflowSub{
						{SubstepID: "2.1", Title: "Sub 2", Order: 1, Roles: []string{"reviewer"}, InputKey: "value", InputType: "formata"},
						{SubstepID: "2.2", Title: "Sub 3", Order: 2, Roles: []string{"missing-role"}, InputKey: "value", InputType: "formata"},
					},
				},
			},
		},
	}
	server := &Server{
		identity: &fakeIdentityStore{listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
			return []IdentityOrg{
				{ID: "team-1", Slug: "org1", Name: "Organization 1", Roles: []IdentityRole{{Slug: "dep1", Name: "Department 1"}}},
				{ID: "team-2", Slug: "org2", Name: "Organization 2", Roles: []IdentityRole{{Slug: "reviewer", Name: "Reviewer"}}},
			}, nil
		}},
		enforceAuth: true,
	}

	err := server.validateWorkflowRefs(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected workflow ref validation error")
	}
	message := err.Error()
	for _, want := range []string{
		"step 1 references organization not in yaml: missing-org",
		"substep 1.1 has no roles",
		"substep 2.1 role reviewer not in step organization org1",
		"substep 2.2 references role not in yaml: missing-role",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("validateWorkflowRefs error %q missing %q", message, want)
		}
	}
}

func TestWorkflowCatalogRejectsEnabledDPPWithInvalidGTIN(t *testing.T) {
	tempDir := t.TempDir()
	writeWorkflowConfigWithDPP(t, filepath.Join(tempDir, "workflow.yaml"), "  enabled: true\n  gtin: \"abc123\"\n")

	server, _ := newServerWithPackCatalog(t, tempDir)
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

	server, _ := newServerWithPackCatalog(t, tempDir)
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog(): %v", err)
	}
	var cfg RuntimeConfig
	for _, item := range catalog {
		cfg = item
		break
	}
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

// installPackIntoFormata copies Stream YAML from configDir into Formata when
// Formata is empty. Use this instead of relying on a runtime disk catalog.
func installPackIntoFormata(t *testing.T, store Store, configDir string) {
	t.Helper()
	if err := ensureEmptyFormataBuilderStreams(t.Context(), store, configDir, time.Now); err != nil {
		t.Fatalf("installPackIntoFormata: %v", err)
	}
}

func newServerWithPackCatalog(t *testing.T, configDir string) (*Server, *MemoryStore) {
	t.Helper()
	store := NewMemoryStore()
	installPackIntoFormata(t, store, configDir)
	return &Server{store: store, configDir: configDir}, store
}

func catalogKeyByWorkflowName(catalog map[string]RuntimeConfig, name string) (string, RuntimeConfig, bool) {
	for key, cfg := range catalog {
		if cfg.Workflow.Name == name {
			return key, cfg, true
		}
	}
	return "", RuntimeConfig{}, false
}

// mustPackCatalogKey installs the pack into Formata and returns the runtime
// catalog key for the named Stream.
func mustPackCatalogKey(t *testing.T, store Store, configDir, workflowName string) string {
	t.Helper()
	installPackIntoFormata(t, store, configDir)
	catalog, err := loadWorkflowCatalog(t.Context(), store)
	if err != nil {
		t.Fatalf("loadWorkflowCatalog: %v", err)
	}
	key, _, ok := catalogKeyByWorkflowName(catalog, workflowName)
	if !ok {
		t.Fatalf("catalog missing workflow %q", workflowName)
	}
	return key
}

func writeWorkflowConfig(t *testing.T, path, name, inputType string, description ...string) {
	t.Helper()
	writeWorkflowConfigWithCategories(t, path, name, inputType, "", "", description...)
}

const (
	publicHomeTestCategorySlug    = "supply-chain"
	publicHomeTestSubCategorySlug = "procurement"
)

func writePublicHomeWorkflowConfig(t *testing.T, path, name, inputType string, description ...string) {
	t.Helper()
	writeWorkflowConfigWithCategories(t, path, name, inputType, publicHomeTestCategorySlug, publicHomeTestSubCategorySlug, description...)
}

func writeWorkflowConfigWithCategories(t *testing.T, path, name, inputType, categorySlug, subCategorySlug string, description ...string) {
	t.Helper()
	normalizedInputType := strings.TrimSpace(inputType)
	switch strings.ToLower(normalizedInputType) {
	case "string", "text", "number", "file":
		normalizedInputType = "formata"
	}
	schema := ""
	if strings.EqualFold(normalizedInputType, "formata") {
		schema = "          schema:\n" +
			"            type: object\n"
	}
	categoryLines := ""
	if strings.TrimSpace(categorySlug) != "" && strings.TrimSpace(subCategorySlug) != "" {
		categoryLines = "  categorySlug: \"" + categorySlug + "\"\n" +
			"  subCategorySlug: \"" + subCategorySlug + "\"\n"
	}
	content := "workflow:\n" +
		"  name: \"" + name + "\"\n" +
		categoryLines +
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
		"          inputType: \"" + normalizedInputType + "\"\n" +
		schema +
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

func writePublicHomeWorkflowConfigWithDPP(t *testing.T, path, dppBlock string) {
	t.Helper()
	content := "workflow:\n" +
		"  name: \"Workflow\"\n" +
		"  categorySlug: \"" + publicHomeTestCategorySlug + "\"\n" +
		"  subCategorySlug: \"" + publicHomeTestSubCategorySlug + "\"\n" +
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
		"          inputType: \"formata\"\n" +
		"          schema:\n" +
		"            type: object\n" +
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
		"          inputType: \"formata\"\n" +
		"          schema:\n" +
		"            type: object\n" +
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
