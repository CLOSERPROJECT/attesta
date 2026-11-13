package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/png"
	"strconv"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/go-pdf/fpdf"
)

// fpdf cannot import the pdf-lib template via gofpdi reliably; stamp a
// full-page PNG of blank chrome from experiment template at 300 DPI.
//
//go:embed assets/etichette-in.png
var labelTemplatePNG []byte

type labelEntry struct {
	ID     string
	QR     string
	Client string
}

type labelSheet struct {
	LoadID            string
	Date              string // YYMMDD
	Client            string
	ClientProgressive string
	NumeroFir         string
	WasteCode         string
	PalletCount       string
	NetWeight         string
	Labels            []labelEntry
}

// Landscape A4 in PDF points (matches experiment minimal-pdf.ts).
const (
	pageWPt = 841.92
	pageHPt = 595.32
	pageWMM = 297.0
	pageHMM = 210.0
)

func labelsPDF(body map[string]any) ([]byte, error) {
	sheet := labelSheetFromBody(body)

	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetCompression(false) // plant smoke: keep streams greppable on disk
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetTitle("Labels "+sheet.LoadID, false)
	pdf.SetAuthor("living-http", false)

	tplName := "etichette-in"
	pdf.RegisterImageOptionsReader(tplName, fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(labelTemplatePNG))
	if pdf.Error() != nil {
		return nil, pdf.Error()
	}

	const qrMM = 45.0

	if len(sheet.Labels) == 0 {
		pdf.AddPage()
		drawLabelBackground(pdf, tplName)
		pdf.SetFont("Helvetica", "B", 16)
		pdf.SetXY(20, 80)
		// Helvetica/WinAnsi has no em dash; ASCII avoids mojibake in viewers.
		pdf.CellFormat(pageWMM-40, 10, "Labels - "+sheet.LoadID, "", 0, "C", false, 0, "")
		pdf.SetXY(20, 95)
		pdf.SetFont("Helvetica", "", 12)
		pdf.CellFormat(pageWMM-40, 8, "(no label ids in request)", "", 0, "C", false, 0, "")
	} else {
		for i, entry := range sheet.Labels {
			pdf.AddPage()
			drawLabelBackground(pdf, tplName)
			drawCenteredStack(pdf, sheetLines(sheet, entry))

			if entry.QR == "" {
				pdf.SetFont("Helvetica", "I", 11)
				pdf.SetXY(pageWMM-qrMM-12, 8)
				pdf.CellFormat(qrMM, 8, "(no QR)", "", 0, "R", false, 0, "")
				continue
			}
			imgName := fmt.Sprintf("qr-%d", i)
			if err := registerQRPNG(pdf, imgName, entry.QR); err != nil {
				return nil, err
			}
			// Top-right; experiment used 64pt/~28pt inset — scale up for print-sharp scan.
			pdf.ImageOptions(imgName, pageWMM-qrMM-ptToMM(28), ptToMM(28), qrMM, qrMM, false,
				fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
			if pdf.Error() != nil {
				return nil, pdf.Error()
			}
		}
	}
	if err := pdf.Error(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawLabelBackground(pdf *fpdf.Fpdf, name string) {
	pdf.ImageOptions(name, 0, 0, pageWMM, pageHMM, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
}

func ptToMM(pt float64) float64 {
	return pt * 25.4 / 72.0
}

// baselineMMFromBottomPt → fpdf Y (mm from top) for a text baseline, matching
// experiment minimal-pdf.ts coords (y measured from page bottom in pt).
func baselineMMFromBottomPt(yBottomPt float64) float64 {
	return (pageHPt - yBottomPt) * pageHMM / pageHPt
}

type labelLine struct {
	Text      string
	Size      float64 // pt
	YBaseline float64 // mm from top (fpdf Text / Cell baseline)
	Style     string
}

func sheetLines(sheet labelSheet, entry labelEntry) []labelLine {
	client := entry.Client
	if client == "" {
		client = sheet.Client
	}
	// Four-line stack in the experiment band (~420→235 pt from bottom). Gaps are
	// sized for glyph ascent above each baseline so a 72pt headline clears FIR.
	return []labelLine{
		{Text: formatPalletHeadline(sheet.Date, client, sheet.ClientProgressive, entry.ID), Size: 72, Style: "B", YBaseline: baselineMMFromBottomPt(420)},
		{Text: formatFirLine(sheet.NumeroFir), Size: 40, Style: "B", YBaseline: baselineMMFromBottomPt(345)},
		{Text: formatCodiceCER(sheet.WasteCode), Size: 48, Style: "B", YBaseline: baselineMMFromBottomPt(290)},
		{Text: formatBancaliLine(sheet.PalletCount, sheet.NetWeight), Size: 40, Style: "B", YBaseline: baselineMMFromBottomPt(235)},
	}
}

func drawCenteredStack(pdf *fpdf.Fpdf, lines []labelLine) {
	for _, line := range lines {
		if line.Text == "" {
			continue
		}
		pdf.SetFont("Helvetica", line.Style, line.Size)
		// Cell Y is the top of the cell; put the top one em above the baseline so
		// large headline glyphs do not extend downward into the FIR line.
		h := ptToMM(line.Size)
		pdf.SetXY(20, line.YBaseline-h)
		pdf.CellFormat(pageWMM-40, h, line.Text, "", 0, "C", false, 0, "")
	}
}

func formatPalletHeadline(date, client, progressive, palletID string) string {
	if date == "" && client == "" && progressive == "" {
		return palletID
	}
	palletNum := palletNumber(palletID)
	// Experiment sample: `261012 FE1` then `.pallet` → `261012 FE1.1`
	tail := client + progressive
	if palletNum != "" {
		if tail != "" {
			tail += "." + palletNum
		} else {
			tail = palletNum
		}
	}
	switch {
	case date != "" && tail != "":
		return date + " " + tail
	case date != "":
		return date
	case tail != "":
		return tail
	default:
		return palletID
	}
}

func palletNumber(palletID string) string {
	if i := strings.LastIndex(palletID, "."); i >= 0 && i+1 < len(palletID) {
		return palletID[i+1:]
	}
	return ""
}

func formatFirLine(numeroFir string) string {
	s := strings.TrimSpace(numeroFir)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToUpper(s), "FIR:") {
		s = strings.TrimSpace(s[4:])
		if s == "" {
			return ""
		}
	}
	return "FIR: " + s
}

func formatCodiceCER(wasteCode string) string {
	code := strings.TrimSpace(wasteCode)
	if code == "" {
		return ""
	}
	upper := strings.ToUpper(code)
	if strings.HasPrefix(upper, "EER ") || strings.HasPrefix(upper, "CER ") {
		return code
	}
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, code)
	if len(digits) == 6 && strings.ReplaceAll(code, " ", "") == digits {
		return "EER " + digits[0:2] + "." + digits[2:4] + "." + digits[4:6]
	}
	if len(digits) >= 6 {
		parts := make([]string, 0, (len(digits)+1)/2)
		for i := 0; i < len(digits); i += 2 {
			end := i + 2
			if end > len(digits) {
				end = len(digits)
			}
			parts = append(parts, digits[i:end])
		}
		return "EER " + strings.Join(parts, ".")
	}
	return "EER " + code
}

func formatBancaliLine(palletCount, netWeight string) string {
	if palletCount == "" && netWeight == "" {
		return ""
	}
	if netWeight == "" {
		return palletCount + " bancali"
	}
	if palletCount == "" {
		return "peso totale " + netWeight
	}
	return palletCount + " bancali / peso totale " + netWeight
}

// registerQRPNG embeds a lossless QR (≥12 px/module) — fpdf barcode contrib uses JPEG/DCT.
func registerQRPNG(pdf *fpdf.Fpdf, name, payload string) error {
	raw, err := qr.Encode(payload, qr.M, qr.Auto)
	if err != nil {
		return err
	}
	modules := raw.Bounds().Dx()
	const pxPerModule = 12
	side := modules * pxPerModule
	scaled, err := barcode.Scale(raw, side, side)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, scaled); err != nil {
		return err
	}
	pdf.RegisterImageOptionsReader(name, fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(buf.Bytes()))
	return pdf.Error()
}

func labelSheetFromBody(body map[string]any) labelSheet {
	sheet := labelSheet{
		LoadID:            "labels",
		Date:              anyString(body["date"]),
		Client:            anyString(body["client"]),
		ClientProgressive: firstString(body, "clientProgressive", "progressive"),
		NumeroFir:         firstString(body, "numeroFir", "numeroFIR"),
		WasteCode:         firstString(body, "wasteCode", "codiceCer", "codiceCER"),
		PalletCount:       anyString(body["palletCount"]),
		NetWeight:         firstString(body, "netWeight", "totalWeight", "pesoTotale", "firWeight"),
		Labels:            labelEntries(body),
	}
	if id, ok := body["lotId"].(string); ok && id != "" {
		sheet.LoadID = id
	}
	if sheet.Date == "" {
		sheet.Date = dateFromLoadID(sheet.LoadID)
	}
	if sheet.ClientProgressive == "" {
		sheet.ClientProgressive = progressiveFromLoadID(sheet.LoadID)
	}
	if sheet.Client == "" && len(sheet.Labels) > 0 {
		sheet.Client = sheet.Labels[0].Client
	}
	return sheet
}

func firstString(body map[string]any, keys ...string) string {
	for _, k := range keys {
		if s := anyString(body[k]); s != "" {
			return s
		}
	}
	return ""
}

func dateFromLoadID(loadID string) string {
	// load id: {client}-{YYMMDD}-{n}
	parts := strings.Split(loadID, "-")
	if len(parts) >= 3 && len(parts[1]) == 6 {
		return parts[1]
	}
	return ""
}

func progressiveFromLoadID(loadID string) string {
	// load id: {client}-{YYMMDD}-{n}
	parts := strings.Split(loadID, "-")
	if len(parts) >= 3 && len(parts[1]) == 6 {
		return parts[2]
	}
	return ""
}

func anyString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	default:
		return fmt.Sprint(x)
	}
}

func labelEntries(body map[string]any) []labelEntry {
	raw, ok := body["labels"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []any:
		out := make([]labelEntry, 0, len(v))
		for _, item := range v {
			if e, ok := labelEntryFromAny(item); ok {
				out = append(out, e)
			}
		}
		return out
	case []string:
		out := make([]labelEntry, 0, len(v))
		for _, id := range v {
			if id != "" {
				out = append(out, labelEntry{ID: id})
			}
		}
		return out
	default:
		// jsonata-go unwraps a one-element $each into a bare object/string.
		if e, ok := labelEntryFromAny(v); ok {
			return []labelEntry{e}
		}
		return nil
	}
}

func labelEntryFromAny(item any) (labelEntry, bool) {
	switch v := item.(type) {
	case string:
		if v == "" {
			return labelEntry{}, false
		}
		return labelEntry{ID: v}, true
	case map[string]any:
		id, _ := v["id"].(string)
		if id == "" {
			return labelEntry{}, false
		}
		qrPayload, _ := v["qr"].(string)
		client, _ := v["client"].(string)
		return labelEntry{ID: id, QR: qrPayload, Client: client}, true
	default:
		return labelEntry{}, false
	}
}

func registerCSV(body map[string]any) ([]byte, string) {
	headers := []string{
		"client", "date", "wasteCode", "truckWeight", "truckTare",
		"palletCount", "notes", "lotId", "palletIds",
	}
	palletIds := ""
	if raw, ok := body["palletIds"]; ok {
		switch v := raw.(type) {
		case []any:
			parts := make([]string, 0, len(v))
			for _, id := range v {
				parts = append(parts, fmt.Sprint(id))
			}
			palletIds = strings.Join(parts, " ")
		case []string:
			palletIds = strings.Join(v, " ")
		default:
			palletIds = fmt.Sprint(v)
		}
	}
	row := make([]string, len(headers))
	for i, h := range headers {
		if h == "palletIds" {
			row[i] = csvEscape(palletIds)
			continue
		}
		row[i] = csvEscape(body[h])
	}
	text := strings.Join(headers, ",") + "\n" + strings.Join(row, ",") + "\n"
	loadID := "register"
	if s, ok := body["lotId"].(string); ok && s != "" {
		loadID = s
	}
	return []byte(text), loadID + ".csv"
}

func csvEscape(value any) string {
	if value == nil {
		return ""
	}
	s := fmt.Sprint(value)
	if strings.ContainsAny(s, ",\"\n\r") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
