package main

// UNTP (United Nations Transparency Protocol) pilot surface, served from the
// existing GS1 Digital Link identifiers (ISO/IEC 18975 structured paths):
//
//	GET /01/{gtin}/10/{lot}/21/{serial}
//	  no params (default link)              HTML landing page
//	  Accept: application/json | ?format=json
//	                                         UNTP Digital Product Passport (DPP)
//	                                         credential, W3C VC 2.0 envelope
//	  ?linkType=linkset|all | Accept: application/linkset+json
//	                                         full linkset (RFC 9264, UNTP IDR /
//	                                         GS1 resolver conventions)
//	  ?linkType=dpp|dte|pip                  linkset filtered to one relation
//	GET /01/{gtin}/10/{lot}/21/{serial}/events
//	                                         UNTP Digital Traceability Event
//	                                         (DTE) credential, one ModifyEvent
//	                                         per completed substep
//
// Credentials are issued unsigned in this pilot (no W3C VC proof). Payload
// integrity is anchored by the per-substep notarizations and merkle root shown
// on the HTML page and in the notarized export. See docs/untp.md.

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

const (
	untpVCCredentialsContext = "https://www.w3.org/ns/credentials/v2"
	untpVocabularyContext    = "https://vocabulary.uncefact.org/untp/"
	untpLinkTypeDTE          = "https://test.uncefact.org/vocabulary/linkTypes/dte"
	gs1DigitalLinkSchemeID   = "https://id.gs1.org"
)

// UNTPCredential is the W3C VC 2.0 envelope shared by the DPP and DTE
// credentials. CredentialSubject is a Product for a DPP and an array of
// lifecycle events for a DTE, so it stays an interface here.
type UNTPCredential struct {
	Type              []string             `json:"type"`
	Context           []string             `json:"@context"`
	ID                string               `json:"id"`
	Issuer            UNTPCredentialIssuer `json:"issuer"`
	Name              string               `json:"name,omitempty"`
	ValidFrom         string               `json:"validFrom,omitempty"`
	CredentialSubject interface{}          `json:"credentialSubject"`
}

type UNTPCredentialIssuer struct {
	Type []string `json:"type"`
	ID   string   `json:"id"`
	Name string   `json:"name,omitempty"`
}

// UNTPProduct is the credential subject of a Digital Product Passport.
type UNTPProduct struct {
	Type            []string               `json:"type"`
	ID              string                 `json:"id"`
	Name            string                 `json:"name,omitempty"`
	Description     string                 `json:"description,omitempty"`
	IDScheme        UNTPIdentifierScheme   `json:"idScheme,omitzero"`
	BatchNumber     string                 `json:"batchNumber,omitempty"`
	ItemNumber      string                 `json:"itemNumber,omitempty"`
	IDGranularity   string                 `json:"idGranularity,omitempty"`
	RelatedDocument []UNTPLink             `json:"relatedDocument,omitempty"`
	Characteristics map[string]interface{} `json:"characteristics,omitempty"`
}

type UNTPIdentifierScheme struct {
	Type []string `json:"type"`
	ID   string   `json:"id"`
	Name string   `json:"name,omitempty"`
}

// UNTPLink is a UNTP core vocabulary Link (relatedDocument entries).
type UNTPLink struct {
	LinkURL   string `json:"linkURL"`
	LinkName  string `json:"linkName,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
	LinkType  string `json:"linkType,omitempty"`
}

// UNTPModifyEvent is a UNTP ModifyEvent lifecycle event (checkpoint
// observation on the product, identity retained).
type UNTPModifyEvent struct {
	Type            []string           `json:"type"`
	ID              string             `json:"id"`
	Name            string             `json:"name,omitempty"`
	EventDate       string             `json:"eventDate,omitempty"`
	ActivityType    UNTPClassification `json:"activityType,omitzero"`
	ModifiedProduct []UNTPEventProduct `json:"modifiedProduct,omitempty"`
	RelatedParty    []UNTPPartyRole    `json:"relatedParty,omitempty"`
	RelatedDocument []UNTPLink         `json:"relatedDocument,omitempty"`
}

type UNTPClassification struct {
	Code       string `json:"code,omitempty"`
	Name       string `json:"name,omitempty"`
	SchemeID   string `json:"schemeId,omitempty"`
	SchemeName string `json:"schemeName,omitempty"`
}

type UNTPEventProduct struct {
	Product     UNTPProductRef `json:"product,omitzero"`
	Disposition string         `json:"disposition,omitempty"`
}

// UNTPProductRef is the product identification carried inside event products.
type UNTPProductRef struct {
	Type          []string `json:"type"`
	ID            string   `json:"id"`
	Name          string   `json:"name,omitempty"`
	BatchNumber   string   `json:"batchNumber,omitempty"`
	ItemNumber    string   `json:"itemNumber,omitempty"`
	IDGranularity string   `json:"idGranularity,omitempty"`
}

type UNTPPartyRole struct {
	Role  string    `json:"role"`
	Party UNTPParty `json:"party,omitzero"`
}

type UNTPParty struct {
	Type []string `json:"type"`
	ID   string   `json:"id"`
	Name string   `json:"name,omitempty"`
}

// UNTPLinkset is an RFC 9264 linkset as returned by the identity resolver.
// Relation arrays (dpp, dte, pip) are omitted when filtered out.
type UNTPLinkset struct {
	Linkset []UNTPLinksetContext `json:"linkset"`
}

type UNTPLinksetContext struct {
	Anchor string              `json:"anchor"`
	DPP    []UNTPLinksetTarget `json:"dpp,omitempty"`
	DTE    []UNTPLinksetTarget `json:"dte,omitempty"`
	PIP    []UNTPLinksetTarget `json:"pip,omitempty"`
}

type UNTPLinksetTarget struct {
	Href     string   `json:"href"`
	Type     string   `json:"type"`
	Title    string   `json:"title,omitempty"`
	Hreflang []string `json:"hreflang,omitempty"`
}

// absoluteDigitalLinkURL returns the fully qualified resolver URI for a
// digital link path. Per the UNTP IDR, identifiers used inside credentials are
// resolver URIs, not relative paths.
func absoluteDigitalLinkURL(baseURL, link string) string {
	if baseURL == "" {
		return link
	}
	return strings.TrimRight(baseURL, "/") + link
}

func untpIssuer(baseURL string, cfg RuntimeConfig) UNTPCredentialIssuer {
	name := strings.TrimSpace(cfg.DPP.OwnerName)
	if name == "" {
		name = strings.TrimSpace(cfg.Workflow.Name)
	}
	return UNTPCredentialIssuer{Type: []string{"CredentialIssuer"}, ID: baseURL, Name: name}
}

func untpProductName(cfg RuntimeConfig) string {
	if name := strings.TrimSpace(cfg.DPP.ProductName); name != "" {
		return name
	}
	return strings.TrimSpace(cfg.Workflow.Name)
}

func untpProductDescription(cfg RuntimeConfig) string {
	if description := strings.TrimSpace(cfg.DPP.ProductDescription); description != "" {
		return description
	}
	return strings.TrimSpace(cfg.Workflow.Description)
}

// buildUNTPDPPCredential maps a completed process to a UNTP Digital Product
// Passport credential. Product identity is the GS1 Digital Link (GTIN + lot +
// serial, item granularity).
func buildUNTPDPPCredential(baseURL string, cfg RuntimeConfig, workflowKey string, process *Process, link string) UNTPCredential {
	productID := absoluteDigitalLinkURL(baseURL, link)
	subject := UNTPProduct{
		Type:          []string{"Product"},
		ID:            productID,
		Name:          untpProductName(cfg),
		Description:   untpProductDescription(cfg),
		IDScheme:      UNTPIdentifierScheme{Type: []string{"IdentifierScheme"}, ID: gs1DigitalLinkSchemeID, Name: "GS1 Digital Link"},
		IDGranularity: "item",
		RelatedDocument: []UNTPLink{{
			LinkURL:   productID + "/events",
			LinkName:  "Digital Traceability Events",
			MediaType: "application/json",
			LinkType:  untpLinkTypeDTE,
		}},
		Characteristics: map[string]interface{}{
			"workflow":     workflowKey,
			"workflowName": strings.TrimSpace(cfg.Workflow.Name),
		},
	}
	if process != nil {
		subject.Characteristics["processId"] = process.ID.Hex()
		if process.DPP != nil {
			subject.BatchNumber = process.DPP.Lot
			subject.ItemNumber = process.DPP.Serial
			subject.Characteristics["gs1Element"] = gs1ElementString(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)
		}
	}
	credential := UNTPCredential{
		Type:              []string{"DigitalProductPassport", "VerifiableCredential"},
		Context:           []string{untpVCCredentialsContext, untpVocabularyContext},
		ID:                productID,
		Issuer:            untpIssuer(baseURL, cfg),
		Name:              "Digital Product Passport — " + untpProductName(cfg),
		CredentialSubject: subject,
	}
	if process != nil && process.DPP != nil {
		credential.ValidFrom = rfc3339UTC(process.DPP.GeneratedAt)
	}
	return credential
}

// buildUNTPTraceabilityEvents maps completed substeps to UNTP ModifyEvents —
// each completion is a checkpoint observation on the product. Substeps that
// are not done are not events.
func buildUNTPTraceabilityEvents(baseURL string, def WorkflowDef, cfg RuntimeConfig, process *Process, link string) []UNTPModifyEvent {
	if process == nil {
		return nil
	}
	productID := absoluteDigitalLinkURL(baseURL, link)
	productRef := UNTPProductRef{
		Type:          []string{"Product"},
		ID:            productID,
		Name:          untpProductName(cfg),
		IDGranularity: "item",
	}
	if process.DPP != nil {
		productRef.BatchNumber = process.DPP.Lot
		productRef.ItemNumber = process.DPP.Serial
	}
	workflowName := strings.TrimSpace(cfg.Workflow.Name)
	events := make([]UNTPModifyEvent, 0)
	for _, sub := range orderedSubsteps(def) {
		progress, ok := process.Progress[sub.SubstepID]
		if !ok || progress.State != "done" {
			continue
		}
		event := UNTPModifyEvent{
			Type: []string{"ModifyEvent", "LifecycleEvent"},
			ID:   productID + "/events#" + url.QueryEscape(sub.SubstepID),
			Name: strings.TrimSpace(sub.Title),
			ActivityType: UNTPClassification{
				Code:       sub.SubstepID,
				Name:       strings.TrimSpace(sub.Title),
				SchemeID:   baseURL,
				SchemeName: workflowName,
			},
			ModifiedProduct: []UNTPEventProduct{{
				Product:     productRef,
				Disposition: "active",
			}},
		}
		if progress.DoneAt != nil {
			event.EventDate = rfc3339UTC(*progress.DoneAt)
		}
		if progress.DoneBy != nil {
			role := strings.TrimSpace(progress.DoneBy.Role)
			if role == "" {
				role = strings.TrimSpace(sub.Role)
				if role == "" && len(sub.Roles) > 0 {
					role = strings.TrimSpace(sub.Roles[0])
				}
			}
			if actorID := strings.TrimSpace(progress.DoneBy.ID); actorID != "" {
				event.RelatedParty = []UNTPPartyRole{{
					Role: role,
					Party: UNTPParty{
						Type: []string{"Party"},
						ID:   baseURL + "/#actor=" + url.QueryEscape(actorID),
						Name: actorID,
					},
				}}
			}
		}
		for _, meta := range attachmentsFromValue(progress.Data) {
			if strings.TrimSpace(meta.AttachmentID) == "" {
				continue
			}
			event.RelatedDocument = append(event.RelatedDocument, UNTPLink{
				LinkURL:   productID + "/attachment/" + url.PathEscape(meta.AttachmentID) + "/file",
				LinkName:  meta.Filename,
				MediaType: meta.ContentType,
			})
		}
		events = append(events, event)
	}
	return events
}

// buildUNTPLinkset returns the identity resolver linkset for a digital link:
// the DPP credential (JSON), the DTE events credential, and the human-readable
// product information page (default link).
func buildUNTPLinkset(baseURL, link string) UNTPLinkset {
	anchor := absoluteDigitalLinkURL(baseURL, link)
	return UNTPLinkset{Linkset: []UNTPLinksetContext{{
		Anchor: anchor,
		DPP: []UNTPLinksetTarget{{
			Href:     anchor + "?format=json",
			Type:     "application/json",
			Title:    "Digital Product Passport",
			Hreflang: []string{"en"},
		}},
		DTE: []UNTPLinksetTarget{{
			Href:     anchor + "/events",
			Type:     "application/json",
			Title:    "Digital Traceability Events",
			Hreflang: []string{"en"},
		}},
		PIP: []UNTPLinksetTarget{{
			Href:     anchor,
			Type:     "text/html",
			Title:    "Product information page",
			Hreflang: []string{"en"},
		}},
	}}}
}

// untpLinksetScope maps a linkType query parameter (GS1 resolver convention,
// UNTP IDR-07) to a linkset scope. The second return is false for unknown
// link types.
func untpLinksetScope(linkType string) (string, bool) {
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

// filterUNTPLinkset restricts a linkset to a single relation. The second
// return is false when the relation has no links.
func filterUNTPLinkset(linkset UNTPLinkset, relation string) (UNTPLinkset, bool) {
	if len(linkset.Linkset) == 0 {
		return UNTPLinkset{}, false
	}
	context := linkset.Linkset[0]
	filtered := UNTPLinksetContext{Anchor: context.Anchor}
	switch relation {
	case "dpp":
		filtered.DPP = context.DPP
	case "dte":
		filtered.DTE = context.DTE
	case "pip":
		filtered.PIP = context.PIP
	default:
		return UNTPLinkset{}, false
	}
	if filtered.DPP == nil && filtered.DTE == nil && filtered.PIP == nil {
		return UNTPLinkset{}, false
	}
	return UNTPLinkset{Linkset: []UNTPLinksetContext{filtered}}, true
}

func prefersLinksetResponse(r *http.Request) bool {
	accept := strings.ToLower(strings.TrimSpace(r.Header.Get("Accept")))
	return strings.Contains(accept, "application/linkset+json")
}

// serveUNTPLinkset writes the resolver linkset for a digital link, optionally
// filtered to one relation by scope ("all" or a relation name). Unknown
// relations 404 per the resolver workflow.
func (s *Server) serveUNTPLinkset(w http.ResponseWriter, r *http.Request, link, scope string) {
	linkset := buildUNTPLinkset(requestBaseURL(r), link)
	if scope != "all" {
		filtered, ok := filterUNTPLinkset(linkset, scope)
		if !ok {
			http.NotFound(w, r)
			return
		}
		linkset = filtered
	}
	w.Header().Set("Content-Type", "application/linkset+json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(linkset)
}

// handleDigitalLinkEvents serves the UNTP Digital Traceability Event
// credential for a digital link: one ModifyEvent per completed substep.
func (s *Server) handleDigitalLinkEvents(w http.ResponseWriter, r *http.Request, gtin, lot, serial string) {
	process, err := s.store.LoadProcessByDigitalLink(r.Context(), gtin, lot, serial)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	process.Progress = normalizeProgressKeys(process.Progress)

	workflowKey := strings.TrimSpace(process.WorkflowKey)
	if workflowKey == "" {
		workflowKey = s.defaultWorkflowKey()
	}
	cfg, err := s.workflowByKey(workflowKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	link := digitalLinkURL(gtin, lot, serial)
	baseURL := requestBaseURL(r)
	credential := UNTPCredential{
		Type:              []string{"DigitalTraceabilityEvent", "VerifiableCredential"},
		Context:           []string{untpVCCredentialsContext, untpVocabularyContext},
		ID:                absoluteDigitalLinkURL(baseURL, link) + "/events",
		Issuer:            untpIssuer(baseURL, cfg),
		Name:              "Digital Traceability Events — " + untpProductName(cfg),
		CredentialSubject: buildUNTPTraceabilityEvents(baseURL, cfg.Workflow, cfg, process, link),
	}
	if process.DPP != nil {
		credential.ValidFrom = rfc3339UTC(process.DPP.GeneratedAt)
	}
	writeJSON(w, credential)
}
