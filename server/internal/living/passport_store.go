package living

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// PassportStore persists issued Passports keyed by org + subject.
type PassportStore interface {
	Get(ctx context.Context, org, subjectID string) (PassportRecord, error)
	// GetByIdentity resolves a public Digital Link (gtin/lot/serial) to a locked Passport.
	GetByIdentity(ctx context.Context, gtin, lot, serial string) (PassportRecord, error)
	// Create inserts once; returns ErrPassportExists when the subject is already locked.
	Create(ctx context.Context, record PassportRecord) error
	// DeleteByOrg removes every Passport for org (seed --replace / Domain rebuild).
	DeleteByOrg(ctx context.Context, org string) error
}

var _ PassportStore = (*MemoryPassportStore)(nil)

// MemoryPassportStore is an in-memory PassportStore.
type MemoryPassportStore struct {
	mu     sync.RWMutex
	byKey  map[string]PassportRecord
}

func NewMemoryPassportStore() *MemoryPassportStore {
	return &MemoryPassportStore{byKey: map[string]PassportRecord{}}
}

func passportStoreKey(org, subjectID string) (string, error) {
	org = strings.TrimSpace(org)
	subjectID = strings.TrimSpace(subjectID)
	if org == "" || subjectID == "" {
		return "", fmt.Errorf("living: passport store key requires org and subject id")
	}
	return org + "\x00" + subjectID, nil
}

func (s *MemoryPassportStore) Get(_ context.Context, org, subjectID string) (PassportRecord, error) {
	key, err := passportStoreKey(org, subjectID)
	if err != nil {
		return PassportRecord{}, ErrNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.byKey[key]
	if !ok {
		return PassportRecord{}, ErrNotFound
	}
	return clonePassportRecord(rec), nil
}

func (s *MemoryPassportStore) GetByIdentity(_ context.Context, gtin, lot, serial string) (PassportRecord, error) {
	gtin = strings.TrimSpace(gtin)
	lot = strings.TrimSpace(lot)
	serial = strings.TrimSpace(serial)
	if gtin == "" || lot == "" || serial == "" {
		return PassportRecord{}, ErrNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, rec := range s.byKey {
		if rec.GTIN == gtin && rec.Lot == lot && rec.Serial == serial {
			return clonePassportRecord(rec), nil
		}
	}
	return PassportRecord{}, ErrNotFound
}

func (s *MemoryPassportStore) Create(_ context.Context, record PassportRecord) error {
	key, err := passportStoreKey(record.Org, record.SubjectID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byKey[key]; exists {
		return ErrPassportExists
	}
	record.Org = strings.TrimSpace(record.Org)
	record.SubjectID = strings.TrimSpace(record.SubjectID)
	s.byKey[key] = clonePassportRecord(record)
	return nil
}

func (s *MemoryPassportStore) DeleteByOrg(_ context.Context, org string) error {
	org = strings.TrimSpace(org)
	if org == "" {
		return fmt.Errorf("living: passport delete by org requires org")
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
