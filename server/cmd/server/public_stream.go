package main

import (
	"context"
	"net/http"
	"strings"
	"time"
)

const publicStreamRecentRunLimit = 8

func processCompletedAt(process *Process) time.Time {
	if process == nil {
		return time.Time{}
	}
	var latest time.Time
	for _, step := range process.Progress {
		if step.DoneAt != nil && step.DoneAt.After(latest) {
			latest = *step.DoneAt
		}
	}
	if latest.IsZero() && process.DPP != nil && !process.DPP.GeneratedAt.IsZero() {
		return process.DPP.GeneratedAt.UTC()
	}
	return latest
}

func buildPublicStreamRuns(def WorkflowDef, processes []Process) []PublicStreamRunView {
	out := make([]PublicStreamRunView, 0, publicStreamRecentRunLimit)
	for i := range processes {
		p := processes[i]
		p.Progress = normalizeProgressKeys(p.Progress)
		if deriveProcessStatus(def, &p) != processStatusDone {
			continue
		}
		if p.DPP == nil {
			continue
		}
		gtin := strings.TrimSpace(p.DPP.GTIN)
		lot := strings.TrimSpace(p.DPP.Lot)
		serial := strings.TrimSpace(p.DPP.Serial)
		if gtin == "" || lot == "" || serial == "" {
			continue
		}
		out = append(out, PublicStreamRunView{
			CompletedAt: humanReadableTraceabilityTime(processCompletedAt(&p)),
			DigitalLink: digitalLinkURL(gtin, lot, serial),
		})
		if len(out) >= publicStreamRecentRunLimit {
			break
		}
	}
	return out
}

func buildPublicStreamOrganizations(cfg RuntimeConfig, logoURLs map[string]string) []PublicStreamCardOrgView {
	orgNames := organizationNameMap(cfg)
	seen := map[string]struct{}{}
	out := make([]PublicStreamCardOrgView, 0)
	for _, step := range sortedSteps(cfg.Workflow) {
		slug := strings.TrimSpace(step.OrganizationSlug)
		if slug == "" {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		name := organizationDisplayName(slug, orgNames)
		view := PublicStreamCardOrgView{
			Name:     name,
			Initials: organizationInitials(name),
		}
		if logoURLs != nil {
			view.LogoURL = strings.TrimSpace(logoURLs[slug])
		}
		out = append(out, view)
	}
	return out
}

func (s *Server) buildPublicStreamBlueprint(ctx context.Context, cfg RuntimeConfig, workflowKey string) StreamInstanceDetailView {
	preview := makeStreamInstanceDetailReadOnly(
		s.buildStreamInstanceDetailView(
			ctx,
			cfg,
			workflowKey,
			buildWorkflowPreviewProcess(cfg.Workflow, workflowKey),
			Actor{},
			"",
			"",
			false,
		),
		"Public preview.",
	)
	preview.HideStatus = true
	preview.WorkflowPath = publicStreamPath(workflowKey)
	// Browse-only blueprint: keep top-level steps open so substeps are visible.
	for i := range preview.Timeline {
		preview.Timeline[i].Expanded = true
	}
	return preview
}

func (s *Server) handlePublicStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimSuffix(r.URL.Path, "/")
	const prefix = "/streams/"
	if path == "/streams" || !strings.HasPrefix(path, prefix) {
		http.NotFound(w, r)
		return
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" || strings.Contains(rest, "/") || rest == "public" {
		http.NotFound(w, r)
		return
	}
	key := strings.TrimSpace(rest)
	cfg, err := s.workflowByKey(key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ctx := r.Context()
	logoURLs := organizationLogoURLMap(ctx, s.identity)
	header, err := s.buildPublicStreamCardView(ctx, key, cfg, logoURLs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	header.Href = "" // page header is not a link back to itself

	var recent []PublicStreamRunView
	if header.PassportEnabled && s.store != nil {
		processes, listErr := s.store.ListRecentProcessesByWorkflow(ctx, key, 0)
		if listErr != nil {
			http.Error(w, listErr.Error(), http.StatusInternalServerError)
			return
		}
		recent = buildPublicStreamRuns(cfg.Workflow, processes)
	}

	base := s.pageBase("public_stream_body", key, cfg.Workflow.Name)
	if user, _, err := s.currentUser(r); err == nil {
		base = s.pageBaseForUser(user, "public_stream_body", key, cfg.Workflow.Name)
	}

	view := PublicStreamPageView{
		PageBase:      base,
		HomeHref:      publicHomePath(cfg.Workflow.CategorySlug, cfg.Workflow.SubCategorySlug),
		Header:        header,
		Organizations: buildPublicStreamOrganizations(cfg, logoURLs),
		RecentRuns:    recent,
		Blueprint:     s.buildPublicStreamBlueprint(ctx, cfg, key),
	}
	if err := s.tmpl.ExecuteTemplate(w, "public_stream.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
