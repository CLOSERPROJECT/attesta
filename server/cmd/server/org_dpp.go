package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/CLOSERPROJECT/attesta/server/internal/org"
	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

// OrgDPPPageView is the public Digital Link HTML page for a locked org DPP.
type OrgDPPPageView struct {
	PageBase
	DigitalLink  string
	GTIN         string
	Lot          string
	Serial       string
	IssuedAt     string
	ProductName  string
	Description  string
	Org          string
	SubjectID    string
	SnapshotJSON string
}

func (s *Server) loadOrgDPPByIdentity(ctx context.Context, gtin, lot, serial string) (org.DPPRecord, bool) {
	s.ensureOrgStores()
	if s.orgDPPs == nil {
		return org.DPPRecord{}, false
	}
	rec, err := s.orgDPPs.GetByIdentity(ctx, gtin, lot, serial)
	if err != nil {
		return org.DPPRecord{}, false
	}
	return rec, true
}

func (s *Server) serveOrgDPP(w http.ResponseWriter, r *http.Request, rec org.DPPRecord) {
	link := strings.TrimSpace(rec.DigitalLink)
	if link == "" {
		link = untp.URL(rec.GTIN, rec.Lot, rec.Serial)
	}
	if linkType := strings.TrimSpace(r.URL.Query().Get("linkType")); linkType != "" {
		scope, ok := untpLinksetScope(linkType)
		if !ok {
			http.NotFound(w, r)
			return
		}
		s.serveUNTPLinkset(w, r, link, scope)
		return
	}
	if prefersLinksetResponse(r) {
		s.serveUNTPLinkset(w, r, link, "all")
		return
	}
	if prefersJSONResponse(r) {
		writeJSON(w, org.BuildDPPCredential(requestBaseURL(r), rec))
		return
	}

	productName := strings.TrimSpace(rec.ProductName)
	if productName == "" {
		productName = "Digital Product Passport"
	}
	snapshotJSON := "{}"
	if rec.Snapshot != nil {
		if raw, err := json.MarshalIndent(rec.Snapshot, "", "  "); err == nil {
			snapshotJSON = string(raw)
		}
	}
	view := OrgDPPPageView{
		PageBase:     s.pageBase("org_dpp_body", "", productName),
		DigitalLink:  link,
		GTIN:         rec.GTIN,
		Lot:          rec.Lot,
		Serial:       rec.Serial,
		IssuedAt:     rec.IssuedAt,
		ProductName:  productName,
		Description:  strings.TrimSpace(rec.Description),
		Org:          rec.Org,
		SubjectID:    rec.SubjectID,
		SnapshotJSON: snapshotJSON,
	}
	if err := s.tmpl.ExecuteTemplate(w, "org_dpp.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) serveOrgDPPEventsGone(w http.ResponseWriter) {
	http.Error(w, "org dpp events not available", http.StatusGone)
}
