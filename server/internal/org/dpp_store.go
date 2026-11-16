package org

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// DPPStore persists issued DPPs keyed by org + subject.
type DPPStore interface {
	Get(ctx context.Context, org, subjectID string) (DPPRecord, error)
	// GetByIdentity resolves a public Digital Link (gtin/lot/serial) to a locked Passport.
	GetByIdentity(ctx context.Context, gtin, lot, serial string) (DPPRecord, error)
	// Create inserts once; returns ErrDPPExists when the subject is already locked.
	Create(ctx context.Context, record DPPRecord) error
	// DeleteByOrg removes every Passport for org (seed --replace / Domain rebuild).
	DeleteByOrg(ctx context.Context, org string) error
}

var _ DPPStore = (*MemoryDPPStore)(nil)

// MemoryDPPStore is an in-memory DPPStore.
type MemoryDPPStore struct {
	mu     sync.RWMutex
	byKey  map[string]DPPRecord
}

func NewMemoryDPPStore() *MemoryDPPStore {
	return &MemoryDPPStore{byKey: map[string]DPPRecord{}}
}

func passportStoreKey(org, subjectID string) (string, error) {
	org = strings.TrimSpace(org)
	subjectID = strings.TrimSpace(subjectID)
	if org == "" || subjectID == "" {
		return "", fmt.Errorf("org: passport store key requires org and subject id")
	}
	return org + "\x00" + subjectID, nil
}

func (s *MemoryDPPStore) Get(_ context.Context, org, subjectID string) (DPPRecord, error) {
	key, err := passportStoreKey(org, subjectID)
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
	key, err := passportStoreKey(record.Org, record.SubjectID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byKey[key]; exists {
		return ErrDPPExists
	}
	record.Org = strings.TrimSpace(record.Org)
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
