package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAttachmentMetaFromPayload(t *testing.T) {
	tests := []struct {
		name   string
		data   map[string]interface{}
		input  string
		wantOK bool
		wantID string
		wantSz int64
	}{
		{name: "nil data", data: nil, input: "attachment"},
		{name: "missing key", data: map[string]interface{}{}, input: "attachment"},
		{name: "wrong type", data: map[string]interface{}{"attachment": "bad"}, input: "attachment"},
		{
			name: "valid payload with float size",
			data: map[string]interface{}{
				"attachment": map[string]interface{}{
					"attachmentId": "att-3",
					"filename":     "proof.pdf",
					"sha256":       "def",
					"size":         float64(14),
				},
			},
			input:  "attachment",
			wantOK: true,
			wantID: "att-3",
			wantSz: 14,
		},
		{
			name: "empty fields are ignored",
			data: map[string]interface{}{
				"attachment": map[string]interface{}{},
			},
			input: "attachment",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta := attachmentMetaFromPayload(tc.data, tc.input)
			if (meta != nil) != tc.wantOK {
				t.Fatalf("attachmentMetaFromPayload presence = %t, want %t", meta != nil, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if meta.AttachmentID != tc.wantID || meta.SizeBytes != tc.wantSz {
				t.Fatalf("unexpected metadata: %#v", meta)
			}
		})
	}
}

func TestParseFormataPayloadStoresDataURLAttachment(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{store: store}
	processID := primitive.NewObjectID()
	now := time.Date(2026, 2, 5, 10, 30, 0, 0, time.UTC)
	substep := WorkflowSub{SubstepID: "3.1", Title: "QA Checklist", InputKey: "qaChecklist", InputType: "formata"}

	form := url.Values{}
	form.Set("value", `{"notes":"ready","evidenceFile":"data:text/plain;base64,aGVsbG8="}`)
	req := httptest.NewRequest(http.MethodPost, "/complete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	payload, err := server.parseFormataPayload(req, processID, substep, now)
	if err != nil {
		t.Fatalf("parseFormataPayload returned error: %v", err)
	}

	root := payload
	if root["notes"] != "ready" {
		t.Fatalf("notes = %#v, want %q", root["notes"], "ready")
	}

	fileMeta, ok := root["evidenceFile"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected evidenceFile attachment object, got %#v", root["evidenceFile"])
	}
	attachmentIDRaw, ok := fileMeta["attachmentId"].(string)
	if !ok || attachmentIDRaw == "" {
		t.Fatalf("expected attachmentId in evidenceFile payload, got %#v", fileMeta["attachmentId"])
	}

	attachmentID, err := primitive.ObjectIDFromHex(attachmentIDRaw)
	if err != nil {
		t.Fatalf("attachmentId parse error: %v", err)
	}

	download, err := store.OpenAttachmentDownload(t.Context(), attachmentID)
	if err != nil {
		t.Fatalf("OpenAttachmentDownload: %v", err)
	}
	defer download.Close()

	content, err := io.ReadAll(download)
	if err != nil {
		t.Fatalf("ReadAll attachment content: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("attachment content = %q, want %q", string(content), "hello")
	}
}

func TestParseFormataPayloadFallbacksToPostedFieldsWhenValueMissing(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{store: store}
	processID := primitive.NewObjectID()
	now := time.Date(2026, 2, 5, 10, 30, 0, 0, time.UTC)
	substep := WorkflowSub{SubstepID: "3.1", Title: "QA Checklist", InputKey: "qaChecklist", InputType: "formata"}

	form := url.Values{}
	form.Set("inspector", "alice")
	form.Set("outcome", "accepted")
	req := httptest.NewRequest(http.MethodPost, "/complete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	payload, err := server.parseFormataPayload(req, processID, substep, now)
	if err != nil {
		t.Fatalf("parseFormataPayload returned error: %v", err)
	}
	root := payload
	if root["inspector"] != "alice" {
		t.Fatalf("inspector = %#v, want %q", root["inspector"], "alice")
	}
	if root["outcome"] != "accepted" {
		t.Fatalf("outcome = %#v, want %q", root["outcome"], "accepted")
	}
}

func TestParseFormataScalarPayloadDefaultsToEmptyObject(t *testing.T) {
	substep := WorkflowSub{SubstepID: "3.1", Title: "QA Checklist", InputKey: "qaChecklist", InputType: "formata"}
	req := httptest.NewRequest(http.MethodPost, "/complete", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	payload, err := parseFormataScalarPayload(req, substep)
	if err != nil {
		t.Fatalf("parseFormataScalarPayload error: %v", err)
	}
	root := payload
	if len(root) != 0 {
		t.Fatalf("root = %#v, want empty object", root)
	}
}
