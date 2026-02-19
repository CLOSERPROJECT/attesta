package main

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestParseFormataScalarPayloadFallbacks(t *testing.T) {
	sub := WorkflowSub{InputType: "formata", InputKey: "payload"}

	reqWithValue := httptest.NewRequest("POST", "/x", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	reqWithValue.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	payload, err := parseFormataScalarPayload(reqWithValue, sub)
	if err != nil {
		t.Fatalf("parse payload with value: %v", err)
	}
	root, ok := payload["payload"].(map[string]interface{})
	if !ok || root["status"] != "ok" {
		t.Fatalf("unexpected payload map: %#v", payload["payload"])
	}

	reqWithFallback := httptest.NewRequest("POST", "/x", strings.NewReader("status=+ok+&tags=a&tags=+b+"))
	reqWithFallback.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	payload, err = parseFormataScalarPayload(reqWithFallback, sub)
	if err != nil {
		t.Fatalf("parse payload with fallback map: %v", err)
	}
	root, ok = payload["payload"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected payload object, got %#v", payload["payload"])
	}
	if root["status"] != "ok" {
		t.Fatalf("expected trimmed fallback value, got %#v", root["status"])
	}
	tags, ok := root["tags"].([]interface{})
	if !ok || len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Fatalf("expected fallback tags slice, got %#v", root["tags"])
	}

	reqEmpty := httptest.NewRequest("POST", "/x", strings.NewReader(""))
	reqEmpty.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	payload, err = parseFormataScalarPayload(reqEmpty, sub)
	if err != nil {
		t.Fatalf("parse payload with empty form: %v", err)
	}
	root, ok = payload["payload"].(map[string]interface{})
	if !ok || len(root) != 0 {
		t.Fatalf("expected empty object fallback, got %#v", payload["payload"])
	}
}

func TestParseFormataScalarPayloadRejectsInvalidJSON(t *testing.T) {
	sub := WorkflowSub{InputType: "formata", InputKey: "payload"}
	req := httptest.NewRequest("POST", "/x", strings.NewReader("value=%7Bbad"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err := parseFormataScalarPayload(req, sub); err == nil {
		t.Fatal("expected invalid JSON formata payload error")
	}
}

func TestFormMapWithoutValue(t *testing.T) {
	if got := formMapWithoutValue(nil); got != nil {
		t.Fatalf("expected nil for nil input, got %#v", got)
	}

	values := url.Values{
		"value": {"ignored"},
		"  ":    {"ignored"},
		"name":  {"  alice  "},
		"tags":  {"  a ", " b"},
		"empty": {},
	}
	got := formMapWithoutValue(values)
	if got["name"] != "alice" {
		t.Fatalf("expected trimmed single item, got %#v", got["name"])
	}
	items, ok := got["tags"].([]string)
	if !ok || len(items) != 2 || items[0] != "a" || items[1] != "b" {
		t.Fatalf("expected trimmed tags list, got %#v", got["tags"])
	}
	if _, exists := got["value"]; exists {
		t.Fatalf("value key should be excluded: %#v", got)
	}
	if _, exists := got["empty"]; exists {
		t.Fatalf("empty key should be excluded: %#v", got)
	}
}

func TestDecodeDataURL(t *testing.T) {
	if _, ok := decodeDataURL("plain"); ok {
		t.Fatal("plain string should not decode as data URL")
	}
	if _, ok := decodeDataURL("data:text/plain;base64"); ok {
		t.Fatal("missing comma should not decode")
	}
	if _, ok := decodeDataURL("data:text/plain,"); ok {
		t.Fatal("empty payload should not decode")
	}
	decoded, ok := decodeDataURL("data:text/plain;base64,aGVsbG8=")
	if !ok || decoded.ContentType != "text/plain" || string(decoded.Data) != "hello" {
		t.Fatalf("unexpected base64 decode: %#v (ok=%t)", decoded, ok)
	}
	decoded, ok = decodeDataURL("data:,%7Bok%7D")
	if !ok || decoded.ContentType != "application/octet-stream" || string(decoded.Data) != "{ok}" {
		t.Fatalf("unexpected path-unescape decode: %#v (ok=%t)", decoded, ok)
	}
	if _, ok := decodeDataURL("data:text/plain,%ZZ"); ok {
		t.Fatal("invalid escaped payload should fail")
	}
}

func TestCompletedValueHelpers(t *testing.T) {
	if got := truncateDisplayValue(strings.Repeat("x", 201)); !strings.HasSuffix(got, "...") {
		t.Fatalf("expected truncated suffix, got %q", got)
	}
	if !isAttachmentMetaValue(primitive.M{"attachmentId": "abc", "filename": "file.txt"}) {
		t.Fatal("expected primitive.M attachment metadata to be detected")
	}
	if isAttachmentMetaValue(123) {
		t.Fatal("non-map should not be detected as attachment metadata")
	}
	if got := marshalJSONCompact(make(chan int)); got != "" {
		t.Fatalf("expected marshal error to return empty string, got %q", got)
	}
}

func TestBuildActionAttachmentsAndDownloadViews(t *testing.T) {
	processID := primitive.NewObjectID()
	process := &Process{ID: processID}
	attachmentID := primitive.NewObjectID().Hex()
	data := map[string]interface{}{
		"docs": []interface{}{
			map[string]interface{}{"attachmentId": attachmentID, "filename": "../zeta.pdf", "sha256": "hash-z"},
			map[string]interface{}{"attachmentId": attachmentID, "filename": "../zeta.pdf", "sha256": "hash-z"},
			map[string]interface{}{"attachmentId": " ", "filename": "skip.pdf"},
			map[string]interface{}{"attachmentId": primitive.NewObjectID().Hex(), "filename": "alpha.pdf", "sha256": "hash-a"},
		},
	}

	if got := buildActionAttachments("workflow", nil, data); got != nil {
		t.Fatalf("expected nil for nil process, got %#v", got)
	}
	attachments := buildActionAttachments("workflow", process, data)
	if len(attachments) != 2 {
		t.Fatalf("expected 2 deduplicated attachments, got %#v", attachments)
	}
	if attachments[0].Filename != ".._zeta.pdf" {
		t.Fatalf("expected sorted attachments by filename, got %#v", attachments)
	}
	if attachments[1].Filename != "alpha.pdf" {
		t.Fatalf("expected sanitized filename, got %#v", attachments[1])
	}
	if !strings.Contains(attachments[0].URL, "/w/workflow/process/"+processID.Hex()+"/attachment/") {
		t.Fatalf("unexpected attachment url: %q", attachments[0].URL)
	}

	files := []ProcessAttachmentExport{
		{SubstepID: "2.1", AttachmentID: "", Filename: "skip.pdf"},
		{SubstepID: "1.2", AttachmentID: "b", Filename: "../b.pdf"},
		{SubstepID: "1.1", AttachmentID: "a", Filename: "a.pdf"},
	}
	if got := buildProcessDownloadAttachments("workflow", nil, files); got != nil {
		t.Fatalf("expected nil with nil process, got %#v", got)
	}
	if got := buildProcessDownloadAttachments("workflow", process, nil); got != nil {
		t.Fatalf("expected nil with empty files, got %#v", got)
	}
	views := buildProcessDownloadAttachments("workflow", process, files)
	if len(views) != 2 {
		t.Fatalf("expected 2 download views, got %#v", views)
	}
	if views[0].SubstepID != "1.1" || views[0].Filename != "a.pdf" {
		t.Fatalf("unexpected first sorted download view: %#v", views[0])
	}
	if views[1].Filename != ".._b.pdf" {
		t.Fatalf("expected sanitized filename, got %#v", views[1])
	}
}

func TestPersistFormataAttachmentsRecursesAndStoresUploads(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{store: store}
	processID := primitive.NewObjectID()
	now := time.Date(2026, 2, 19, 10, 0, 0, 0, time.UTC)
	substep := WorkflowSub{SubstepID: "3.1", InputKey: "payload", InputType: "formata"}

	raw := primitive.M{
		"docs": []interface{}{
			"data:text/plain;base64,aGVsbG8=",
			"plain-text",
		},
		"nested": map[string]interface{}{
			"count": 3.0,
		},
	}

	converted, err := server.persistFormataAttachments(context.Background(), processID, substep, raw, now, []string{substep.InputKey})
	if err != nil {
		t.Fatalf("persistFormataAttachments error: %v", err)
	}
	root, ok := converted.(map[string]interface{})
	if !ok {
		t.Fatalf("expected converted root map, got %#v", converted)
	}
	docs, ok := root["docs"].([]interface{})
	if !ok || len(docs) != 2 {
		t.Fatalf("expected docs array, got %#v", root["docs"])
	}
	firstDoc, ok := docs[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first doc attachment map, got %#v", docs[0])
	}
	attachmentID, ok := firstDoc["attachmentId"].(string)
	if !ok || attachmentID == "" {
		t.Fatalf("expected attachment metadata in first doc, got %#v", firstDoc)
	}
	if docs[1] != "plain-text" {
		t.Fatalf("expected non-data-url value unchanged, got %#v", docs[1])
	}
	nested, ok := root["nested"].(map[string]interface{})
	if !ok || nested["count"] != 3.0 {
		t.Fatalf("expected nested map preserved, got %#v", root["nested"])
	}

	oid, err := primitive.ObjectIDFromHex(attachmentID)
	if err != nil {
		t.Fatalf("invalid attachment id %q: %v", attachmentID, err)
	}
	download, err := store.OpenAttachmentDownload(context.Background(), oid)
	if err != nil {
		t.Fatalf("OpenAttachmentDownload: %v", err)
	}
	defer download.Close()
}

func TestParseFormataPayloadReturnsAttachmentError(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{store: store}
	processID := primitive.NewObjectID()
	substep := WorkflowSub{SubstepID: "3.1", InputKey: "payload", InputType: "formata"}
	t.Setenv("ATTACHMENT_MAX_BYTES", "1")

	form := url.Values{}
	form.Set("value", `{"file":"data:text/plain;base64,aGVsbG8="}`)
	req := httptest.NewRequest("POST", "/x", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if _, err := server.parseFormataPayload(req, processID, substep, time.Now().UTC()); err == nil {
		t.Fatal("expected attachment persistence error due to size limit")
	}
}

func TestParseFormataPayloadRejectsInvalidFormataJSON(t *testing.T) {
	server := &Server{store: NewMemoryStore()}
	substep := WorkflowSub{SubstepID: "3.1", InputKey: "payload", InputType: "formata"}
	form := url.Values{}
	form.Set("value", "{bad")
	req := httptest.NewRequest("POST", "/x", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err := server.parseFormataPayload(req, primitive.NewObjectID(), substep, time.Now().UTC()); err == nil {
		t.Fatal("expected parseFormataPayload error for invalid JSON")
	}
}

func TestEnsureProcessCompletionArtifactsUpdatesDoneStatus(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID: "1",
				Substep: []WorkflowSub{
					{SubstepID: "1.1", Order: 1, Role: "dep1", InputKey: "value", InputType: "string"},
				},
			},
		},
	}
	cfg := RuntimeConfig{Workflow: def}
	store := NewMemoryStore()
	server := &Server{
		store: store,
		now:   func() time.Time { return time.Date(2026, 2, 19, 10, 0, 0, 0, time.UTC) },
	}

	processID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:     processID,
		Status: "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", Data: map[string]interface{}{"value": "ok"}},
		},
	})

	process, err := store.LoadProcessByID(context.Background(), processID)
	if err != nil {
		t.Fatalf("LoadProcessByID: %v", err)
	}
	process.Progress = normalizeProgressKeys(process.Progress)

	updated := server.ensureProcessCompletionArtifacts(context.Background(), cfg, "workflow", process)
	if updated.Status != "done" {
		t.Fatalf("expected updated status done, got %q", updated.Status)
	}

	stored, err := store.LoadProcessByID(context.Background(), processID)
	if err != nil {
		t.Fatalf("LoadProcessByID (stored): %v", err)
	}
	if stored.Status != "done" {
		t.Fatalf("expected persisted status done, got %q", stored.Status)
	}
}

func TestEnsureProcessCompletionArtifactsNoopAndReloadFallback(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID: "1",
				Substep: []WorkflowSub{
					{SubstepID: "1.1", Order: 1, Role: "dep1", InputKey: "value", InputType: "string"},
				},
			},
		},
	}
	cfg := RuntimeConfig{Workflow: def}
	store := NewMemoryStore()
	server := &Server{
		store: store,
		now:   func() time.Time { return time.Date(2026, 2, 19, 10, 0, 0, 0, time.UTC) },
	}

	if got := server.ensureProcessCompletionArtifacts(context.Background(), cfg, "workflow", nil); got != nil {
		t.Fatalf("expected nil process passthrough, got %#v", got)
	}

	pending := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
		},
	}
	if got := server.ensureProcessCompletionArtifacts(context.Background(), cfg, "workflow", pending); got != pending {
		t.Fatalf("expected pending process passthrough, got %#v", got)
	}

	processID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:     processID,
		Status: "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", Data: map[string]interface{}{"value": "ok"}},
		},
	})
	process, err := store.LoadProcessByID(context.Background(), processID)
	if err != nil {
		t.Fatalf("LoadProcessByID: %v", err)
	}
	process.Progress = normalizeProgressKeys(process.Progress)
	store.LoadProcessErr = context.DeadlineExceeded
	got := server.ensureProcessCompletionArtifacts(context.Background(), cfg, "workflow", process)
	if got.Status != "active" {
		t.Fatalf("expected original process status when reload fails, got %q", got.Status)
	}
	store.LoadProcessErr = nil
	stored, err := store.LoadProcessByID(context.Background(), processID)
	if err != nil {
		t.Fatalf("LoadProcessByID (stored): %v", err)
	}
	if stored.Status != "done" {
		t.Fatalf("expected persisted status done, got %q", stored.Status)
	}
}

func TestMarshalJSONCompactNil(t *testing.T) {
	if got := marshalJSONCompact(nil); got != "" {
		t.Fatalf("expected empty string for nil value, got %q", got)
	}
}

func TestEnsureProcessCompletionArtifactsGeneratesDPP(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID: "1",
				Substep: []WorkflowSub{
					{SubstepID: "1.1", Order: 1, Role: "dep1", InputKey: "value", InputType: "string"},
				},
			},
		},
	}
	cfg := RuntimeConfig{
		Workflow: def,
		DPP: DPPConfig{
			Enabled:        true,
			GTIN:           "09506000134352",
			LotDefault:     "LOT-DEFAULT",
			SerialStrategy: "process_id_hex",
		},
	}
	store := NewMemoryStore()
	server := &Server{
		store: store,
		now:   func() time.Time { return time.Date(2026, 2, 19, 10, 0, 0, 0, time.UTC) },
	}

	processID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:     processID,
		Status: "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", Data: map[string]interface{}{"value": "ok"}},
		},
	})
	process, err := store.LoadProcessByID(context.Background(), processID)
	if err != nil {
		t.Fatalf("LoadProcessByID: %v", err)
	}
	process.Progress = normalizeProgressKeys(process.Progress)

	updated := server.ensureProcessCompletionArtifacts(context.Background(), cfg, "workflow", process)
	if updated.Status != "done" {
		t.Fatalf("expected status done, got %q", updated.Status)
	}
	if updated.DPP == nil {
		t.Fatal("expected DPP to be generated and persisted")
	}
	if updated.DPP.GTIN != "09506000134352" || updated.DPP.Lot != "LOT-DEFAULT" || updated.DPP.Serial != processID.Hex() {
		t.Fatalf("unexpected dpp payload: %#v", updated.DPP)
	}
}
