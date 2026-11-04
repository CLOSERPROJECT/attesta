package living

import (
	"strings"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

// PassportProductIdentity maps a frozen Passport record to UNTP product identity.
func PassportProductIdentity(rec PassportRecord) untp.ProductIdentity {
	name := strings.TrimSpace(rec.ProductName)
	if name == "" {
		name = "Digital Product Passport"
	}
	var validFrom time.Time
	if issued := strings.TrimSpace(rec.IssuedAt); issued != "" {
		if parsed, err := time.Parse(time.RFC3339, issued); err == nil {
			validFrom = parsed
		}
	}
	return untp.ProductIdentity{
		Name:        name,
		Description: strings.TrimSpace(rec.Description),
		OwnerName:   strings.TrimSpace(rec.Org),
		BatchNumber: strings.TrimSpace(rec.Lot),
		ItemNumber:  strings.TrimSpace(rec.Serial),
		GTIN:        strings.TrimSpace(rec.GTIN),
		ValidFrom:   validFrom,
	}
}

// BuildPassportDPPCredential builds a UNTP DPP credential from a locked Passport snapshot.
func BuildPassportDPPCredential(baseURL string, rec PassportRecord) untp.Credential {
	link := strings.TrimSpace(rec.DigitalLink)
	if link == "" {
		link = untp.URL(rec.GTIN, rec.Lot, rec.Serial)
	}
	characteristics := map[string]interface{}{
		"org":       strings.TrimSpace(rec.Org),
		"subjectId": strings.TrimSpace(rec.SubjectID),
		"snapshot":  cloneAnyMap(rec.Snapshot),
	}
	if v := strings.TrimSpace(rec.ConfigVersion); v != "" {
		characteristics["configVersion"] = v
	}
	return untp.BuildDPPCredential(untp.DPPInput{
		BaseURL:         baseURL,
		Link:            link,
		Product:         PassportProductIdentity(rec),
		Characteristics: characteristics,
	})
}
