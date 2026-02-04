package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestReadAttachmentPayload(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]interface{}
		input   string
		wantOK  bool
		wantID  string
		wantSz  int64
		wantSHA string
	}{
		{name: "nil data", data: nil, input: "attachment"},
		{name: "missing key", data: map[string]interface{}{}, input: "attachment"},
		{name: "wrong payload type", data: map[string]interface{}{"attachment": "bad"}, input: "attachment"},
		{
			name: "map payload",
			data: map[string]interface{}{
				"attachment": map[string]interface{}{
					"attachmentId": "att-1",
					"filename":     "proof.pdf",
					"contentType":  "application/pdf",
					"size":         float64(11),
					"sha256":       "abc",
				},
			},
			input:   "attachment",
			wantOK:  true,
			wantID:  "att-1",
			wantSz:  11,
			wantSHA: "abc",
		},
		{
			name: "primitive map payload",
			data: map[string]interface{}{
				"attachment": primitive.M{
					"attachmentId": "att-2",
					"filename":     "proof.txt",
					"size":         "12",
				},
			},
			input:  "attachment",
			wantOK: true,
			wantID: "att-2",
			wantSz: 12,
		},
		{
			name: "missing attachment id is invalid",
			data: map[string]interface{}{
				"attachment": map[string]interface{}{"filename": "proof.pdf"},
			},
			input: "attachment",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := readAttachmentPayload(tc.data, tc.input)
			if ok != tc.wantOK {
				t.Fatalf("readAttachmentPayload ok = %t, want %t", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if got.AttachmentID != tc.wantID || got.Size != tc.wantSz || got.SHA256 != tc.wantSHA {
				t.Fatalf("unexpected payload: %#v", got)
			}
		})
	}
}

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

func TestParseFilePayloadDetectsContentTypeWhenMissingHeader(t *testing.T) {
	server := &Server{store: NewMemoryStore()}
	processID := primitive.NewObjectID()
	substep := WorkflowSub{SubstepID: "1.3", InputKey: "attachment", InputType: "file"}
	now := time.Date(2026, 2, 4, 13, 0, 0, 0, time.UTC)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="attachment"; filename="proof.txt"`)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write([]byte("hello")); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/complete", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	payload, err := server.parseFilePayload(rec, req, processID, substep, now)
	if err != nil {
		t.Fatalf("parseFilePayload returned error: %v", err)
	}
	attachment, ok := readAttachmentPayload(payload, "attachment")
	if !ok {
		t.Fatalf("expected attachment payload, got %#v", payload)
	}
	if !strings.HasPrefix(attachment.ContentType, "text/plain") {
		t.Fatalf("content-type = %q, want text/plain prefix", attachment.ContentType)
	}
	if attachment.Size != 5 {
		t.Fatalf("size = %d, want 5", attachment.Size)
	}
}

func TestParseFilePayloadRequiresSingleFile(t *testing.T) {
	server := &Server{store: NewMemoryStore()}
	processID := primitive.NewObjectID()
	substep := WorkflowSub{SubstepID: "1.3", InputKey: "attachment", InputType: "file"}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/complete", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	_, err := server.parseFilePayload(rec, req, processID, substep, time.Now().UTC())
	if err != errFileRequired {
		t.Fatalf("parseFilePayload error = %v, want errFileRequired", err)
	}
}
