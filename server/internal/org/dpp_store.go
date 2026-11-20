package org

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// DPPStore persists issued DPPs keyed by (org, stream, subject).
type DPPStore interface {
	Get(ctx context.Context, scope Scope, subjectID string) (DPPRecord, error)
	// GetByIdentity resolves a public Digital Link (gtin/lot/serial) to a locked Passport.
	// Identity lookup is global across streams so the shared Digital Link stays public.
	GetByIdentity(ctx context.Context, gtin, lot, serial string) (DPPRecord, error)
	// Create inserts once; returns ErrDPPExists when the subject is already locked in
	// this stream, or when the identity is already issued in any stream (fail closed).
	Create(ctx context.Context, record DPPRecord) error
	// DeleteByOrg removes every Passport for org (seed --replace / Domain rebuild).
	DeleteByOrg(ctx context.Context, org string) error
}

var _ DPPStore = (*MemoryDPPStore)(nil)

// MemoryDPPStore is an in-memory DPPStore.
type MemoryDPPStore struct {
	mu    sync.RWMutex
	byKey map[string]DPPRecord
}

func NewMemoryDPPStore() *MemoryDPPStore {
	return &MemoryDPPStore{byKey: map[string]DPPRecord{}}
}

func passportStoreKey(scope Scope, subjectID string) (string, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return "", err
	}
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return "", fmt.Errorf("org: passport store key requires subject id")
	}
	return scope.Org + "\x00" + scope.Stream + "\x00" + subjectID, nil
}

func (s *MemoryDPPStore) Get(_ context.Context, scope Scope, subjectID string) (DPPRecord, error) {
	key, err := passportStoreKey(scope, subjectID)
	if err != nil {
		return DPPRecord{}, ErrNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.byKey[key]
	if !ok {
		return DPPRecord{}, ErrNotFound
	}
	return clonePassportRecord(rec), nil
}

func (s *MemoryDPPStore) GetByIdentity(_ context.Context, gtin, lot, serial string) (DPPRecord, error) {
	gtin = strings.TrimSpace(gtin)
	lot = strings.TrimSpace(lot)
	serial = strings.TrimSpace(serial)
	if gtin == "" || lot == "" || serial == "" {
		return DPPRecord{}, ErrNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, rec := range s.byKey {
		if rec.GTIN == gtin && rec.Lot == lot && rec.Serial == serial {
			return clonePassportRecord(rec), nil
		}
	}
	return DPPRecord{}, ErrNotFound
}

func (s *MemoryDPPStore) Create(_ context.Context, record DPPRecord) error {
	scope, err := Scope{Org: record.Org, Stream: record.Stream}.Normalize()
	if err != nil {
		return err
	}
	key, err := passportStoreKey(scope, record.SubjectID)
	if err != nil {
		return err
	}
	gtin := strings.TrimSpace(record.GTIN)
	lot := strings.TrimSpace(record.Lot)
	serial := strings.TrimSpace(record.Serial)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byKey[key]; exists {
		return ErrDPPExists
	}
	if gtin != "" && lot != "" && serial != "" {
		for _, rec := range s.byKey {
			if rec.GTIN == gtin && rec.Lot == lot && rec.Serial == serial {
				return ErrDPPExists
			}
		}
	}
	record.Org = scope.Org
	record.Stream = scope.Stream
	record.SubjectID = strings.TrimSpace(record.SubjectID)
	s.byKey[key] = clonePassportRecord(record)
	return nil
}

func (s *MemoryDPPStore) DeleteByOrg(_ context.Context, org string) error {
	org = strings.TrimSpace(org)
	if org == "" {
		return fmt.Errorf("org: passport delete by org requires org")
	}
	prefix := org + "\x00"
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.byKey {
		if strings.HasPrefix(key, prefix) {
			delete(s.byKey, key)
		}
	}
	return nil
}
