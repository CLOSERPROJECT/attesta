package main

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func organizationLogoRequest(t *testing.T, filename, contentType string, data []byte) *http.Request {
	t.Helper()
	_ = contentType

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("logo", filename)
	if err != nil {
		t.Fatalf("CreateFormFile error: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("part.Write error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/org-admin/users", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(int64(body.Len())); err != nil {
		t.Fatalf("ParseMultipartForm error: %v", err)
	}
	return req
}

func TestReadOrganizationLogoUpload(t *testing.T) {
	server := &Server{}

	t.Run("accepts png upload", func(t *testing.T) {
		req := organizationLogoRequest(t, "logo.png", "", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
		upload, errMsg := server.readOrganizationLogoUpload(req)
		if errMsg != "" {
			t.Fatalf("errMsg = %q", errMsg)
		}
		if upload == nil || upload.Filename != "logo.png" || upload.ContentType != "image/png" {
			t.Fatalf("upload = %#v", upload)
		}
	})

	t.Run("rejects unsupported file type", func(t *testing.T) {
		req := organizationLogoRequest(t, "logo.txt", "text/plain", []byte("plain text"))
		upload, errMsg := server.readOrganizationLogoUpload(req)
		if upload != nil {
			t.Fatalf("upload = %#v, want nil", upload)
		}
		if errMsg != "logo must be a PNG, JPG, WEBP, or SVG image" {
			t.Fatalf("errMsg = %q", errMsg)
		}
	})
}

func TestParseOrganizationLogoUpload(t *testing.T) {
	now := time.Date(2026, 2, 26, 15, 0, 0, 0, time.UTC)

	t.Run("saves attachment", func(t *testing.T) {
		server := &Server{
			store: NewMemoryStore(),
			now:   func() time.Time { return now },
		}
		req := organizationLogoRequest(t, "logo.png", "", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
		attachmentID, errMsg := server.parseOrganizationLogoUpload(context.Background(), req, "Acme Org")
		if errMsg != "" {
			t.Fatalf("errMsg = %q", errMsg)
		}
		if strings.TrimSpace(attachmentID) == "" {
			t.Fatal("expected attachment id")
		}
	})

	t.Run("rejects oversized upload", func(t *testing.T) {
		t.Setenv("ORG_LOGO_MAX_BYTES", "4")
		server := &Server{
			store: NewMemoryStore(),
			now:   func() time.Time { return now },
		}
		req := organizationLogoRequest(t, "logo.png", "", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
		attachmentID, errMsg := server.parseOrganizationLogoUpload(context.Background(), req, "Acme Org")
		if attachmentID != "" {
			t.Fatalf("attachmentID = %q, want empty", attachmentID)
		}
		if errMsg != "logo file too large" {
			t.Fatalf("errMsg = %q", errMsg)
		}
	})
}
