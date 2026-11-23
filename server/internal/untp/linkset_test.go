package untp

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLinksetScopeFilterAndPrefers(t *testing.T) {
	if got, ok := LinksetScope("ALL"); !ok || got != "all" {
		t.Fatalf("all = %q ok=%v", got, ok)
	}
	if got, ok := LinksetScope("linkset"); !ok || got != "all" {
		t.Fatalf("linkset = %q ok=%v", got, ok)
	}
	if got, ok := LinksetScope("dpp"); !ok || got != "dpp" {
		t.Fatalf("dpp = %q ok=%v", got, ok)
	}
	if got, ok := LinksetScope("nope"); ok || got != "" {
		t.Fatalf("unknown = %q ok=%v", got, ok)
	}

	full := BuildLinkset("https://ex.test", "/01/1")
	if _, ok := FilterLinkset(Linkset{}, "dpp"); ok {
		t.Fatal("empty linkset")
	}
	if _, ok := FilterLinkset(full, "nope"); ok {
		t.Fatal("bad relation")
	}
	for _, rel := range []string{"dpp", "dte", "pip"} {
		got, ok := FilterLinkset(full, rel)
		if !ok || len(got.Linkset) != 1 {
			t.Fatalf("%s filter = %#v ok=%v", rel, got, ok)
		}
	}
	emptyRel := Linkset{Linkset: []LinksetContext{{Anchor: "https://ex.test/01/1"}}}
	if _, ok := FilterLinkset(emptyRel, "dpp"); ok {
		t.Fatal("nil targets")
	}

	if PrefersLinkset(nil) {
		t.Fatal("nil request")
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if PrefersLinkset(req) {
		t.Fatal("no accept")
	}
	req.Header.Set("Accept", "application/json, application/linkset+json")
	if !PrefersLinkset(req) {
		t.Fatal("linkset accept")
	}

	if FormatRFC3339UTC(time.Time{}) != "" {
		t.Fatal("zero time")
	}
	if FormatRFC3339UTC(time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)) == "" {
		t.Fatal("formatted time")
	}

	if _, _, _, _, ok, err := ParseAttachmentPath("/not/attachment"); ok || err != nil {
		t.Fatalf("non-attachment = ok=%v err=%v", ok, err)
	}
	if _, _, _, _, ok, err := ParseAttachmentPath("/01/x/10/l/21/s/attachment/%zz/file"); !ok || err == nil {
		t.Fatalf("bad unescape ok=%v err=%v", ok, err)
	}
	if _, _, _, _, ok, err := ParseAttachmentPath("/01/09506000134352/10/l/21/s/attachment/%20/file"); !ok || err == nil {
		t.Fatalf("blank id ok=%v err=%v", ok, err)
	}
	if _, _, _, ok, err := ParseEventsPath("/01/x/10/l/21/s/nope"); ok || err != nil {
		t.Fatalf("non-events = ok=%v err=%v", ok, err)
	}
	if _, _, _, ok, err := ParseEventsPath("/01/%zz/10/l/21/s/events"); !ok || err == nil {
		t.Fatalf("bad events gtin ok=%v err=%v", ok, err)
	}

	if _, err := NormalizeGTIN("123456789012345"); err == nil {
		t.Fatal("gtin too long")
	}
	if _, _, _, _, ok, err := ParseAttachmentPath("/01/09506000134352/10/l/21/s/attachment/%zz/file"); !ok || err == nil {
		t.Fatalf("attachment unescape ok=%v err=%v", ok, err)
	}
	if _, _, _, err := ParsePath("/01/09506000134352/10/%zz/21/s"); err == nil {
		t.Fatal("lot unescape")
	}
	if _, _, _, err := ParsePath("/01/09506000134352/10/l/21/%zz"); err == nil {
		t.Fatal("serial unescape")
	}
	if _, _, _, err := ParsePath("/01/09506000134352/10/%20/21/s"); err == nil {
		t.Fatal("blank lot")
	}
	if _, _, _, err := ParsePath("/01/09506000134352/10/l/21/%20"); err == nil {
		t.Fatal("blank serial")
	}
}
