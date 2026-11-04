package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/CLOSERPROJECT/attesta/server/internal/living"
	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

// LivingPassportPageView is the public Digital Link HTML page for a locked living Passport.
type LivingPassportPageView struct {
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

func (s *Server) loadLivingPassportByIdentity(ctx context.Context, gtin, lot, serial string) (living.PassportRecord, bool) {
	s.ensureLivingStores()
	if s.livingPassports == nil {
		return living.PassportRecord{}, false
	}
	rec, err := s.livingPassports.GetByIdentity(ctx, gtin, lot, serial)
	if err != nil {
		return living.PassportRecord{}, false
	}
	return rec, true
}

func (s *Server) serveLivingPassport(w http.ResponseWriter, r *http.Request, rec living.PassportRecord) {
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
		writeJSON(w, living.BuildPassportDPPCredential(requestBaseURL(r), rec))
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
	view := LivingPassportPageView{
		PageBase:     s.pageBase("living_passport_body", "", productName),
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
	if err := s.tmpl.ExecuteTemplate(w, "living_passport.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) serveLivingPassportEventsGone(w http.ResponseWriter) {
	// Living DTE is out of v1; locked Passports still advertise /events in the linkset.
	http.Error(w, "living passport events not available", http.StatusGone)
}
