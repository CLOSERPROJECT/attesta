package main

import "testing"

func TestBoolEnvOrAndSessionTTLDays(t *testing.T) {
	t.Run("bool env parsing", func(t *testing.T) {
		t.Setenv("FLAG_BOOL_ENV", "true")
		if !boolEnvOr("FLAG_BOOL_ENV", false) {
			t.Fatal("expected true from boolEnvOr")
		}

		t.Setenv("FLAG_BOOL_ENV", "off")
		if boolEnvOr("FLAG_BOOL_ENV", true) {
			t.Fatal("expected false from boolEnvOr")
		}

		t.Setenv("FLAG_BOOL_ENV", "not-a-bool")
		if !boolEnvOr("FLAG_BOOL_ENV", true) {
			t.Fatal("expected fallback on invalid bool env value")
		}
	})

	t.Run("session ttl fallback for non-positive values", func(t *testing.T) {
		t.Setenv("SESSION_TTL_DAYS", "-5")
		if got := sessionTTLDays(); got != 30 {
			t.Fatalf("sessionTTLDays = %d, want %d", got, 30)
		}
	})
}

func TestAttachmentAndCompletionFormMaxBytes(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("ATTACHMENT_MAX_BYTES", "")
		if got := attachmentMaxBytes(); got != 25*1024*1024 {
			t.Fatalf("attachmentMaxBytes = %d, want default", got)
		}
		if got := completionFormMaxBytes(); got != 25*1024*1024*4+1<<20 {
			t.Fatalf("completionFormMaxBytes = %d, want encoded form allowance", got)
		}
	})

	t.Run("custom value", func(t *testing.T) {
		t.Setenv("ATTACHMENT_MAX_BYTES", "1024")
		if got := attachmentMaxBytes(); got != 1024 {
			t.Fatalf("attachmentMaxBytes = %d, want 1024", got)
		}
		if got := completionFormMaxBytes(); got != 1024*4+1<<20 {
			t.Fatalf("completionFormMaxBytes = %d, want custom encoded form allowance", got)
		}
	})

	t.Run("invalid value falls back", func(t *testing.T) {
		t.Setenv("ATTACHMENT_MAX_BYTES", "not-a-number")
		if got := attachmentMaxBytes(); got != 25*1024*1024 {
			t.Fatalf("attachmentMaxBytes = %d, want default", got)
		}
	})
}
