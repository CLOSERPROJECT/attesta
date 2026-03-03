package main

import "testing"

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
