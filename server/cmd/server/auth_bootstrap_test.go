package main

import "testing"

func TestBootstrapPlatformAdminNoops(t *testing.T) {
	if err := bootstrapPlatformAdmin(t.Context(), NewMemoryStore(), nil); err != nil {
		t.Fatalf("bootstrapPlatformAdmin error: %v", err)
	}
}
