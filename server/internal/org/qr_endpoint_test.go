package org

import (
	"testing"
)

func TestLivingAppBaseURLFromPublicBase(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:3030/")
	if got := OrgAppBaseURL("pv-recovery"); got != "http://localhost:3030/app/pv-recovery" {
		t.Fatalf("OrgAppBaseURL = %q", got)
	}
}

func TestLivingAppBaseURLTrimsSpace(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", " https://plant.example ")
	if got := OrgAppBaseURL("pv-recovery"); got != "https://plant.example/app/pv-recovery" {
		t.Fatalf("OrgAppBaseURL = %q", got)
	}
}

func TestLivingAppBaseURLEmpty(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "")
	if got := OrgAppBaseURL("pv-recovery"); got != "" {
		t.Fatalf("OrgAppBaseURL = %q", got)
	}
	t.Setenv("PUBLIC_BASE_URL", "https://plant.example")
	if got := OrgAppBaseURL(""); got != "" {
		t.Fatalf("OrgAppBaseURL empty stream = %q", got)
	}
}

func TestSubmitBindingsForStreamIncludesQrEndpoint(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "https://plant.example")
	bindings := SubmitBindingsForStream("pv-recovery")
	if bindings["qrEndpoint"] != "https://plant.example/app/pv-recovery" {
		t.Fatalf("SubmitBindingsForStream = %#v", bindings)
	}
}

func TestSubmitBindingsForStreamOmitsWhenUnset(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "")
	if got := SubmitBindingsForStream("pv-recovery"); got != nil {
		t.Fatalf("SubmitBindingsForStream = %#v", got)
	}
}
