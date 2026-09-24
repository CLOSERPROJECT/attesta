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
//	                                         (DTE) credential: one GS1 EPCIS 2.0
//	                                         ObjectEvent per completed substep,
//	                                         carrying the operator input values
//
// Credentials are issued unsigned in this pilot (no W3C VC proof). Payload
// integrity is anchored by the per-substep notarizations and merkle root shown
// on the HTML page and in the notarized export. See docs/untp.md.

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	untpVCCredentialsContext = "https://www.w3.org/ns/credentials/v2"
	untpVocabularyContext    = "https://vocabulary.uncefact.org/untp/0.8.0/context/"
	untpEPCISContext         = "https://ref.gs1.org/standards/epcis/2.0.0/epcis-context.jsonld"
	untpLinkTypeDTE          = "https://test.uncefact.org/vocabulary/linkTypes/dte"
	gs1DigitalLinkSchemeID   = "https://id.gs1.org"
	untpProductType          = "Product"

	// EPCIS user extension namespace for the Attesta workflow payload. The
	// prefix is declared inline in the DTE @context, which UNTP DTE 0.8.0
	// permits precisely because EPCIS extensions namespace at document level.
	attestaContextPrefix  = "attesta"
	attestaVocabularyPath = "/vocabulary/untp/"

	// CBV terms for "an existing product is observed/inspected without a
	// change of identity", which is what completing a workflow substep is.
	epcisActionObserve     = "OBSERVE"
	epcisBizStepInspecting = "inspecting"
	epcisObjectEventType   = "ObjectEvent"
)

// UNTPCredential is the W3C VC 2.0 envelope shared by the DPP and DTE
// credentials. CredentialSubject is a Product for a DPP and an array of
// lifecycle events for a DTE, so it stays an interface here.
type UNTPCredential struct {
	Type              []string             `json:"type"`
	Context           []interface{}        `json:"@context"`
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
	Type                []string               `json:"type"`
	ID                  string                 `json:"id"`
	Name                string                 `json:"name,omitempty"`
	Description         string                 `json:"description,omitempty"`
	IDScheme            UNTPIdentifierScheme   `json:"idScheme,omitzero"`
	BatchNumber         string                 `json:"batchNumber,omitempty"`
	ItemNumber          string                 `json:"itemNumber,omitempty"`
	IDGranularity       string                 `json:"idGranularity,omitempty"`
	ProductCategory     []UNTPClassification   `json:"productCategory,omitempty"`
	ProducedAtFacility  *UNTPFacility          `json:"producedAtFacility,omitzero"`
	CountryOfProduction *UNTPCountry           `json:"countryOfProduction,omitzero"`
	RelatedDocument     []UNTPLink             `json:"relatedDocument,omitempty"`
	Characteristics     map[string]interface{} `json:"characteristics,omitempty"`
}

// UNTPFacility identifies a production facility (producedAtFacility or
// modifiedAtFacility).
type UNTPFacility struct {
	Type         []string `json:"type,omitempty"`
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	RegisteredID string   `json:"registeredId,omitempty"`
}

// UNTPCountry is the ISO 3166-1 country of production.
type UNTPCountry struct {
	CountryCode string `json:"countryCode"`
	CountryName string `json:"countryName,omitempty"`
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

// UNTPObjectEvent is a GS1 EPCIS 2.0 ObjectEvent — the credential subject
// shape required by UNTP DTE 0.8.0, which references the EPCIS JSON Schema
// unmodified. One completed workflow substep is one OBSERVE event on the
// product, and the substep payload rides in the `attesta:substep` EPCIS user
// extension.
type UNTPObjectEvent struct {
	Type                string               `json:"type"`
	EventID             string               `json:"eventID,omitempty"`
	EventTime           string               `json:"eventTime"`
	EventTimeZoneOffset string               `json:"eventTimeZoneOffset"`
	Action              string               `json:"action"`
	BizStep             string               `json:"bizStep,omitempty"`
	EPCList             []string             `json:"epcList,omitempty"`
	ReadPoint           *UNTPEPCISLocation   `json:"readPoint,omitzero"`
	BizLocation         *UNTPEPCISLocation   `json:"bizLocation,omitzero"`
	BizTransactionList  []UNTPBizTransaction `json:"bizTransactionList,omitempty"`
	Substep             UNTPSubstepRecord    `json:"attesta:substep,omitzero"`
}

// UNTPEPCISLocation is an EPCIS readPoint / bizLocation reference.
type UNTPEPCISLocation struct {
	ID string `json:"id"`
}

// UNTPBizTransaction links a business document to an event. Attachments use a
// namespaced extension type because no CBV type describes a generic workflow
// evidence upload.
type UNTPBizTransaction struct {
	Type           string `json:"type"`
	BizTransaction string `json:"bizTransaction"`
}

// UNTPSubstepRecord is the EPCIS user extension that carries the workflow
// substep behind an event, including the values the operator entered. EPCIS
// has no generic slot for arbitrary captured form data, so this is the
// machine-readable home for it.
type UNTPSubstepRecord struct {
	SubstepID    string      `json:"substepId"`
	Title        string      `json:"title,omitempty"`
	StepID       string      `json:"stepId,omitempty"`
	StepTitle    string      `json:"stepTitle,omitempty"`
	Organization string      `json:"organization,omitempty"`
	Role         string      `json:"role,omitempty"`
	UNTPRole     string      `json:"untpRole,omitempty"`
	CompletedBy  string      `json:"completedBy,omitempty"`
	Input        interface{} `json:"input,omitempty"`
	Digest       string      `json:"digest,omitempty"`
}

type UNTPClassification struct {
	Code       string `json:"code,omitempty"`
	Name       string `json:"name,omitempty"`
	SchemeID   string `json:"schemeId,omitempty"`
	SchemeName string `json:"schemeName,omitempty"`
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

// The UNTP Product subject requires productCategory, producedAtFacility and
// countryOfProduction. Workflow config MAY supply them; when it does not, the
// builders below derive a value from what the stream already knows rather than
// emitting an invalid credential — or, worse, refusing to load the stream.

const (
	// ISO 3166-1 user-assigned code for an unspecified country. Emitted only
	// when neither the stream nor the deployment declares one: stating
	// "unspecified" beats inventing a country of production.
	untpUnknownCountryCode = "ZZ"
	untpUnknownCountryName = "Unspecified"

	untpCategorySchemeName = "Attesta stream categories"
	untpUnclassifiedCode   = "unclassified"
	untpUnclassifiedName   = "Unclassified"
)

// untpCategorySchemeID is the classification scheme for categories derived
// from the Attesta taxonomy. It resolves: see untp_vocabulary.go.
func untpCategorySchemeID(baseURL string) string {
	return attestaVocabularyIRI(baseURL) + "categories"
}

// untpProductCategory prefers the configured UN CPC classification and falls
// back to the stream's own taxonomy path under an Attesta-owned scheme.
func untpProductCategory(baseURL string, cfg RuntimeConfig) []UNTPClassification {
	if cfg.DPP.ProductCategory != nil {
		return []UNTPClassification{{
			Code:       cfg.DPP.ProductCategory.Code,
			Name:       cfg.DPP.ProductCategory.Name,
			SchemeID:   cfg.DPP.ProductCategory.SchemeID,
			SchemeName: cfg.DPP.ProductCategory.SchemeName,
		}}
	}
	code, name := untpUnclassifiedCode, untpUnclassifiedName
	if slug := strings.TrimSpace(cfg.Workflow.SubCategorySlug); slug != "" {
		code = slug
		name = humanizeSlug(slug)
	}
	return []UNTPClassification{{
		Code:       code,
		Name:       name,
		SchemeID:   untpCategorySchemeID(baseURL),
		SchemeName: untpCategorySchemeName,
	}}
}

func untpFacility(cfg *DPPFacility) *UNTPFacility {
	if cfg == nil {
		return nil
	}
	return &UNTPFacility{
		Type:         []string{"Facility"},
		ID:           cfg.ID,
		Name:         cfg.Name,
		RegisteredID: cfg.RegisteredID,
	}
}

// untpProducedAtFacility prefers the configured facility and falls back to the
// organization that owns the stream's first step — the closest thing to a
// production site the workflow actually declares.
func untpProducedAtFacility(baseURL string, cfg RuntimeConfig) *UNTPFacility {
	if facility := untpFacility(cfg.DPP.ProducedAtFacility); facility != nil {
		return facility
	}
	org, ok := untpFallbackOrganization(cfg)
	if !ok {
		return &UNTPFacility{
			Type: []string{"Facility"},
			ID:   strings.TrimRight(baseURL, "/"),
			Name: untpIssuerName(cfg),
		}
	}
	return &UNTPFacility{
		Type:         []string{"Facility"},
		ID:           strings.TrimRight(baseURL, "/") + "/organization/" + url.PathEscape(org.Slug),
		Name:         org.Name,
		RegisteredID: org.Slug,
	}
}

// untpFallbackOrganization returns the organization owning the first ordered
// step, else the first declared organization.
func untpFallbackOrganization(cfg RuntimeConfig) (WorkflowOrganization, bool) {
	orgs := cfg.Organizations
	if len(orgs) == 0 {
		return WorkflowOrganization{}, false
	}
	wanted := ""
	for _, step := range sortedSteps(cfg.Workflow) {
		if slug := strings.TrimSpace(step.OrganizationSlug); slug != "" {
			wanted = slug
			break
		}
	}
	for _, org := range orgs {
		if strings.TrimSpace(org.Slug) == wanted && strings.TrimSpace(org.Name) != "" {
			return org, true
		}
	}
	if strings.TrimSpace(orgs[0].Slug) == "" || strings.TrimSpace(orgs[0].Name) == "" {
		return WorkflowOrganization{}, false
	}
	return orgs[0], true
}

// untpCountry prefers the stream's country, then the deployment default
// (DPP_COUNTRY_OF_PRODUCTION), then an explicit "unspecified".
func untpCountry(cfg *DPPCountry) *UNTPCountry {
	if cfg != nil {
		return &UNTPCountry{
			CountryCode: cfg.CountryCode,
			CountryName: cfg.CountryName,
		}
	}
	if code, name := deploymentCountryOfProduction(); code != "" {
		return &UNTPCountry{CountryCode: code, CountryName: name}
	}
	return &UNTPCountry{CountryCode: untpUnknownCountryCode, CountryName: untpUnknownCountryName}
}

// humanizeSlug turns a taxonomy slug into a display name ("battery-cells" →
// "Battery cells"), used for derived classification names.
func humanizeSlug(slug string) string {
	words := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' })
	if len(words) == 0 {
		return slug
	}
	joined := strings.Join(words, " ")
	return strings.ToUpper(joined[:1]) + joined[1:]
}

// deploymentCountryOfProduction reads the operator-declared country used when
// a stream does not name one. Format: "NL" or "NL,Netherlands".
func deploymentCountryOfProduction() (string, string) {
	raw := strings.TrimSpace(os.Getenv("DPP_COUNTRY_OF_PRODUCTION"))
	if raw == "" {
		return "", ""
	}
	code, name, _ := strings.Cut(raw, ",")
	return strings.TrimSpace(code), strings.TrimSpace(name)
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

// attestaVocabularyIRI is the namespace IRI of the EPCIS user extension that
// carries substep payloads. It is declared as an inline @context entry on the
// DTE credential, so extension terms expand to a namespace this deployment
// owns instead of leaking into the EPCIS vocabulary.
func attestaVocabularyIRI(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + attestaVocabularyPath
}

// untpDTEContext is the three-entry DTE context (VCDM, UNTP, EPCIS) plus the
// inline Attesta extension definition, which declares the namespace and scopes
// the substep terms (see untp_vocabulary.go).
func untpDTEContext(baseURL string) []interface{} {
	return []interface{}{
		untpVCCredentialsContext,
		untpVocabularyContext,
		untpEPCISContext,
		attestaContextDefinition(baseURL),
	}
}

// untpRoleFor maps a workflow role slug to the UNTP PartyRole vocabulary.
// Returns "" when the role has no untpRole mapping; the caller then omits the
// value (arbitrary role strings are not valid PartyRole values).
func untpRoleFor(roles []WorkflowRole, roleSlug string) string {
	roleSlug = strings.TrimSpace(roleSlug)
	if roleSlug == "" {
		return ""
	}
	for _, r := range roles {
		if strings.TrimSpace(r.Slug) == roleSlug {
			return strings.TrimSpace(r.UNTpRole)
		}
	}
	return ""
}

// untpIssuerName is the operator name shown on credentials, falling back to
// the workflow name.
func untpIssuerName(cfg RuntimeConfig) string {
	if name := strings.TrimSpace(cfg.DPP.OwnerName); name != "" {
		return name
	}
	return strings.TrimSpace(cfg.Workflow.Name)
}

func untpIssuer(baseURL string, cfg RuntimeConfig) UNTPCredentialIssuer {
	return UNTPCredentialIssuer{Type: []string{"CredentialIssuer"}, ID: baseURL, Name: untpIssuerName(cfg)}
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
		Type:                []string{untpProductType},
		ID:                  productID,
		Name:                untpProductName(cfg),
		Description:         untpProductDescription(cfg),
		IDScheme:            UNTPIdentifierScheme{Type: []string{"IdentifierScheme"}, ID: gs1DigitalLinkSchemeID, Name: "GS1 Digital Link"},
		IDGranularity:       "item",
		ProductCategory:     untpProductCategory(baseURL, cfg),
		ProducedAtFacility:  untpProducedAtFacility(baseURL, cfg),
		CountryOfProduction: untpCountry(cfg.DPP.CountryOfProduction),
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
		Context:           []interface{}{untpVCCredentialsContext, untpVocabularyContext},
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

// buildUNTPTraceabilityEvents maps completed substeps to EPCIS 2.0
// ObjectEvents — each completion is an OBSERVE checkpoint on the product,
// carrying the values the operator entered in the `attesta:substep`
// extension. Substeps that are not done are not events.
func buildUNTPTraceabilityEvents(baseURL string, def WorkflowDef, cfg RuntimeConfig, process *Process, link string) []UNTPObjectEvent {
	if process == nil {
		return nil
	}
	productID := absoluteDigitalLinkURL(baseURL, link)
	var location *UNTPEPCISLocation
	if facility := untpProducedAtFacility(baseURL, cfg); facility != nil && strings.TrimSpace(facility.ID) != "" {
		location = &UNTPEPCISLocation{ID: facility.ID}
	}
	attachmentType := attestaVocabularyIRI(baseURL) + "attachment"
	fallbackTime := untpFallbackEventTime(process)
	events := make([]UNTPObjectEvent, 0)
	for _, step := range sortedSteps(def) {
		for _, sub := range sortedSubsteps(step) {
			progress, ok := process.Progress[sub.SubstepID]
			if !ok || progress.State != "done" {
				continue
			}
			eventTime := fallbackTime
			if progress.DoneAt != nil {
				eventTime = rfc3339UTC(*progress.DoneAt)
			}
			event := UNTPObjectEvent{
				Type:                epcisObjectEventType,
				EventID:             productID + "/events#" + url.QueryEscape(sub.SubstepID),
				EventTime:           eventTime,
				EventTimeZoneOffset: "+00:00",
				Action:              epcisActionObserve,
				BizStep:             epcisBizStepInspecting,
				EPCList:             []string{productID},
				ReadPoint:           location,
				BizLocation:         location,
				Substep:             buildUNTPSubstepRecord(step, sub, cfg, progress),
			}
			for _, meta := range attachmentsFromValue(progress.Data) {
				if strings.TrimSpace(meta.AttachmentID) == "" {
					continue
				}
				event.BizTransactionList = append(event.BizTransactionList, UNTPBizTransaction{
					Type:           attachmentType,
					BizTransaction: productID + "/attachment/" + url.PathEscape(meta.AttachmentID) + "/file",
				})
			}
			events = append(events, event)
		}
	}
	return events
}

// untpFallbackEventTime covers substeps completed before DoneAt was recorded:
// EPCIS requires eventTime on every event, so fall back to the passport
// generation time and then to process creation.
func untpFallbackEventTime(process *Process) string {
	if process.DPP != nil && !process.DPP.GeneratedAt.IsZero() {
		return rfc3339UTC(process.DPP.GeneratedAt)
	}
	return rfc3339UTC(process.CreatedAt)
}

// buildUNTPSubstepRecord assembles the EPCIS user extension for one completed
// substep: which workflow checkpoint it was, who closed it, the entered
// values, and the notarization digest over those values.
func buildUNTPSubstepRecord(step WorkflowStep, sub WorkflowSub, cfg RuntimeConfig, progress ProcessStep) UNTPSubstepRecord {
	record := UNTPSubstepRecord{
		SubstepID:    sub.SubstepID,
		Title:        strings.TrimSpace(sub.Title),
		StepID:       strings.TrimSpace(step.StepID),
		StepTitle:    strings.TrimSpace(step.Title),
		Organization: strings.TrimSpace(step.OrganizationSlug),
		Input:        untpSubstepInput(progress),
	}
	role := ""
	if progress.DoneBy != nil {
		role = strings.TrimSpace(progress.DoneBy.Role)
		record.CompletedBy = strings.TrimSpace(progress.DoneBy.ID)
	}
	if role == "" {
		role = strings.TrimSpace(sub.Role)
		if role == "" && len(sub.Roles) > 0 {
			role = strings.TrimSpace(sub.Roles[0])
		}
	}
	record.Role = role
	record.UNTPRole = untpRoleFor(cfg.Roles, role)
	if len(progress.Data) > 0 {
		record.Digest = "sha256:" + digestPayload(progress.Data)
	}
	return record
}

// untpSubstepInput returns the entered values with their JSON types intact.
// Attachment metadata is dropped: those are published as bizTransactionList
// download links instead.
func untpSubstepInput(progress ProcessStep) interface{} {
	if len(progress.Data) == 0 {
		return nil
	}
	input := make(map[string]interface{}, len(progress.Data))
	for key, raw := range progress.Data {
		if isAttachmentMetaValue(raw) {
			continue
		}
		input[key] = raw
	}
	if len(input) == 0 {
		return nil
	}
	return input
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
// credential for a digital link: one EPCIS ObjectEvent per completed substep.
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
		Context:           untpDTEContext(baseURL),
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
