package living

import (
	"testing"
)

func TestLivingAppBaseURLFromPublicBase(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:3030/")
	if got := LivingAppBaseURL(); got != "http://localhost:3030/app" {
		t.Fatalf("LivingAppBaseURL = %q", got)
	}
}

func TestLivingAppBaseURLTrimsSpace(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", " https://plant.example ")
	if got := LivingAppBaseURL(); got != "https://plant.example/app" {
		t.Fatalf("LivingAppBaseURL = %q", got)
	}
}

func TestLivingAppBaseURLEmpty(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "")
	if got := LivingAppBaseURL(); got != "" {
		t.Fatalf("LivingAppBaseURL = %q", got)
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
