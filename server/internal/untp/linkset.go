package untp

import (
	"net/http"
	"strings"
)

// BuildLinkset returns the identity resolver linkset for a digital link:
// DPP credential (JSON), DTE events credential, and the HTML product page.
func BuildLinkset(baseURL, link string) Linkset {
	anchor := AbsoluteURL(baseURL, link)
	return Linkset{Linkset: []LinksetContext{{
		Anchor: anchor,
		DPP: []LinksetTarget{{
			Href:     anchor + "?format=json",
			Type:     "application/json",
			Title:    "Digital Product Passport",
			Hreflang: []string{"en"},
		}},
		DTE: []LinksetTarget{{
			Href:     anchor + "/events",
			Type:     "application/json",
			Title:    "Digital Traceability Events",
			Hreflang: []string{"en"},
		}},
		PIP: []LinksetTarget{{
			Href:     anchor,
			Type:     "text/html",
			Title:    "Product information page",
			Hreflang: []string{"en"},
		}},
	}}}
}

// LinksetScope maps a linkType query parameter (GS1 resolver / UNTP IDR-07)
// to a linkset scope. The second return is false for unknown link types.
func LinksetScope(linkType string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(linkType))
	switch normalized {
	case "linkset", "all":
		return "all", true
	case "dpp", "dte", "pip":
		return normalized, true
	default:
		return "", false
	}
}

// FilterLinkset restricts a linkset to a single relation. The second return is
// false when the relation has no links.
func FilterLinkset(linkset Linkset, relation string) (Linkset, bool) {
	if len(linkset.Linkset) == 0 {
		return Linkset{}, false
	}
	context := linkset.Linkset[0]
	filtered := LinksetContext{Anchor: context.Anchor}
	switch relation {
	case "dpp":
		filtered.DPP = context.DPP
	case "dte":
		filtered.DTE = context.DTE
	case "pip":
		filtered.PIP = context.PIP
	default:
		return Linkset{}, false
	}
	if filtered.DPP == nil && filtered.DTE == nil && filtered.PIP == nil {
		return Linkset{}, false
	}
	return Linkset{Linkset: []LinksetContext{filtered}}, true
}

// PrefersLinkset reports whether the request Accept header asks for linkset+json.
func PrefersLinkset(r *http.Request) bool {
	if r == nil {
		return false
	}
	accept := strings.ToLower(strings.TrimSpace(r.Header.Get("Accept")))
	return strings.Contains(accept, "application/linkset+json")
}
