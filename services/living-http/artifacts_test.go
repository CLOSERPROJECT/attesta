package main

import (
	"bytes"
	"strings"
	"testing"
)

func sampleLabelBody(withQR bool) map[string]any {
	labels := []any{
		map[string]any{"id": "NO-261012-1.1", "client": "NO"},
		map[string]any{"id": "NO-261012-1.2", "client": "NO"},
	}
	if withQR {
		labels[0].(map[string]any)["qr"] = "https://plant.example/app?entityId=NO-261012-1.1"
		labels[1].(map[string]any)["qr"] = "https://plant.example/app?entityId=NO-261012-1.2"
	}
	return map[string]any{
		"lotId":            "NO-261012-1",
		"date":              "261012",
		"client":            "NO",
		"clientProgressive": 1.0,
		"numeroFir":         "FIR-99",
		"wasteCode":         "160214",
		"palletCount":       2.0,
		"netWeight":         300.0,
		"labels":            labels,
	}
}

func TestLabelsPDFIncludesPlantFields(t *testing.T) {
	pdf, err := labelsPDF(sampleLabelBody(false))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("not a pdf: %q", pdf[:min(20, len(pdf))])
	}
	for _, want := range []string{
		"261012 NO1.1",
		"FIR: FIR-99",
		"EER 16.02.14",
		"2 bancali / peso totale 300",
	} {
		if !bytes.Contains(pdf, []byte(want)) {
			t.Fatalf("pdf missing %q (len=%d)", want, len(pdf))
		}
	}
}

func TestLabelsPDFEncodesQrAsPNGNotJPEG(t *testing.T) {
	payload := "https://plant.example/app?entityId=NO-261012-1.1"
	pdf, err := labelsPDF(map[string]any{
		"lotId": "NO-261012-1",
		"date":   "261012",
		"client": "NO",
		"clientProgressive": "1",
		"labels": []any{map[string]any{
			"id": "NO-261012-1.1",
			"qr": payload,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(pdf, []byte("JFIF")) || bytes.Contains(pdf, []byte("/DCTDecode")) {
		t.Fatalf("QR must not use JPEG/DCT (len=%d)", len(pdf))
	}
	if !bytes.Contains(pdf, []byte("/FlateDecode")) {
		t.Fatalf("expected FlateDecode image stream for PNG QR (len=%d)", len(pdf))
	}
	if bytes.Count(pdf, []byte("/Subtype /Image")) < 2 {
		// template background + QR
		t.Fatalf("expected template+QR images, got %d", bytes.Count(pdf, []byte("/Subtype /Image")))
	}
	plain, err := labelsPDF(map[string]any{
		"lotId": "NO-261012-1",
		"labels": []any{map[string]any{"id": "NO-261012-1.1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) <= len(plain) {
		t.Fatalf("qr pdf len=%d should exceed plain len=%d", len(pdf), len(plain))
	}
}

func TestLabelsPDFOnePagePerPallet(t *testing.T) {
	pdf, err := labelsPDF(sampleLabelBody(true))
	if err != nil {
		t.Fatal(err)
	}
	pageObjs := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
	if pageObjs != 2 {
		t.Fatalf("page objects = %d, want 2 (len=%d)", pageObjs, len(pdf))
	}
	if !bytes.Contains(pdf, []byte("/Count 2")) {
		t.Fatalf("pdf missing /Count 2 (len=%d)", len(pdf))
	}
	// template + one QR per page → at least 3 image XObjects (shared template may dedupe)
	if n := bytes.Count(pdf, []byte("/Subtype /Image")); n < 2 {
		t.Fatalf("expected image XObjects for template/QR, got %d (len=%d)", n, len(pdf))
	}
	if bytes.Contains(pdf, []byte("/DCTDecode")) || bytes.Contains(pdf, []byte("JFIF")) {
		t.Fatalf("unexpected JPEG QR encoding")
	}
}

func TestLabelEntriesExtractQr(t *testing.T) {
	entries := labelEntries(map[string]any{
		"labels": []any{
			map[string]any{"id": "a", "qr": "https://x/app?entityId=a"},
			"bare-id",
			map[string]any{"id": "b"},
		},
	})
	if len(entries) != 3 {
		t.Fatalf("entries = %#v", entries)
	}
	if entries[0].ID != "a" || !strings.HasPrefix(entries[0].QR, "https://x/app") {
		t.Fatalf("entry0 = %#v", entries[0])
	}
	if entries[1].ID != "bare-id" || entries[1].QR != "" {
		t.Fatalf("entry1 = %#v", entries[1])
	}
	if entries[2].ID != "b" || entries[2].QR != "" {
		t.Fatalf("entry2 = %#v", entries[2])
	}
}

func TestLabelEntriesAcceptsJsonataSingletonObject(t *testing.T) {
	// One-pallet register: $each unwraps to a bare object, not []any.
	entries := labelEntries(map[string]any{
		"labels": map[string]any{
			"id": "NO-261103-2.1", "client": "NO",
			"qr": "https://plant.example/app?entityId=NO-261103-2.1",
		},
	})
	if len(entries) != 1 {
		t.Fatalf("entries = %#v", entries)
	}
	if entries[0].ID != "NO-261103-2.1" || entries[0].Client != "NO" {
		t.Fatalf("entry = %#v", entries[0])
	}
	if entries[0].QR != "https://plant.example/app?entityId=NO-261103-2.1" {
		t.Fatalf("qr = %q", entries[0].QR)
	}

	pdf, err := labelsPDF(map[string]any{
		"lotId": "NO-261103-2",
		"labels": map[string]any{
			"id": "NO-261103-2.1", "client": "NO",
			"qr": "https://plant.example/app?entityId=NO-261103-2.1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(pdf, []byte("no label ids in request")) {
		t.Fatalf("singleton label still treated as empty (len=%d)", len(pdf))
	}
}

func TestFormatPalletHeadlineAndCER(t *testing.T) {
	got := formatPalletHeadline("261012", "NO", "1", "NO-261012-1.1")
	if got != "261012 NO1.1" {
		t.Fatalf("headline = %q", got)
	}
	if got := formatPalletHeadline("261012", "FE", "1", "FE-261012-1.3"); got != "261012 FE1.3" {
		t.Fatalf("experiment-style headline = %q", got)
	}
	if formatCodiceCER("160214") != "EER 16.02.14" {
		t.Fatalf("cer = %q", formatCodiceCER("160214"))
	}
	if formatCodiceCER("EER 16.02.14") != "EER 16.02.14" {
		t.Fatalf("cer passthrough = %q", formatCodiceCER("EER 16.02.14"))
	}
	if formatFirLine("99") != "FIR: 99" {
		t.Fatalf("fir = %q", formatFirLine("99"))
	}
	if formatFirLine("FIR: 99") != "FIR: 99" {
		t.Fatalf("fir normalize = %q", formatFirLine("FIR: 99"))
	}
	if formatBancaliLine("16", "1200") != "16 bancali / peso totale 1200" {
		t.Fatalf("bancali = %q", formatBancaliLine("16", "1200"))
	}
}

func TestLabelSheetProgressiveFallback(t *testing.T) {
	sheet := labelSheetFromBody(map[string]any{
		"lotId": "FE-261012-1",
		"client": "FE",
		"labels": []any{map[string]any{"id": "FE-261012-1.2"}},
	})
	if sheet.ClientProgressive != "1" {
		t.Fatalf("progressive from loadId = %q", sheet.ClientProgressive)
	}
	got := formatPalletHeadline(sheet.Date, sheet.Client, sheet.ClientProgressive, "FE-261012-1.2")
	if got != "261012 FE1.2" {
		t.Fatalf("headline = %q", got)
	}
}

func TestSheetLineBaselinesClearHeadline(t *testing.T) {
	sheet := labelSheetFromBody(sampleLabelBody(false))
	lines := sheetLines(sheet, sheet.Labels[0])
	if len(lines) != 4 {
		t.Fatalf("lines = %d", len(lines))
	}
	if !strings.HasPrefix(lines[1].Text, "FIR: ") {
		t.Fatalf("fir line = %q", lines[1].Text)
	}
	// Cell top = baseline - em; consecutive tops must not overlap prior glyph box.
	prevBottom := 0.0
	for i, line := range lines {
		h := ptToMM(line.Size)
		top := line.YBaseline - h
		if i > 0 && top < prevBottom {
			t.Fatalf("line %d overlaps previous: top=%.2f prevBottom=%.2f", i, top, prevBottom)
		}
		prevBottom = line.YBaseline
	}
}
