package org

import (
	"testing"
)

func TestLivingAppBaseURLFromPublicBase(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:3030/")
	if got := OrgAppBaseURL(); got != "http://localhost:3030/app" {
		t.Fatalf("OrgAppBaseURL = %q", got)
	}
}

func TestLivingAppBaseURLTrimsSpace(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", " https://plant.example ")
	if got := OrgAppBaseURL(); got != "https://plant.example/app" {
		t.Fatalf("OrgAppBaseURL = %q", got)
	}
}

func TestLivingAppBaseURLEmpty(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "")
	if got := OrgAppBaseURL(); got != "" {
		t.Fatalf("OrgAppBaseURL = %q", got)
	}
}

func TestSubmitBindingsFromEnvIncludesQrEndpoint(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "https://plant.example")
	bindings := SubmitBindingsFromEnv()
	if bindings["qrEndpoint"] != "https://plant.example/app" {
		t.Fatalf("SubmitBindingsFromEnv = %#v", bindings)
	}
}

func TestSubmitBindingsFromEnvOmitsWhenUnset(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "")
	if got := SubmitBindingsFromEnv(); got != nil {
		t.Fatalf("SubmitBindingsFromEnv = %#v", got)
	}
}
