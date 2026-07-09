package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestResolveRolePalette(t *testing.T) {
	t.Run("palette set", func(t *testing.T) {
		got := resolveRolePalette(IdentityRole{
			Slug:    "chemist",
			Name:    "Chemist",
			Palette: "emerald",
		})
		if got != "emerald" {
			t.Fatalf("palette set = %q, want emerald", got)
		}
	})

	t.Run("legacy color only", func(t *testing.T) {
		got := resolveRolePalette(IdentityRole{
			Slug:  "chemist",
			Name:  "Chemist",
			Color: "var(--role-blue-bg)",
		})
		if got != "blue" {
			t.Fatalf("legacy color = %q, want blue", got)
		}
	})

	t.Run("unknown values", func(t *testing.T) {
		got := resolveRolePalette(IdentityRole{
			Slug: "unknown",
			Name: "Unknown Role",
		})
		if got != "fallback" {
			t.Fatalf("unknown values = %q, want fallback", got)
		}
	})

	t.Run("invalid palette falls back to legacy", func(t *testing.T) {
		got := resolveRolePalette(IdentityRole{
			Slug:    "chemist",
			Name:    "Chemist",
			Palette: "not-a-real-palette",
			Color:   "var(--role-blue-bg)",
		})
		if got != "blue" {
			t.Fatalf("invalid palette with legacy = %q, want blue", got)
		}
	})
}

func TestDefaultRolePaletteFromInput(t *testing.T) {
	if got := defaultRolePaletteFromInput(""); got != "red" {
		t.Fatalf("empty input palette = %q, want %q", got, "red")
	}

	paletteA := defaultRolePaletteFromInput("  Chief   Quality Officer ")
	paletteB := defaultRolePaletteFromInput("chief quality officer")
	if paletteA != paletteB {
		t.Fatalf("expected normalized input to produce same palette, got %q vs %q", paletteA, paletteB)
	}

	paletteSet := map[string]bool{}
	for _, palette := range rolePaletteKeys {
		paletteSet[palette] = true
	}
	got := defaultRolePaletteFromInput("line leader")
	if !paletteSet[got] {
		t.Fatalf("derived palette %q is not part of configured palettes", got)
	}
}

func TestRolePaletteKeysMatchCSS(t *testing.T) {
	cssPath := filepath.Join("..", "..", "..", "web", "src", "styles", "role-palette.css")
	raw, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("read role palette css: %v", err)
	}

	re := regexp.MustCompile(`\[data-role-palette="([^"]+)"\]`)
	matches := re.FindAllStringSubmatch(string(raw), -1)
	if len(matches) == 0 {
		t.Fatal("no data-role-palette selectors found in role-palette.css")
	}

	cssKeys := make(map[string]bool, len(matches))
	for _, match := range matches {
		cssKeys[match[1]] = true
	}

	goSet := make(map[string]bool, len(rolePaletteKeys))
	for _, key := range rolePaletteKeys {
		goSet[key] = true
		if !cssKeys[key] {
			t.Errorf("rolePaletteKeys contains %q but role-palette.css does not", key)
		}
	}

	for key := range cssKeys {
		if !goSet[key] {
			t.Errorf("role-palette.css contains %q but rolePaletteKeys does not", key)
		}
	}
}
