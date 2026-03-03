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
