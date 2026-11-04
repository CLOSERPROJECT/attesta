package untp

import "testing"

func TestParsePathValidAndInvalid(t *testing.T) {
	gtin, lot, serial, err := ParsePath("/01/09506000134352/10/LOT-001/21/SERIAL-001")
	if err != nil {
		t.Fatalf("ParsePath(valid): %v", err)
	}
	if gtin != "09506000134352" || lot != "LOT-001" || serial != "SERIAL-001" {
		t.Fatalf("unexpected parsed values: gtin=%q lot=%q serial=%q", gtin, lot, serial)
	}

	if _, _, _, err = ParsePath("/01/09506000134352/10/LOT-001"); err == nil {
		t.Fatal("expected invalid path shape error")
	}
	if _, _, _, err = ParsePath("/01/not-digits/10/LOT-001/21/SERIAL-001"); err == nil {
		t.Fatal("expected invalid gtin error")
	}
}

func TestParseAttachmentPath(t *testing.T) {
	gtin, lot, serial, attachmentID, ok, err := ParseAttachmentPath("/01/09506000134352/10/LOT-001/21/SERIAL-001/attachment/file%201/file")
	if err != nil {
		t.Fatalf("ParseAttachmentPath(valid): %v", err)
	}
	if !ok {
		t.Fatal("expected attachment path")
	}
	if gtin != "09506000134352" || lot != "LOT-001" || serial != "SERIAL-001" || attachmentID != "file 1" {
		t.Fatalf("unexpected parsed values: gtin=%q lot=%q serial=%q attachmentID=%q", gtin, lot, serial, attachmentID)
	}

	if _, _, _, _, ok, err = ParseAttachmentPath("/01/09506000134352/10/LOT-001/21/SERIAL-001"); ok || err != nil {
		t.Fatalf("plain path returned ok=%v err=%v, want ok=false err=nil", ok, err)
	}
}

func TestParseEventsPath(t *testing.T) {
	gtin, lot, serial, ok, err := ParseEventsPath("/01/09506000134352/10/LOT-1/21/SER-1/events")
	if !ok || err != nil || gtin != "09506000134352" || lot != "LOT-1" || serial != "SER-1" {
		t.Fatalf("ParseEventsPath = (%q, %q, %q, %v, %v)", gtin, lot, serial, ok, err)
	}
	if _, _, _, ok, err = ParseEventsPath("/01/09506000134352/10/LOT-1/21/SER-1"); ok || err != nil {
		t.Fatalf("expected no match for plain digital link, got ok=%v err=%v", ok, err)
	}
}

func TestURLAndGS1ElementString(t *testing.T) {
	if got := URL("09506000134352", "LOT 001", "SERIAL/001"); got != "/01/09506000134352/10/LOT%20001/21/SERIAL%2F001" {
		t.Fatalf("URL() = %q", got)
	}
	if got := GS1ElementString(" 09506000134352 ", " LOT-42 ", " SERIAL-9 "); got != "(01)09506000134352(10)LOT-42(21)SERIAL-9" {
		t.Fatalf("GS1ElementString() = %q", got)
	}
	if got := GS1ElementString("09506000134352", "", "SERIAL-9"); got != "" {
		t.Fatalf("GS1ElementString(missing lot) = %q, want empty", got)
	}
}

func TestNormalizeGTIN(t *testing.T) {
	got, err := NormalizeGTIN("9506000134352")
	if err != nil {
		t.Fatalf("NormalizeGTIN: %v", err)
	}
	if got != "09506000134352" {
		t.Fatalf("NormalizeGTIN = %q", got)
	}
	if _, err := NormalizeGTIN(""); err == nil {
		t.Fatal("expected required error")
	}
	if _, err := NormalizeGTIN("abc"); err == nil {
		t.Fatal("expected non-digit error")
	}
}

func TestAbsoluteURL(t *testing.T) {
	if got := AbsoluteURL("https://dl.example.com/", "/01/x"); got != "https://dl.example.com/01/x" {
		t.Fatalf("AbsoluteURL = %q", got)
	}
	if got := AbsoluteURL("", "/01/x"); got != "/01/x" {
		t.Fatalf("AbsoluteURL empty base = %q", got)
	}
}
