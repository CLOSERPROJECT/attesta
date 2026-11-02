package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

func labelsPDF(body map[string]any) ([]byte, error) {
	loadID := "labels"
	if id, ok := body["loadId"].(string); ok && id != "" {
		loadID = id
	}
	ids := labelIDs(body)

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetCompression(false) // plant smoke: keep streams greppable on disk
	pdf.SetTitle("Labels "+loadID, false)
	pdf.SetAuthor("living-http", false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.Cell(0, 10, "Labels")
	pdf.Ln(12)
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(0, 8, "Load: "+loadID)
	pdf.Ln(10)
	if len(ids) == 0 {
		pdf.Cell(0, 8, "(no label ids in request)")
	} else {
		pdf.SetFont("Helvetica", "B", 12)
		pdf.Cell(0, 8, "Pallet labels")
		pdf.Ln(8)
		pdf.SetFont("Helvetica", "", 11)
		for _, id := range ids {
			pdf.Cell(0, 7, "- "+id)
			pdf.Ln(7)
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

func labelIDs(body map[string]any) []string {
	raw, ok := body["labels"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if id := labelIDFromAny(item); id != "" {
				out = append(out, id)
			}
		}
		return out
	case []string:
		return append([]string(nil), v...)
	default:
		return nil
	}
}

func labelIDFromAny(item any) string {
	switch v := item.(type) {
	case string:
		return v
	case map[string]any:
		if id, ok := v["id"].(string); ok {
			return id
		}
	}
	return ""
}

func registerCSV(body map[string]any) ([]byte, string) {
	headers := []string{
		"client", "date", "wasteCode", "truckWeight", "truckTare",
		"palletCount", "notes", "loadId", "palletIds",
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
	if s, ok := body["loadId"].(string); ok && s != "" {
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
