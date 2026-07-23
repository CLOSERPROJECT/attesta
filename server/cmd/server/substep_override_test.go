package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleSaveSubstepOverrideCreatesAndUpdatesBeforeCompletion(t *testing.T) {
	store := NewMemoryStore()
	server, processID, fixedNow := newServerForCompleteTests(t, store, fakeAuthorizer{})

	saveSubstepOverrideForTest(t, server, processID, "1.1", `{"type":"object","properties":{"available":{"type":"string"}}}`, `{"available":{"ui:widget":"textarea"}}`, "available data")
	override := loadSubstepOverrideForTest(t, store, processID, "1.1")
	if override.Reason != "available data" {
		t.Fatalf("reason = %q", override.Reason)
	}
	if override.ModifiedBy == "" || override.ModifiedByRole != "dep1" {
		t.Fatalf("audit actor = %q/%q", override.ModifiedBy, override.ModifiedByRole)
	}
	if !override.CreatedAt.Equal(fixedNow) || !override.UpdatedAt.Equal(fixedNow) {
		t.Fatalf("timestamps = %s/%s, want %s", override.CreatedAt, override.UpdatedAt, fixedNow)
	}

	saveSubstepOverrideForTest(t, server, processID, "1.1", `{"type":"object","properties":{"updated":{"type":"number"}}}`, `{}`, "updated reason")
	updated := loadSubstepOverrideForTest(t, store, processID, "1.1")
	if updated.Reason != "updated reason" {
		t.Fatalf("updated reason = %q", updated.Reason)
	}
	if _, ok := updated.Schema["properties"].(map[string]interface{})["updated"]; !ok {
		t.Fatalf("updated schema not persisted: %#v", updated.Schema)
	}
	if !updated.CreatedAt.Equal(fixedNow) {
		t.Fatalf("createdAt changed on update: %s", updated.CreatedAt)
	}
}

func TestHandleSaveSubstepOverrideAcceptsFormataChangeReason(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
	rr := postSubstepOverrideForTest(t, server, processID, "1.1", `{"schema":{"type":"object"},"uiSchema":{},"changeReason":"source field changed"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	override := loadSubstepOverrideForTest(t, store, processID, "1.1")
	if override.Reason != "source field changed" {
		t.Fatalf("reason = %q", override.Reason)
	}
}

func TestHandleSaveSubstepOverrideRejectsUnauthorizedCompletedUnsupportedAndInvalid(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		store := NewMemoryStore()
		server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{
			decide: func(Actor, string, string, WorkflowSub, int, string, bool) (bool, error) {
				return false, nil
			},
		})
		rr := postSubstepOverrideForTest(t, server, processID, "1.1", `{"schema":{"type":"object"},"uiSchema":{},"changeReason":"because"}`)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusForbidden)
		}
	})

	t.Run("completed", func(t *testing.T) {
		store := NewMemoryStore()
		server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
		id, _ := primitive.ObjectIDFromHex(processID)
		process, _ := store.SnapshotProcess(id)
		process.Progress["1_1"] = ProcessStep{State: "done"}
		store.SeedProcess(process)
		rr := postSubstepOverrideForTest(t, server, processID, "1.1", `{"schema":{"type":"object"},"uiSchema":{},"changeReason":"because"}`)
		if rr.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		store := NewMemoryStore()
		server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
		cfg := testFormataRuntimeConfig()
		cfg.Workflow.Steps[0].Substep[0].InputType = "file"
		server.configProvider = func() (RuntimeConfig, error) { return cfg, nil }
		rr := postSubstepOverrideForTest(t, server, processID, "1.1", `{"schema":{"type":"object"},"uiSchema":{},"changeReason":"because"}`)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty reason", func(t *testing.T) {
		store := NewMemoryStore()
		server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
		rr := postSubstepOverrideForTest(t, server, processID, "1.1", `{"schema":{"type":"object"},"uiSchema":{},"changeReason":" "}`)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid schema", func(t *testing.T) {
		store := NewMemoryStore()
		server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
		rr := postSubstepOverrideForTest(t, server, processID, "1.1", `{"schema":[],"uiSchema":{},"changeReason":"because"}`)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		store := NewMemoryStore()
		server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
		rr := postSubstepOverrideForTest(t, server, processID, "1.1", `{`)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid ui schema", func(t *testing.T) {
		store := NewMemoryStore()
		server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
		rr := postSubstepOverrideForTest(t, server, processID, "1.1", `{"schema":{"type":"object"},"uiSchema":[],"changeReason":"because"}`)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})
}

func TestAuthorizeSubstepOverrideRequestBranches(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
	cfg := testFormataRuntimeConfig()
	user := &AccountUser{Email: "u1@example.com", RoleSlugs: []string{"dep1"}}
	req := httptest.NewRequest(http.MethodGet, "/instance/"+processID+"/substep/1.2/override", nil)
	if _, _, _, _, status, _, ok := server.authorizeSubstepOverrideRequest(req, user, "workflow", cfg, processID, "1.2"); ok || status != http.StatusConflict {
		t.Fatalf("locked status = %d ok=%v, want conflict false", status, ok)
	}
	if _, _, _, _, status, _, ok := server.authorizeSubstepOverrideRequest(req, user, "workflow", cfg, primitive.NewObjectID().Hex(), "1.1"); ok || status != http.StatusNotFound {
		t.Fatalf("missing process status = %d ok=%v, want not found false", status, ok)
	}
	server.authorizer = nil
	if _, _, _, _, status, _, ok := server.authorizeSubstepOverrideRequest(req, user, "workflow", cfg, processID, "1.1"); ok || status != http.StatusBadGateway {
		t.Fatalf("nil authorizer status = %d ok=%v, want bad gateway false", status, ok)
	}
	server.authorizer = fakeAuthorizer{}
	server.enforceAuth = true
	user.RoleSlugs = []string{"dep2"}
	if _, _, _, _, status, _, ok := server.authorizeSubstepOverrideRequest(req, user, "workflow", cfg, processID, "1.1"); ok || status != http.StatusForbidden {
		t.Fatalf("role mismatch status = %d ok=%v, want forbidden false", status, ok)
	}
}

func TestSubstepOverrideEffectiveSchemaAndCanonicalWorkflowUnchanged(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
	cfg := testFormataRuntimeConfig()
	cfg.Workflow.Steps[0].Substep[0].Schema = map[string]interface{}{"type": "object", "properties": map[string]interface{}{"canonical": map[string]interface{}{"type": "string"}}}
	server.configProvider = func() (RuntimeConfig, error) { return cfg, nil }

	saveSubstepOverrideForTest(t, server, processID, "1.1", `{"type":"object","properties":{"local":{"type":"string"}}}`, `{}`, "local reason")
	id, _ := primitive.ObjectIDFromHex(processID)
	process, _ := store.SnapshotProcess(id)
	process.Progress = normalizeProgressKeys(process.Progress)
	process.Overrides = normalizeSubstepOverrideKeys(process.Overrides)
	actions := buildSubstepViews(cfg.Workflow, &process, "workflow", Actor{ID: "u1", Role: "dep1", RoleSlugs: []string{"dep1"}}, false, map[roleMetaKey]RoleMeta{}, nil)
	if len(actions) == 0 || !strings.Contains(actions[0].FormSchema, "local") {
		t.Fatalf("effective form schema = %q", actions[0].FormSchema)
	}
	if _, ok := cfg.Workflow.Steps[0].Substep[0].Schema["properties"].(map[string]interface{})["canonical"]; !ok {
		t.Fatalf("canonical schema changed: %#v", cfg.Workflow.Steps[0].Substep[0].Schema)
	}
}

func TestCompletedSubstepBodyViewExposesLocalAdaptationReason(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
	saveSubstepOverrideForTest(t, server, processID, "1.1", `{"type":"object"}`, `{}`, "missing field in source system")
	id, _ := primitive.ObjectIDFromHex(processID)
	process, _ := store.SnapshotProcess(id)
	process.Progress["1_1"] = ProcessStep{State: "done", Data: map[string]interface{}{"value": "ok"}}
	store.SeedProcess(process)
	process, _ = store.SnapshotProcess(id)
	process.Progress = normalizeProgressKeys(process.Progress)
	process.Overrides = normalizeSubstepOverrideKeys(process.Overrides)
	actions := buildSubstepViews(testFormataRuntimeConfig().Workflow, &process, "workflow", Actor{ID: "u1", Role: "dep1", RoleSlugs: []string{"dep1"}}, false, map[roleMetaKey]RoleMeta{}, nil)
	if !actions[0].HasOverride || !strings.Contains(actions[0].Reason, "missing field") {
		t.Fatalf("adaptation reason not exposed: %#v", actions[0])
	}
}

func TestGetSubstepOverrideRendersFormataURLAndTargetData(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
	server.formataArchURL = "https://forms.example.test"
	req := httptest.NewRequest(http.MethodGet, "/instance/"+processID+"/substep/1.1/override", nil)
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
	rr := httptest.NewRecorder()
	server.handleGetSubstepOverride(rr, req, processID, "1.1")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "https://forms.example.test") || !strings.Contains(body, "/instance/"+processID+"/substep/1.1/override") {
		t.Fatalf("editor body missing formata url/save route: %s", body)
	}
}

func TestGetSubstepOverrideAllowsSameOriginBuilderWhenFormataURLMissing(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
	req := httptest.NewRequest(http.MethodGet, "/instance/"+processID+"/substep/1.1/override", nil)
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
	rr := httptest.NewRecorder()
	server.handleGetSubstepOverride(rr, req, processID, "1.1")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if strings.Contains(rr.Body.String(), "FORMATA_ARCH_URL is not configured") {
		t.Fatalf("did not expect configuration error: %s", rr.Body.String())
	}
}

func TestGetSubstepOverrideReturnsRequestErrors(t *testing.T) {
	t.Run("missing process", func(t *testing.T) {
		store := NewMemoryStore()
		server, _, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
		missingID := primitive.NewObjectID().Hex()
		req := httptest.NewRequest(http.MethodGet, "/instance/"+missingID+"/substep/1.1/override", nil)
		req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
		rr := httptest.NewRecorder()
		server.handleGetSubstepOverride(rr, req, missingID, "1.1")
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("unsupported substep", func(t *testing.T) {
		store := NewMemoryStore()
		server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
		cfg := testFormataRuntimeConfig()
		cfg.Workflow.Steps[0].Substep[0].InputType = "plain"
		server.configProvider = func() (RuntimeConfig, error) { return cfg, nil }
		req := httptest.NewRequest(http.MethodGet, "/instance/"+processID+"/substep/1.1/override", nil)
		req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
		rr := httptest.NewRecorder()
		server.handleGetSubstepOverride(rr, req, processID, "1.1")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})
}

func TestSaveSubstepOverrideDefaultsMissingUISchema(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
	rr := postSubstepOverrideForTest(t, server, processID, "1.1", `{"schema":{"type":"object"},"changeReason":"no ui schema"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	override := loadSubstepOverrideForTest(t, store, processID, "1.1")
	if override.UISchema == nil || len(override.UISchema) != 0 {
		t.Fatalf("uiSchema = %#v, want empty object", override.UISchema)
	}
}

func TestTraceabilityAndExportExposeLocalAdaptation(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
	saveSubstepOverrideForTest(t, server, processID, "1.1", `{"type":"object"}`, `{}`, "local source shape")
	id, _ := primitive.ObjectIDFromHex(processID)
	process, _ := store.SnapshotProcess(id)
	process.Progress["1_1"] = ProcessStep{State: "done", Data: map[string]interface{}{"value": "ok"}}
	store.SeedProcess(process)
	process, _ = store.SnapshotProcess(id)
	process.Progress = normalizeProgressKeys(process.Progress)
	process.Overrides = normalizeSubstepOverrideKeys(process.Overrides)

	trace := buildDPPTraceabilityView(testFormataRuntimeConfig().Workflow, &process, "workflow", map[roleMetaKey]RoleMeta{}, nil, nil)
	body := trace[0].Substeps[0].Body
	if body == nil || !body.HasOverride || body.OverrideReason != "local source shape" {
		t.Fatalf("trace adaptation fields = %#v", body)
	}
	if body.DetailMessage != "" {
		t.Fatalf("override trace should not use DetailMessage, got %q", body.DetailMessage)
	}
	export := buildNotarizedExport(testFormataRuntimeConfig().Workflow, &process)
	if export.Steps[0].Substeps[0].LocalAdaptationReason != "local source shape" {
		t.Fatalf("export adaptation reason = %#v", export.Steps[0].Substeps[0])
	}
}

func TestMemoryStoreSubstepOverrideUsesDotSafeKey(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})
	saveSubstepOverrideForTest(t, server, processID, "1.1", `{"type":"object"}`, `{}`, "dot key")
	id, _ := primitive.ObjectIDFromHex(processID)
	process, _ := store.SnapshotProcess(id)
	if _, ok := process.Overrides["1_1"]; !ok {
		t.Fatalf("expected encoded override key, got %#v", process.Overrides)
	}
	override := loadSubstepOverrideForTest(t, store, processID, "1.1")
	if override.SubstepID != "1.1" {
		t.Fatalf("override substep id = %q", override.SubstepID)
	}
}

func TestSubstepOverrideHelpersCloneNestedSchemaValues(t *testing.T) {
	source := map[string]interface{}{
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
		"order": []interface{}{"name"},
	}
	cloned := cloneInterfaceMap(source)
	cloned["properties"].(map[string]interface{})["name"].(map[string]interface{})["type"] = "number"
	cloned["order"].([]interface{})[0] = "other"
	if source["properties"].(map[string]interface{})["name"].(map[string]interface{})["type"] != "string" {
		t.Fatalf("source map was mutated: %#v", source)
	}
	if source["order"].([]interface{})[0] != "name" {
		t.Fatalf("source slice was mutated: %#v", source)
	}
	if _, err := decodeJSONObject(nil); err == nil {
		t.Fatal("expected missing object error")
	}
}

func saveSubstepOverrideForTest(t *testing.T, server *Server, processID, substepID, schema, uiSchema, reason string) {
	t.Helper()
	body := map[string]json.RawMessage{
		"schema":       json.RawMessage(schema),
		"uiSchema":     json.RawMessage(uiSchema),
		"changeReason": json.RawMessage(strconvQuote(reason)),
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal override: %v", err)
	}
	rr := postSubstepOverrideForTest(t, server, processID, substepID, string(data))
	if rr.Code != http.StatusOK {
		t.Fatalf("save override status = %d body = %s", rr.Code, rr.Body.String())
	}
}

func postSubstepOverrideForTest(t *testing.T, server *Server, processID, substepID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/instance/"+processID+"/substep/"+substepID+"/override", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
	rr := httptest.NewRecorder()
	server.handleSaveSubstepOverride(rr, req, processID, substepID)
	return rr
}

func loadSubstepOverrideForTest(t *testing.T, store *MemoryStore, processID, substepID string) SubstepOverride {
	t.Helper()
	id, _ := primitive.ObjectIDFromHex(processID)
	override, err := store.GetSubstepOverride(t.Context(), id, substepID)
	if err != nil {
		t.Fatalf("GetSubstepOverride: %v", err)
	}
	return *override
}

func strconvQuote(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
