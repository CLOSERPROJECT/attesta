package untp

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrGTINRequired = errors.New("gtin is required")
	ErrGTINNonDigit = errors.New("gtin must contain only digits")
	ErrGTINTooLong  = errors.New("gtin must be at most 14 digits")
)

// NormalizeGTIN trims, validates digits, and left-pads to 14 digits.
func NormalizeGTIN(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrGTINRequired
	}
	for _, char := range trimmed {
		if char < '0' || char > '9' {
			return "", fmt.Errorf("%w: %q", ErrGTINNonDigit, raw)
		}
	}
	if len(trimmed) > 14 {
		return "", fmt.Errorf("%w: %q", ErrGTINTooLong, raw)
	}
	if len(trimmed) < 14 {
		trimmed = strings.Repeat("0", 14-len(trimmed)) + trimmed
	}
	return trimmed, nil
}

// ParsePath parses /01/{gtin}/10/{lot}/21/{serial}.
func ParsePath(path string) (gtin, lot, serial string, err error) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	parts := strings.Split(trimmed, "/")
	return parseParts(parts)
}

// ParseAttachmentPath matches /01/{gtin}/10/{lot}/21/{serial}/attachment/{id}/file.
// ok is true when the path carries the attachment suffix.
func ParseAttachmentPath(path string) (gtin, lot, serial, attachmentID string, ok bool, err error) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 9 || parts[6] != "attachment" || parts[8] != "file" {
		return "", "", "", "", false, nil
	}
	gtin, lot, serial, err = parseParts(parts[:6])
	if err != nil {
		return "", "", "", "", true, err
	}
	attachmentID, err = url.PathUnescape(parts[7])
	if err != nil {
		return "", "", "", "", true, err
	}
	attachmentID = strings.TrimSpace(attachmentID)
	if attachmentID == "" {
		return "", "", "", "", true, errors.New("missing attachment id")
	}
	return gtin, lot, serial, attachmentID, true, nil
}

// ParseEventsPath matches /01/{gtin}/10/{lot}/21/{serial}/events.
// ok is true when the path carries the /events suffix.
func ParseEventsPath(path string) (gtin, lot, serial string, ok bool, err error) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 7 || parts[6] != "events" {
		return "", "", "", false, nil
	}
	gtin, lot, serial, err = parseParts(parts[:6])
	if err != nil {
		return "", "", "", true, err
	}
	return gtin, lot, serial, true, nil
}

func parseParts(parts []string) (string, string, string, error) {
	if len(parts) != 6 || parts[0] != "01" || parts[2] != "10" || parts[4] != "21" {
		return "", "", "", errors.New("invalid digital link path")
	}
	gtinRaw, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", "", "", err
	}
	gtin, err := NormalizeGTIN(gtinRaw)
	if err != nil {
		return "", "", "", err
	}
	lot, err := url.PathUnescape(parts[3])
	if err != nil {
		return "", "", "", err
	}
	serial, err := url.PathUnescape(parts[5])
	if err != nil {
		return "", "", "", err
	}
	lot = strings.TrimSpace(lot)
	serial = strings.TrimSpace(serial)
	if lot == "" || serial == "" {
		return "", "", "", errors.New("missing lot or serial")
	}
	return gtin, lot, serial, nil
}

// URL builds the relative GS1 Digital Link path for gtin/lot/serial.
func URL(gtin, lot, serial string) string {
	return "/01/" + url.PathEscape(strings.TrimSpace(gtin)) +
		"/10/" + url.PathEscape(strings.TrimSpace(lot)) +
		"/21/" + url.PathEscape(strings.TrimSpace(serial))
}

// GS1ElementString formats the AI concatenation (01)(10)(21), or "" if any part is blank.
func GS1ElementString(gtin, lot, serial string) string {
	trimmedGTIN := strings.TrimSpace(gtin)
	trimmedLot := strings.TrimSpace(lot)
	trimmedSerial := strings.TrimSpace(serial)
	if trimmedGTIN == "" || trimmedLot == "" || trimmedSerial == "" {
		return ""
	}
	return fmt.Sprintf("(01)%s(10)%s(21)%s", trimmedGTIN, trimmedLot, trimmedSerial)
}

// AbsoluteURL joins baseURL with a relative digital link path for credential IDs.
func AbsoluteURL(baseURL, link string) string {
	if baseURL == "" {
		return link
	}
	return strings.TrimRight(baseURL, "/") + link
}
