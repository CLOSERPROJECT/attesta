package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"mime/multipart"
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

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=10"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "PROCESS_PAGE") {
		t.Fatalf("expected non-HTMX process page marker, got %q", rr.Body.String())
	}
}

func TestHandleCompleteSubstepMissingActorFallsBackToDefault(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=10"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "malformed"})
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	id, _ := primitive.ObjectIDFromHex(processID)
	process, _ := store.SnapshotProcess(id)
	if process.Progress["1_1"].DoneBy == nil || process.Progress["1_1"].DoneBy.UserID != "u1" {
		t.Fatalf("expected fallback actor u1|dep1, got %#v", process.Progress["1_1"].DoneBy)
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

	htmxReq := httptest.NewRequest(http.MethodPost, "/process/"+missingID+"/substep/1.1/complete", strings.NewReader("value=10"))
	htmxReq.Header.Set("HX-Request", "true")
	htmxReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	htmxRec := httptest.NewRecorder()
	server.handleCompleteSubstep(htmxRec, htmxReq, missingID, "1.1")
	if htmxRec.Code != http.StatusNotFound {
		t.Fatalf("expected HTMX status %d, got %d", http.StatusNotFound, htmxRec.Code)
	}
	if strings.Contains(htmxRec.Body.String(), "PROCESS_PAGE") {
		t.Fatalf("expected HTMX path to render action list only, got %q", htmxRec.Body.String())
	}

	fullReq := httptest.NewRequest(http.MethodPost, "/process/"+missingID+"/substep/1.1/complete", strings.NewReader("value=10"))
	fullReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	fullRec := httptest.NewRecorder()
	server.handleCompleteSubstep(fullRec, fullReq, missingID, "1.1")
	if fullRec.Code != http.StatusNotFound {
		t.Fatalf("expected non-HTMX status %d, got %d", http.StatusNotFound, fullRec.Code)
	}
	if !strings.Contains(fullRec.Body.String(), "PROCESS_PAGE") {
		t.Fatalf("expected non-HTMX path to render process page, got %q", fullRec.Body.String())
	}
}

func TestHandleCompleteSubstepSubstepNotFoundReturns404(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/404/complete", strings.NewReader("value=10"))
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

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/2.1/complete", strings.NewReader("value=10"))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u2|dep2"})
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "2.1")

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rr.Code)
	}
}

func TestHandleCompleteSubstepFormAndValueValidation(t *testing.T) {
	store := NewMemoryStore()
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	invalidFormReq := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("%zz"))
	invalidFormReq.Header.Set("HX-Request", "true")
	invalidFormReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidFormRec := httptest.NewRecorder()
	server.handleCompleteSubstep(invalidFormRec, invalidFormReq, processID, "1.1")
	if invalidFormRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid form status %d, got %d", http.StatusBadRequest, invalidFormRec.Code)
	}

	missingValueReq := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value="))
	missingValueReq.Header.Set("HX-Request", "true")
	missingValueReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	missingValueRec := httptest.NewRecorder()
	server.handleCompleteSubstep(missingValueRec, missingValueReq, processID, "1.1")
	if missingValueRec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing value status %d, got %d", http.StatusBadRequest, missingValueRec.Code)
	}

	invalidNumberReq := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=abc"))
	invalidNumberReq.Header.Set("HX-Request", "true")
	invalidNumberReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidNumberRec := httptest.NewRecorder()
	server.handleCompleteSubstep(invalidNumberRec, invalidNumberReq, processID, "1.1")
	if invalidNumberRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid number status %d, got %d", http.StatusBadRequest, invalidNumberRec.Code)
	}
	if !strings.Contains(invalidNumberRec.Body.String(), "Value must be a number.") {
		t.Fatalf("expected parse error message in body, got %q", invalidNumberRec.Body.String())
	}
}

func TestHandleCompleteSubstepFileRequiresUpload(t *testing.T) {
	store := NewMemoryStore()
	server, processID := newServerForFileCompleteTests(store)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", &body)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "File is required.") {
		t.Fatalf("expected missing file message, got %q", rr.Body.String())
	}
}

func TestHandleCompleteSubstepFileInvalidMultipartReturns400(t *testing.T) {
	store := NewMemoryStore()
	server, processID := newServerForFileCompleteTests(store)

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("not-multipart"))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Invalid form.") {
		t.Fatalf("expected invalid form message, got %q", rr.Body.String())
	}
}

func TestHandleCompleteSubstepFileTooLargeReturns413(t *testing.T) {
	t.Setenv("ATTACHMENT_MAX_BYTES", "8")

	store := NewMemoryStore()
	server, processID := newServerForFileCompleteTests(store)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("attachment", "test.txt")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("content-too-large")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", &body)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "File too large.") {
		t.Fatalf("expected size error message, got %q", rr.Body.String())
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

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=10"))
	req = req.WithContext(context.WithValue(req.Context(), workflowContextKey{}, workflowContextValue{
		Key: "workflow",
		Cfg: testRuntimeConfig(),
	}))
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestHandleCompleteSubstepFileUploadStoresMetadataAndDigest(t *testing.T) {
	store := NewMemoryStore()
	server, processID := newServerForFileCompleteTests(store)
	fileContent := []byte("certification-content")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("attachment", "certificate.pdf")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(fileContent); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", &body)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, processID, "1.1")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	processObjectID, _ := primitive.ObjectIDFromHex(processID)
	processSnapshot, ok := store.SnapshotProcess(processObjectID)
	if !ok {
		t.Fatal("expected process snapshot")
	}
	progress := processSnapshot.Progress["1_1"]
	attachmentPayload, ok := readAttachmentPayload(progress.Data, "attachment")
	if !ok {
		t.Fatalf("expected attachment payload, got %#v", progress.Data)
	}
	if attachmentPayload.AttachmentID == "" {
		t.Fatal("expected attachment id in payload")
	}

	sum := sha256.Sum256(fileContent)
	expectedSHA256 := hex.EncodeToString(sum[:])
	if attachmentPayload.SHA256 != expectedSHA256 {
		t.Fatalf("expected sha256 %q, got %q", expectedSHA256, attachmentPayload.SHA256)
	}

	notarizations := store.Notarizations()
	if len(notarizations) != 1 {
		t.Fatalf("expected one notarization, got %d", len(notarizations))
	}
	notaryPayload, ok := readAttachmentPayload(notarizations[0].Payload, "attachment")
	if !ok {
		t.Fatalf("expected attachment payload in notarization, got %#v", notarizations[0].Payload)
	}
	if notaryPayload.SHA256 != expectedSHA256 {
		t.Fatalf("expected notarized sha256 %q, got %q", expectedSHA256, notaryPayload.SHA256)
	}
}

func TestHandleCompleteSubstepFileSubstep13Multipart(t *testing.T) {
	store := NewMemoryStore()
	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done"},
			"1_2": {State: "done"},
			"1_3": {State: "pending"},
			"2_1": {State: "pending"},
			"2_2": {State: "pending"},
			"3_1": {State: "pending"},
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
			return testRuntimeConfig(), nil
		},
		now: func() time.Time { return time.Date(2026, 2, 2, 14, 0, 0, 0, time.UTC) },
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("attachment", "cert.pdf")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("substep-13-file")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/process/"+process.ID.Hex()+"/substep/1.3/complete", &body)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "demo_user", Value: "u1|dep1"})
	rr := httptest.NewRecorder()

	server.handleCompleteSubstep(rr, req, process.ID.Hex(), "1.3")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	snapshot, ok := store.SnapshotProcess(process.ID)
	if !ok {
		t.Fatal("expected process snapshot")
	}
	if snapshot.Progress["1_3"].State != "done" {
		t.Fatalf("expected 1_3 state done, got %q", snapshot.Progress["1_3"].State)
	}
}

func TestHandleCompleteSubstepStoreFailures(t *testing.T) {
	store := NewMemoryStore()
	store.UpdateProgressErr = assertErr("update")
	server, processID, _ := newServerForCompleteTests(t, store, fakeAuthorizer{})

	updateReq := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=10"))
	updateReq.Header.Set("HX-Request", "true")
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateRec := httptest.NewRecorder()
	server.handleCompleteSubstep(updateRec, updateReq, processID, "1.1")
	if updateRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected update error status %d, got %d", http.StatusInternalServerError, updateRec.Code)
	}

	store.UpdateProgressErr = nil
	store.InsertNotarizeErr = assertErr("notarize")
	notarizeReq := httptest.NewRequest(http.MethodPost, "/process/"+processID+"/substep/1.1/complete", strings.NewReader("value=10"))
	notarizeReq.Header.Set("HX-Request", "true")
	notarizeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	notarizeRec := httptest.NewRecorder()
	server.handleCompleteSubstep(notarizeRec, notarizeReq, processID, "1.1")
	if notarizeRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected notarization error status %d, got %d", http.StatusInternalServerError, notarizeRec.Code)
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
			return testRuntimeConfig(), nil
		},
		now: func() time.Time { return time.Date(2026, 2, 2, 15, 0, 0, 0, time.UTC) },
	}

	req := httptest.NewRequest(http.MethodPost, "/process/"+process.ID.Hex()+"/substep/3.2/complete", strings.NewReader("value=done"))
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

type assertErr string

func (e assertErr) Error() string { return string(e) }

func newServerForFileCompleteTests(store *MemoryStore) (*Server, string) {
	process := Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
		},
	}
	store.SeedProcess(process)

	server := &Server{
		store:      store,
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			cfg := testRuntimeConfig()
			cfg.Workflow.Steps = []WorkflowStep{
				{
					StepID: "1",
					Title:  "Step 1",
					Order:  1,
					Substep: []WorkflowSub{
						{SubstepID: "1.1", Title: "Upload", Order: 1, Role: "dep1", InputKey: "attachment", InputType: "file"},
					},
				},
			}
			return cfg, nil
		},
		now: func() time.Time { return time.Date(2026, 2, 2, 14, 0, 0, 0, time.UTC) },
	}
	return server, process.ID.Hex()
}
