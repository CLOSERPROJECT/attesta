package org

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Scope identifies one stream of one org for Config and Domain persistence.
type Scope struct {
	Org    string
	Stream string
}

// Normalize trims Org and Stream; both are required.
func (s Scope) Normalize() (Scope, error) {
	s.Org = strings.TrimSpace(s.Org)
	s.Stream = strings.TrimSpace(s.Stream)
	if s.Org == "" || s.Stream == "" {
		return Scope{}, fmt.Errorf("org: scope requires org and stream")
	}
	return s, nil
}

// ScopeFromConfig builds a Scope from an Org Config's org and stream.key.
func ScopeFromConfig(cfg OrgConfig) (Scope, error) {
	if cfg.Stream == nil {
		return Scope{}, fmt.Errorf("org: config missing stream")
	}
	return Scope{Org: cfg.Org, Stream: cfg.Stream.Key}.Normalize()
}

// ErrAmbiguousStreams is returned when RequireSoleStream finds more than one stream.
var ErrAmbiguousStreams = fmt.Errorf("org: expected exactly one stream")

// RequireSoleStream returns the org's only stream Scope.
// Temporary until #309 scopes the Org API to an explicit stream.
func RequireSoleStream(ctx context.Context, configs ConfigStore, org string) (Scope, error) {
	org = strings.TrimSpace(org)
	if org == "" {
		return Scope{}, fmt.Errorf("org: org required")
	}
	if configs == nil {
		return Scope{}, fmt.Errorf("org: config store required")
	}
	streams, err := configs.ListStreams(ctx, org)
	if err != nil {
		return Scope{}, err
	}
	switch len(streams) {
	case 0:
		return Scope{}, ErrNotFound
	case 1:
		return Scope{Org: org, Stream: streams[0]}, nil
	default:
		return Scope{}, fmt.Errorf("%w for %q (got %d)", ErrAmbiguousStreams, org, len(streams))
	}
}

// DomainStore persists the stream Domain document.
type DomainStore interface {
	Get(ctx context.Context, scope Scope) (Domain, error)
	// Put overwrites unconditionally (seeds, wipes). Commit-path writes use Update.
	Put(ctx context.Context, scope Scope, domain Domain) error
	// Update runs fn on the current Domain (EmptyDomain when missing) and writes
	// its result only if no other write landed in between. fn may run more than
	// once and must not have side effects; its error is returned unchanged.
	// Returns ErrDomainConflict when retries are exhausted.
	Update(ctx context.Context, scope Scope, fn func(Domain) (Domain, error)) (Domain, error)
}

// ConfigStore persists versioned Org Config documents and the Active Config version pointer.
type ConfigStore interface {
	Get(ctx context.Context, scope Scope, version string) (OrgConfig, error)
	Put(ctx context.Context, scope Scope, version string, cfg OrgConfig) error
	ActiveVersion(ctx context.Context, scope Scope) (string, error)
	SetActiveVersion(ctx context.Context, scope Scope, version string) error
	// ListStreams returns stream keys that have an Active Config pointer for org.
	ListStreams(ctx context.Context, org string) ([]string, error)
}

// ErrNotFound is returned when a Domain or Org Config document is missing.
var ErrNotFound = fmt.Errorf("org: not found")

var ErrDomainConflict = fmt.Errorf("org: domain changed concurrently")

// MemoryDomainStore is an in-memory DomainStore.
type MemoryDomainStore struct {
	mu    sync.RWMutex
	byKey map[string]Domain
}

func NewMemoryDomainStore() *MemoryDomainStore {
	return &MemoryDomainStore{byKey: map[string]Domain{}}
}

func (s *MemoryDomainStore) Get(_ context.Context, scope Scope) (Domain, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return EmptyDomain(), ErrNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	domain, ok := s.byKey[scopeKey(scope)]
	if !ok {
		return EmptyDomain(), ErrNotFound
	}
	return cloneDomain(domain), nil
}

func (s *MemoryDomainStore) Put(_ context.Context, scope Scope, domain Domain) error {
	scope, err := scope.Normalize()
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byKey[scopeKey(scope)] = cloneDomain(domain)
	return nil
}

func (s *MemoryDomainStore) Update(_ context.Context, scope Scope, fn func(Domain) (Domain, error)) (Domain, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return Domain{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scopeKey(scope)
	current, ok := s.byKey[key]
	if !ok {
		current = EmptyDomain()
	}
	next, err := fn(cloneDomain(current))
	if err != nil {
		return Domain{}, err
	}
	s.byKey[key] = cloneDomain(next)
	return next, nil
}

// MemoryConfigStore is an in-memory ConfigStore.
type MemoryConfigStore struct {
	mu       sync.RWMutex
	byOrgVer map[string]OrgConfig // key: org + "\x00" + stream + "\x00" + version
	active   map[string]string    // org + "\x00" + stream → active version
}

func NewMemoryConfigStore() *MemoryConfigStore {
	return &MemoryConfigStore{
		byOrgVer: map[string]OrgConfig{},
		active:   map[string]string{},
	}
}

func scopeKey(scope Scope) string {
	return scope.Org + "\x00" + scope.Stream
}

func configKey(scope Scope, version string) string {
	return scopeKey(scope) + "\x00" + version
}

func (s *MemoryConfigStore) Get(_ context.Context, scope Scope, version string) (OrgConfig, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return OrgConfig{}, ErrNotFound
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return OrgConfig{}, ErrNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.byOrgVer[configKey(scope, version)]
	if !ok {
		return OrgConfig{}, ErrNotFound
	}
	return cfg, nil
}

func (s *MemoryConfigStore) Put(_ context.Context, scope Scope, version string, cfg OrgConfig) error {
	scope, err := scope.Normalize()
	if err != nil {
		return err
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("org: version required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg.Org = scope.Org
	cfg.Version = version
	if cfg.Stream == nil {
		cfg.Stream = &StreamConfig{Key: scope.Stream}
	} else {
		cfg.Stream.Key = scope.Stream
	}
	s.byOrgVer[configKey(scope, version)] = cfg
	return nil
}

func (s *MemoryConfigStore) ActiveVersion(_ context.Context, scope Scope) (string, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return "", ErrNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	version, ok := s.active[scopeKey(scope)]
	if !ok || version == "" {
		return "", ErrNotFound
	}
	return version, nil
}

func (s *MemoryConfigStore) SetActiveVersion(_ context.Context, scope Scope, version string) error {
	scope, err := scope.Normalize()
	if err != nil {
		return err
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("org: version required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byOrgVer[configKey(scope, version)]; !ok {
		return ErrNotFound
	}
	s.active[scopeKey(scope)] = version
	return nil
}

func (s *MemoryConfigStore) ListStreams(_ context.Context, org string) ([]string, error) {
	org = strings.TrimSpace(org)
	if org == "" {
		return nil, fmt.Errorf("org: org required")
	}
	prefix := org + "\x00"
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0)
	for key, version := range s.active {
		if version == "" || !strings.HasPrefix(key, prefix) {
			continue
		}
		stream := strings.TrimPrefix(key, prefix)
		if stream == "" || strings.Contains(stream, "\x00") {
			continue
		}
		out = append(out, stream)
	}
	sort.Strings(out)
	return out, nil
}

func cloneDomain(d Domain) Domain {
	ensureEntities(&d)
	out := Domain{
		Entities:     make(map[string]EntityMap, len(d.Entities)),
		Submissions:  append([]Submission(nil), d.Submissions...),
		JobInstances: append([]JobInstance(nil), d.JobInstances...),
	}
	for kind, m := range d.Entities {
		out.Entities[kind] = cloneEntityMap(m)
	}
	if out.Submissions == nil {
		out.Submissions = []Submission{}
	}
	if out.JobInstances == nil {
		out.JobInstances = []JobInstance{}
	}
	return out
}

func cloneEntityMap(m EntityMap) EntityMap {
	if m == nil {
		return EntityMap{}
	}
	out := make(EntityMap, len(m))
	for id, facts := range m {
		copied := make(map[string]any, len(facts))
		for k, v := range facts {
			copied[k] = v
		}
		out[id] = copied
	}
	return out
}
