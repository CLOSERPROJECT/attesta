package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleCompleteSubstepSuccessNonHTMX(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "PROCESS "+processID) {
		t.Fatalf("expected non-HTMX process page marker, got %q", rr.Body.String())
	}
}

func TestHandleCompleteSubstepMissingActorFallsBackToDefault(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	id, _ := primitive.ObjectIDFromHex(processID)
	process, _ := store.SnapshotProcess(id)
	if process.Progress["1_1"].DoneBy == nil || process.Progress["1_1"].DoneBy.ID != "legacy-user" {
		t.Fatalf("expected fallback actor legacy-user|dep1, got %#v", process.Progress["1_1"].DoneBy)
	}
}

func TestHandleCompleteSubstepProcessNotFoundPaths(t *testing.T) {
	missingID := primitive.NewObjectID().Hex()
	server := &Server{
		store:      NewMemoryStore(),
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}

	htmxReq := httptest.NewRequest(http.MethodPost, "/process/"+missingID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	htmxReq.Header.Set("HX-Request", "true")
	htmxReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	htmxRec := httptest.NewRecorder()
	server.handleCompleteSubstep(htmxRec, htmxReq, missingID, "1.1")
	if htmxRec.Code != http.StatusNotFound {
		t.Fatalf("expected HTMX status %d, got %d", http.StatusNotFound, htmxRec.Code)
	}
	if strings.Contains(htmxRec.Body.String(), "PROCESS "+missingID) {
		t.Fatalf("expected HTMX path to render action list only, got %q", htmxRec.Body.String())
	}

	fullReq := httptest.NewRequest(http.MethodPost, "/process/"+missingID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	fullReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	fullRec := httptest.NewRecorder()
	server.handleCompleteSubstep(fullRec, fullReq, missingID, "1.1")
	if fullRec.Code != http.StatusNotFound {
		t.Fatalf("expected non-HTMX status %d, got %d", http.StatusNotFound, fullRec.Code)
	}
	if !strings.Contains(fullRec.Body.String(), "PROCESS ") {
		t.Fatalf("expected non-HTMX path to render process page, got %q", fullRec.Body.String())
	}
}

func TestHandleCompleteSubstepSubstepNotFoundReturns404(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/404/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "404")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandleCompleteSubstepSequenceConflictReturns409(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/2.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u2|dep2"})
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "2.1")

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rr.Code)
	}
}

func TestHandleCompleteSubstepRejectsInvalidFormataJSON(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	invalidJSONReq := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=abc"))
	invalidJSONReq.Header.Set("HX-Request", "true")
	invalidJSONReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidJSONRec := httptest.NewRecorder()
	server.handleCompleteSubstep(invalidJSONRec, invalidJSONReq, processID, "1.1")
	if invalidJSONRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid json status %d, got %d", http.StatusBadRequest, invalidJSONRec.Code)
	}
	if !strings.Contains(invalidJSONRec.Body.String(), "Value must be a valid JSON object.") {
		t.Fatalf("expected parse error message in body, got %q", invalidJSONRec.Body.String())
	}
}

func TestHandleCompleteSubstepReturns404ForWorkflowMismatch(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	id, _ := primitive.ObjectIDFromHex(processID)
	process, ok := store.SnapshotProcess(id)
	if !ok {
		t.Fatalf("process %s not found", processID)
	}
	process.WorkflowKey = "other"
	store.SeedProcess(process)

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testFormataRuntimeConfig(),
	}))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandleCompleteSubstepDeniesInvalidRoleForWorkflowContext(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	id, _ := primitive.ObjectIDFromHex(processID)
	process, ok := store.SnapshotProcess(id)
	if !ok {
		t.Fatalf("process %s not found", processID)
	}
	process.WorkflowKey = "secondary"
	store.SeedProcess(process)

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D&activeRole=dep2"))
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "secondary",
		Cfg: testFormataRuntimeConfig(),
	}))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	updated, ok := store.SnapshotProcess(id)
	if !ok {
		t.Fatalf("process %s not found after completion", processID)
	}
	if updated.Progress["1_1"].State != "pending" {
		t.Fatalf("expected no mutation when workflow cookie mismatches, got state %q", updated.Progress["1_1"].State)
	}
}

func TestHandleCompleteSubstepStoreFailures(t *testing.T) {
	store := NewMemoryStore()
	store.UpdateProgressErr = assertErr("update")
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	updateReq := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	updateReq.Header.Set("HX-Request", "true")
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateRec := httptest.NewRecorder()
	server.handleCompleteSubstep(updateRec, updateReq, processID, "1.1")
	if updateRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected update error status %d, got %d", http.StatusInternalServerError, updateRec.Code)
	}

	store.UpdateProgressErr = nil
	store.InsertNotarizeErr = assertErr("notarize")
	notarizeReq := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	notarizeReq.Header.Set("HX-Request", "true")
	notarizeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	notarizeRec := httptest.NewRecorder()
	server.handleCompleteSubstep(notarizeRec, notarizeReq, processID, "1.1")
	if notarizeRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected notarization error status %d, got %d", http.StatusInternalServerError, notarizeRec.Code)
	}
}

func TestHandleCompleteSubstepLogsPreciseStoreError(t *testing.T) {
	store := NewMemoryStore()
	store.UpdateProgressErr = assertErr("write failed: duplicate key on progress update")
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	var logs bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	oldPrefix := log.Prefix()
	log.SetOutput(&logs)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
		log.SetPrefix(oldPrefix)
	})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=%7B%22status%22%3A%22ok%22%7D"))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	server.handleCompleteSubstep(rec, req, processID, "1.1")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Failed to update process.") {
		t.Fatalf("expected human readable error message, got %q", rec.Body.String())
	}
	logOutput := logs.String()
	if !strings.Contains(logOutput, "failed to update process "+processID+" substep 1.1") {
		t.Fatalf("expected contextual log message, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "write failed: duplicate key on progress update") {
		t.Fatalf("expected precise store error in logs, got %q", logOutput)
	}
}

func TestHandleCompleteSubstepFinalCompletionMarksProcessDone(t *testing.T) {
	store := NewMemoryStore()
	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done"},
			"1_2": {State: "done"},
			"1_3": {State: "done"},
			"2_1": {State: "done"},
			"2_2": {State: "done"},
			"3_1": {State: "done"},
			"3_2": {State: "pending"},
		},
	}
	store.SeedProcess(process)
	server := &Server{
		store:      store,
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return testFormataRuntimeConfig(), nil
		},
		now: func() time.Time { return time.Date(2026, 2, 2, 15, 0, 0, 0, time.UTC) },
	}

	req := httptest.NewRequest(http.MethodPost, "/process/"+process.ID.Hex()+"/substep/3.2/complete", strings.NewReader("value=%7B%22status%22%3A%22done%22%7D"))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u3|dep3"})
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, process.ID.Hex(), "3.2")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	snapshot, ok := store.SnapshotProcess(process.ID)
	if !ok {
		t.Fatal("expected process snapshot")
	}
	if snapshot.Status != "done" {
		t.Fatalf("expected final process status done, got %q", snapshot.Status)
	}
}

func TestHandleCompleteSubstepFinalCompletionGeneratesDPP(t *testing.T) {
	store := NewMemoryStore()
	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", Data: map[string]interface{}{"value": float64(10)}},
			"1_2": {State: "done", Data: map[string]interface{}{"note": "LOT-2026"}},
			"1_3": {State: "done"},
			"2_1": {State: "done"},
			"2_2": {State: "done"},
			"3_1": {State: "done"},
			"3_2": {State: "pending"},
		},
	}
	store.SeedProcess(process)
	server := &Server{
		store:      store,
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			cfg := testFormataRuntimeConfig()
			cfg.DPP = DPPConfig{
				Enabled:        true,
				GTIN:           "09506000134352",
				LotInputKey:    "note",
				SerialStrategy: "process_id_hex",
			}
			return cfg, nil
		},
		now: func() time.Time { return time.Date(2026, 2, 2, 15, 0, 0, 0, time.UTC) },
	}

	req := httptest.NewRequest(http.MethodPost, "/process/"+process.ID.Hex()+"/substep/3.2/complete", strings.NewReader("value=%7B%22status%22%3A%22done%22%7D"))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u3|dep3"})
	rr := httptest.NewRecorder()
	server.handleCompleteSubstep(rr, req, process.ID.Hex(), "3.2")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	snapshot, ok := store.SnapshotProcess(process.ID)
	if !ok {
		t.Fatal("expected process snapshot")
	}
	if snapshot.DPP == nil {
		t.Fatal("expected process DPP to be generated")
	}
	if snapshot.DPP.GTIN != "09506000134352" {
		t.Fatalf("gtin = %q, want 09506000134352", snapshot.DPP.GTIN)
	}
	if snapshot.DPP.Lot != "LOT-2026" {
		t.Fatalf("lot = %q, want LOT-2026", snapshot.DPP.Lot)
	}
	if snapshot.DPP.Serial != process.ID.Hex() {
		t.Fatalf("serial = %q, want %q", snapshot.DPP.Serial, process.ID.Hex())
	}
}

func TestHandleCompleteSubstepFinalCompletionDPPIdempotent(t *testing.T) {
	store := NewMemoryStore()
	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", Data: map[string]interface{}{"value": float64(10)}},
			"1_2": {State: "done", Data: map[string]interface{}{"note": "LOT-2026"}},
			"1_3": {State: "done"},
			"2_1": {State: "done"},
			"2_2": {State: "done"},
			"3_1": {State: "done"},
			"3_2": {State: "pending"},
		},
	}
	store.SeedProcess(process)
	server := &Server{
		store:      store,
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			cfg := testFormataRuntimeConfig()
			cfg.DPP = DPPConfig{
				Enabled:        true,
				GTIN:           "09506000134352",
				LotInputKey:    "note",
				SerialStrategy: "process_id_hex",
			}
			return cfg, nil
		},
		now: func() time.Time { return time.Date(2026, 2, 2, 15, 0, 0, 0, time.UTC) },
	}

	firstReq := httptest.NewRequest(http.MethodPost, "/process/"+process.ID.Hex()+"/substep/3.2/complete", strings.NewReader("value=%7B%22status%22%3A%22done%22%7D"))
	firstReq.Header.Set("HX-Request", "true")
	firstReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	firstReq.AddCookie(&http.Cookie{Name: "demo_user", Value: "u3|dep3"})
	firstRec := httptest.NewRecorder()
	server.handleCompleteSubstep(firstRec, firstReq, process.ID.Hex(), "3.2")
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first completion status = %d, want %d", firstRec.Code, http.StatusOK)
	}

	before, ok := store.SnapshotProcess(process.ID)
	if !ok || before.DPP == nil {
		t.Fatal("expected process DPP after first completion")
	}
	want := *before.DPP

	secondReq := httptest.NewRequest(http.MethodPost, "/process/"+process.ID.Hex()+"/substep/3.2/complete", strings.NewReader("value=%7B%22status%22%3A%22changed%22%7D"))
	secondReq.Header.Set("HX-Request", "true")
	secondReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	secondReq.AddCookie(&http.Cookie{Name: "demo_user", Value: "u3|dep3"})
	secondRec := httptest.NewRecorder()
	server.handleCompleteSubstep(secondRec, secondReq, process.ID.Hex(), "3.2")
	if secondRec.Code != http.StatusOK {
		t.Fatalf("second completion status = %d, want %d", secondRec.Code, http.StatusOK)
	}

	after, ok := store.SnapshotProcess(process.ID)
	if !ok || after.DPP == nil {
		t.Fatal("expected process DPP after second completion")
	}
	if *after.DPP != want {
		t.Fatalf("expected stable DPP value %#v, got %#v", want, *after.DPP)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
