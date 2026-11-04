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
//
// Shared Digital Link / UNTP shaping lives in internal/untp; this file adapts
// Stream process state into those helpers and owns Stream HTTP handlers.

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

const (
	untpVCCredentialsContext = untp.VCCredentialsContext
	untpVocabularyContext    = untp.VocabularyContext
	untpLinkTypeDTE          = untp.LinkTypeDTE
	gs1DigitalLinkSchemeID   = untp.GS1DigitalLinkScheme
)

type UNTPCredential = untp.Credential
type UNTPCredentialIssuer = untp.CredentialIssuer
type UNTPProduct = untp.Product
type UNTPIdentifierScheme = untp.IdentifierScheme
type UNTPLink = untp.Link
type UNTPModifyEvent = untp.ModifyEvent
type UNTPClassification = untp.Classification
type UNTPEventProduct = untp.EventProduct
type UNTPProductRef = untp.ProductRef
type UNTPPartyRole = untp.PartyRole
type UNTPParty = untp.Party
type UNTPLinkset = untp.Linkset
type UNTPLinksetContext = untp.LinksetContext
type UNTPLinksetTarget = untp.LinksetTarget

func untpProductIdentity(cfg RuntimeConfig, process *Process) untp.ProductIdentity {
	identity := untp.ProductIdentity{
		Name:        untpProductName(cfg),
		Description: untpProductDescription(cfg),
		OwnerName:   untpOwnerName(cfg),
	}
	if process != nil && process.DPP != nil {
		identity.BatchNumber = process.DPP.Lot
		identity.ItemNumber = process.DPP.Serial
		identity.GTIN = process.DPP.GTIN
		identity.ValidFrom = process.DPP.GeneratedAt
	}
	return identity
}

func untpOwnerName(cfg RuntimeConfig) string {
	name := strings.TrimSpace(cfg.DPP.OwnerName)
	if name == "" {
		name = strings.TrimSpace(cfg.Workflow.Name)
	}
	return name
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

func buildUNTPDPPCredential(baseURL string, cfg RuntimeConfig, workflowKey string, process *Process, link string) UNTPCredential {
	characteristics := map[string]interface{}{
		"workflow":     workflowKey,
		"workflowName": strings.TrimSpace(cfg.Workflow.Name),
	}
	if process != nil {
		characteristics["processId"] = process.ID.Hex()
	}
	return untp.BuildDPPCredential(untp.DPPInput{
		BaseURL:         baseURL,
		Link:            link,
		Product:         untpProductIdentity(cfg, process),
		Characteristics: characteristics,
	})
}

func buildUNTPTraceabilityEvents(baseURL string, def WorkflowDef, cfg RuntimeConfig, process *Process, link string) []UNTPModifyEvent {
	if process == nil {
		return nil
	}
	return untp.BuildModifyEvents(baseURL, link, untpProductIdentity(cfg, process), strings.TrimSpace(cfg.Workflow.Name), streamUNTPEventInputs(def, process))
}

func streamUNTPEventInputs(def WorkflowDef, process *Process) []untp.EventInput {
	events := make([]untp.EventInput, 0)
	for _, sub := range orderedSubsteps(def) {
		progress, ok := process.Progress[sub.SubstepID]
		if !ok || progress.State != "done" {
			continue
		}
		event := untp.EventInput{
			ID:    sub.SubstepID,
			Title: strings.TrimSpace(sub.Title),
		}
		if progress.DoneAt != nil {
			event.EventDate = *progress.DoneAt
		}
		if progress.DoneBy != nil {
			role := strings.TrimSpace(progress.DoneBy.Role)
			if role == "" {
				role = strings.TrimSpace(sub.Role)
				if role == "" && len(sub.Roles) > 0 {
					role = strings.TrimSpace(sub.Roles[0])
				}
			}
			event.ActorID = strings.TrimSpace(progress.DoneBy.ID)
			event.ActorRole = role
		}
		for _, meta := range attachmentsFromValue(progress.Data) {
			if strings.TrimSpace(meta.AttachmentID) == "" {
				continue
			}
			event.Attachments = append(event.Attachments, untp.AttachmentRef{
				ID:          meta.AttachmentID,
				Filename:    meta.Filename,
				ContentType: meta.ContentType,
			})
		}
		events = append(events, event)
	}
	return events
}

func buildUNTPLinkset(baseURL, link string) UNTPLinkset {
	return untp.BuildLinkset(baseURL, link)
}

func untpLinksetScope(linkType string) (string, bool) {
	return untp.LinksetScope(linkType)
}

func filterUNTPLinkset(linkset UNTPLinkset, relation string) (UNTPLinkset, bool) {
	return untp.FilterLinkset(linkset, relation)
}

func prefersLinksetResponse(r *http.Request) bool {
	return untp.PrefersLinkset(r)
}

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
	writeJSON(w, untp.BuildDTECredential(untp.DTEInput{
		BaseURL: requestBaseURL(r),
		Link:    link,
		Product: untpProductIdentity(cfg, process),
		Scheme:  strings.TrimSpace(cfg.Workflow.Name),
		Events:  streamUNTPEventInputs(cfg.Workflow, process),
	}))
}
