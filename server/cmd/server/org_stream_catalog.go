package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/org"
)

const (
	workflowKindOrg     = "org"
	orgCatalogKeyPrefix = "org-"
)

func orgCatalogKey(streamKey string) string {
	return orgCatalogKeyPrefix + strings.TrimSpace(streamKey)
}

func parseOrgCatalogKey(key string) (streamKey string, ok bool) {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, orgCatalogKeyPrefix) {
		return "", false
	}
	streamKey = strings.TrimSpace(strings.TrimPrefix(key, orgCatalogKeyPrefix))
	if streamKey == "" || strings.Contains(streamKey, "/") {
		return "", false
	}
	return streamKey, true
}

func (cfg RuntimeConfig) isOrgKind() bool {
	return strings.TrimSpace(cfg.Kind) == workflowKindOrg
}

func appOrgStreamNewPath(streamKey string) string {
	return "/app/" + strings.TrimSpace(streamKey) + "/new"
}

func runtimeConfigFromOrgStream(cfg org.OrgConfig, owningOrg string) RuntimeConfig {
	stream := cfg.Stream
	if stream == nil {
		return RuntimeConfig{Kind: workflowKindOrg}
	}
	out := RuntimeConfig{
		Kind:         workflowKindOrg,
		OrgStreamKey: strings.TrimSpace(stream.Key),
		Workflow: WorkflowDef{
			Name:            strings.TrimSpace(stream.Title),
			Description:     strings.TrimSpace(stream.Description),
			CategorySlug:    strings.TrimSpace(stream.CategorySlug),
			SubCategorySlug: strings.TrimSpace(stream.SubCategorySlug),
		},
	}
	orgSlug := strings.TrimSpace(owningOrg)
	if orgSlug == "" {
		orgSlug = strings.TrimSpace(cfg.Org)
	}
	if orgSlug != "" {
		out.Organizations = []WorkflowOrganization{{Slug: orgSlug, Name: orgSlug}}
	}
	return out
}

func (s *Server) loadOrgActiveConfig(ctx context.Context, scope org.Scope) (org.OrgConfig, error) {
	s.ensureOrgStores()
	if s.orgConfigs == nil {
		return org.OrgConfig{}, org.ErrNotFound
	}
	version, err := s.orgConfigs.ActiveVersion(ctx, scope)
	if err != nil {
		return org.OrgConfig{}, err
	}
	return s.orgConfigs.Get(ctx, scope, version)
}

func (s *Server) orgRuntimeConfigForUser(ctx context.Context, user *AccountUser, streamKey string) (RuntimeConfig, error) {
	if user == nil {
		return RuntimeConfig{}, fmt.Errorf("workflow %q not found", orgCatalogKey(streamKey))
	}
	orgSlug := strings.TrimSpace(user.OrgSlug)
	if orgSlug == "" || user.IsPlatformAdmin {
		return RuntimeConfig{}, fmt.Errorf("workflow %q not found", orgCatalogKey(streamKey))
	}
	streamKey = strings.TrimSpace(streamKey)
	scope := org.Scope{Org: orgSlug, Stream: streamKey}
	cfg, err := s.loadOrgActiveConfig(ctx, scope)
	if err != nil {
		if errors.Is(err, org.ErrNotFound) {
			return RuntimeConfig{}, fmt.Errorf("workflow %q not found", orgCatalogKey(streamKey))
		}
		return RuntimeConfig{}, err
	}
	if cfg.Stream == nil || strings.TrimSpace(cfg.Stream.Key) != streamKey {
		return RuntimeConfig{}, fmt.Errorf("workflow %q not found", orgCatalogKey(streamKey))
	}
	return runtimeConfigFromOrgStream(cfg, orgSlug), nil
}

func (s *Server) resolveWorkflowForUser(ctx context.Context, key string, user *AccountUser) (RuntimeConfig, error) {
	if streamKey, ok := parseOrgCatalogKey(key); ok {
		return s.orgRuntimeConfigForUser(ctx, user, streamKey)
	}
	return s.workflowByKey(key)
}

func orgStreamIsStartable(user *AccountUser, cfg org.OrgConfig) bool {
	if user == nil || user.IsPlatformAdmin {
		return false
	}
	if strings.TrimSpace(user.OrgSlug) == "" {
		return false
	}
	allowed := org.AnchorStartRoles(cfg)
	if len(allowed) == 0 {
		return false
	}
	return len(intersectRoles(allowed, user.RoleSlugs)) > 0
}

func (s *Server) listOrgDiscoveryStreams(ctx context.Context, user *AccountUser) ([]StreamDiscoveryItem, error) {
	if s == nil || user == nil || user.IsPlatformAdmin {
		return nil, nil
	}
	orgSlug := strings.TrimSpace(user.OrgSlug)
	if orgSlug == "" {
		return nil, nil
	}
	s.ensureOrgStores()
	if s.orgConfigs == nil {
		return nil, nil
	}
	streams, err := s.orgConfigs.ListStreams(ctx, orgSlug)
	if err != nil {
		return nil, err
	}
	var startable, rest []StreamDiscoveryItem
	for _, streamKey := range streams {
		scope := org.Scope{Org: orgSlug, Stream: streamKey}
		cfg, loadErr := s.loadOrgActiveConfig(ctx, scope)
		if loadErr != nil {
			if errors.Is(loadErr, org.ErrNotFound) {
				continue
			}
			return nil, loadErr
		}
		if cfg.Stream == nil {
			continue
		}
		key := orgCatalogKey(streamKey)
		canStart := orgStreamIsStartable(user, cfg)
		item := StreamDiscoveryItem{
			WorkflowKey:     key,
			WorkflowName:    strings.TrimSpace(cfg.Stream.Title),
			Description:     normalizeOptionalText(cfg.Stream.Description),
			CategorySlug:    strings.TrimSpace(cfg.Stream.CategorySlug),
			SubCategorySlug: strings.TrimSpace(cfg.Stream.SubCategorySlug),
			Href:            streamPath(key) + "/",
			Startable:       canStart,
		}
		if canStart {
			item.StartAction = streamPath(key) + "/instance/start"
			startable = append(startable, item)
			continue
		}
		rest = append(rest, item)
	}
	return append(startable, rest...), nil
}

func (s *Server) buildOrgPublicStreamCardView(ctx context.Context, user *AccountUser, key string, cfg RuntimeConfig) (PublicStreamCardView, error) {
	instanceCount := 0
	activeCount := 0
	if user != nil && s.orgConfigs != nil && s.orgDomains != nil {
		scope := org.Scope{Org: strings.TrimSpace(user.OrgSlug), Stream: cfg.OrgStreamKey}
		orgCfg, err := s.loadOrgActiveConfig(ctx, scope)
		if err == nil {
			domain, domainErr := s.orgDomains.Get(ctx, scope)
			if domainErr != nil && !errors.Is(domainErr, org.ErrNotFound) {
				return PublicStreamCardView{}, domainErr
			}
			if errors.Is(domainErr, org.ErrNotFound) {
				domain = org.EmptyDomain()
			}
			op := org.OperatorIdentity{
				Org:   scope.Org,
				Roles: append([]string(nil), user.RoleSlugs...),
			}
			summaries := org.ListStreamInstances(orgCfg, domain, op)
			instanceCount = len(summaries)
			for _, sum := range summaries {
				if sum.Status == org.StreamInstanceActive {
					activeCount++
				}
			}
		}
	}
	stepCount := 0
	if cfg.OrgStreamKey != "" && user != nil {
		if orgCfg, err := s.loadOrgActiveConfig(ctx, org.Scope{Org: strings.TrimSpace(user.OrgSlug), Stream: cfg.OrgStreamKey}); err == nil && orgCfg.Stream != nil {
			stepCount = len(orgCfg.Stream.Steps)
		}
	}
	return PublicStreamCardView{
		Name:            cfg.Workflow.Name,
		Description:     normalizeOptionalText(cfg.Workflow.Description),
		Href:            streamPath(key) + "/",
		InstanceCount:   instanceCount,
		ActiveCount:     activeCount,
		AllCompleted:    instanceCount > 0 && activeCount == 0,
		StepCount:       stepCount,
		OrganizationCount: len(cfg.Organizations),
		Organizations: publicStreamCardOrgViews(cfg.Organizations),
	}, nil
}

func publicStreamCardOrgViews(orgs []WorkflowOrganization) []PublicStreamCardOrgView {
	out := make([]PublicStreamCardOrgView, 0, len(orgs))
	for _, o := range orgs {
		name := strings.TrimSpace(o.Name)
		if name == "" {
			name = strings.TrimSpace(o.Slug)
		}
		if name == "" {
			continue
		}
		out = append(out, PublicStreamCardOrgView{
			Name:     name,
			Initials: organizationInitials(name),
		})
	}
	return out
}

func orgStreamInstanceCards(cfg org.OrgConfig, domain org.Domain, op org.OperatorIdentity, workflowKey string) []StreamInstanceCard {
	summaries := org.ListStreamInstances(cfg, domain, op)
	out := make([]StreamInstanceCard, 0, len(summaries))
	for _, sum := range summaries {
		done, total := 0, 0
		for _, step := range sum.Steps {
			done += step.Done
			total += step.Total
		}
		percent := 0
		if total > 0 {
			percent = int(float64(done) / float64(total) * 100)
		}
		createdAt, createdAtISO := "", ""
		var createdAtTime time.Time
		if ts := strings.TrimSpace(sum.CreatedAt); ts != "" {
			if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
				createdAtTime = parsed
				createdAt = humanReadableTraceabilityTime(parsed)
				createdAtISO = rfc3339UTC(parsed)
			} else {
				createdAt = ts
				createdAtISO = ts
			}
		}
		status := sum.Status
		if status == "" {
			status = processStatusActive
		}
		out = append(out, StreamInstanceCard{
			ID:            sum.ID,
			Name:          strings.TrimSpace(sum.Label),
			Status:        status,
			StatusLabel:   processStatusLabel(status),
			DetailHref:    streamInstancePath(workflowKey, sum.ID),
			CreatedAt:     createdAt,
			CreatedAtISO:  createdAtISO,
			CreatedAtTime: createdAtTime,
			DoneSubsteps:  done,
			TotalSubsteps: total,
			Percent:       percent,
		})
	}
	return out
}
