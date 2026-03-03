package main

import "testing"

func TestCSSValue(t *testing.T) {
	if got := string(cssValue("  var(--accent) ", "var(--fallback)")); got != "var(--accent)" {
		t.Fatalf("cssValue explicit = %q, want %q", got, "var(--accent)")
	}
	if got := string(cssValue("   ", "  var(--fallback)  ")); got != "var(--fallback)" {
		t.Fatalf("cssValue fallback = %q, want %q", got, "var(--fallback)")
	}
}
