package main

import (
	"strings"
	"testing"
)

func TestSanitizeAttachmentFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trim spaces", input: "  report.pdf  ", want: "report.pdf"},
		{name: "empty fallback", input: "   ", want: "attachment"},
		{name: "nul fallback", input: "\x00", want: "attachment"},
		{name: "unix traversal", input: "../../etc/passwd", want: ".._.._etc_passwd"},
		{name: "windows traversal", input: "..\\secret.txt", want: ".._secret.txt"},
		{name: "control chars", input: "bad\r\nname.txt", want: "bad__name.txt"},
		{name: "quotes removed", input: "\"file\".txt", want: "file.txt"},
		{name: "unicode kept", input: "résumé.pdf", want: "résumé.pdf"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeAttachmentFilename(tc.input); got != tc.want {
				t.Fatalf("sanitizeAttachmentFilename(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestDetectAttachmentContentType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		prefix  string
	}{
		{name: "known text extension", input: "notes.txt", prefix: "text/plain"},
		{name: "known uppercase extension", input: "REPORT.PDF", want: "application/pdf"},
		{name: "unknown extension", input: "file.unknownext", want: "application/octet-stream"},
		{name: "no extension", input: "README", want: "application/octet-stream"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectAttachmentContentType(tc.input)
			if tc.prefix != "" {
				if !strings.HasPrefix(got, tc.prefix) {
					t.Fatalf("detectAttachmentContentType(%q) = %q, want prefix %q", tc.input, got, tc.prefix)
				}
				return
			}
			if got != tc.want {
				t.Fatalf("detectAttachmentContentType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
