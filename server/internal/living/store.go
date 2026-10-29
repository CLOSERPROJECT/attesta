package living

import (
	"context"
	"fmt"
	"sync"
)

// DomainStore persists the org Domain document.
type DomainStore interface {
	Get(ctx context.Context, org string) (Domain, error)
	Put(ctx context.Context, org string, domain Domain) error
}

// ConfigStore persists versioned Org Config documents and the Active Config version pointer.
type ConfigStore interface {
	Get(ctx context.Context, org, version string) (OrgConfig, error)
	Put(ctx context.Context, org, version string, cfg OrgConfig) error
	ActiveVersion(ctx context.Context, org string) (string, error)
	SetActiveVersion(ctx context.Context, org, version string) error
}

// ErrNotFound is returned when a Domain or Org Config document is missing.
var ErrNotFound = fmt.Errorf("living: not found")

// MemoryDomainStore is an in-memory DomainStore.
type MemoryDomainStore struct {
	mu   sync.RWMutex
	byOrg map[string]Domain
}

func NewMemoryDomainStore() *MemoryDomainStore {
	return &MemoryDomainStore{byOrg: map[string]Domain{}}
}

func (s *MemoryDomainStore) Get(_ context.Context, org string) (Domain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	domain, ok := s.byOrg[org]
	if !ok {
		return EmptyDomain(), ErrNotFound
	}
	return cloneDomain(domain), nil
}

func (s *MemoryDomainStore) Put(_ context.Context, org string, domain Domain) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byOrg[org] = cloneDomain(domain)
	return nil
}

// MemoryConfigStore is an in-memory ConfigStore.
type MemoryConfigStore struct {
	mu       sync.RWMutex
	byOrgVer map[string]OrgConfig // key: org + "\x00" + version
	active   map[string]string    // org → active version
}

func NewMemoryConfigStore() *MemoryConfigStore {
	return &MemoryConfigStore{
		byOrgVer: map[string]OrgConfig{},
		active:   map[string]string{},
	}
}

func configKey(org, version string) string {
	return org + "\x00" + version
}

func (s *MemoryConfigStore) Get(_ context.Context, org, version string) (OrgConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.byOrgVer[configKey(org, version)]
	if !ok {
		return OrgConfig{}, ErrNotFound
	}
	return cfg, nil
}

func (s *MemoryConfigStore) Put(_ context.Context, org, version string, cfg OrgConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg.Org = org
	cfg.Version = version
	s.byOrgVer[configKey(org, version)] = cfg
	return nil
}

func (s *MemoryConfigStore) ActiveVersion(_ context.Context, org string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	version, ok := s.active[org]
	if !ok || version == "" {
		return "", ErrNotFound
	}
	return version, nil
}

func (s *MemoryConfigStore) SetActiveVersion(_ context.Context, org, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byOrgVer[configKey(org, version)]; !ok {
		return ErrNotFound
	}
	s.active[org] = version
	return nil
}

func cloneDomain(d Domain) Domain {
	out := Domain{
		Load:         cloneEntityMap(d.Load),
		Pallet:       cloneEntityMap(d.Pallet),
		Container:    cloneEntityMap(d.Container),
		Submissions:  append([]Submission(nil), d.Submissions...),
		JobInstances: append([]JobInstance(nil), d.JobInstances...),
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
