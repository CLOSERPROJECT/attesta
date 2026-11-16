package living

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// SubmissionAddress keys a Submission path (job / instance / task [+ group]).
// LinkStore also requires Submission.Timestamp so many-submit Commits under the
// same address keep distinct link docs.
type SubmissionAddress struct {
	JobID           string
	JobInstanceID   string
	TaskID          string
	GroupID         string
	GroupInstanceID string
}

func (a SubmissionAddress) normalized() SubmissionAddress {
	return SubmissionAddress{
		JobID:           strings.TrimSpace(a.JobID),
		JobInstanceID:   strings.TrimSpace(a.JobInstanceID),
		TaskID:          strings.TrimSpace(a.TaskID),
		GroupID:         strings.TrimSpace(a.GroupID),
		GroupInstanceID: strings.TrimSpace(a.GroupInstanceID),
	}
}

func (a SubmissionAddress) valid() bool {
	a = a.normalized()
	if a.JobID == "" || a.JobInstanceID == "" || a.TaskID == "" {
		return false
	}
	if (a.GroupID == "") != (a.GroupInstanceID == "") {
		return false
	}
	return true
}

// LinkStore persists post-Commit HTTP links separately from Submission facts.
// Put/Get are keyed by Submission address + timestamp (no address-only fallback).
type LinkStore interface {
	Put(ctx context.Context, org string, addr SubmissionAddress, timestamp string, links []HttpLinkItem) error
	Get(ctx context.Context, org string, addr SubmissionAddress, timestamp string) ([]HttpLinkItem, error)
}

var _ LinkStore = (*MemoryLinkStore)(nil)

// MemoryLinkStore is an in-memory LinkStore.
type MemoryLinkStore struct {
	mu    sync.RWMutex
	byKey map[string][]HttpLinkItem
}

func NewMemoryLinkStore() *MemoryLinkStore {
	return &MemoryLinkStore{byKey: map[string][]HttpLinkItem{}}
}

func linkStoreKey(org string, addr SubmissionAddress, timestamp string) (string, error) {
	org = strings.TrimSpace(org)
	addr = addr.normalized()
	timestamp = strings.TrimSpace(timestamp)
	if org == "" || !addr.valid() || timestamp == "" {
		return "", fmt.Errorf("living: link store key requires org, submission address, and timestamp")
	}
	parts := []string{org, addr.JobID, addr.JobInstanceID, addr.TaskID}
	if addr.GroupID != "" {
		parts = append(parts, addr.GroupID, addr.GroupInstanceID)
	}
	parts = append(parts, timestamp)
	return strings.Join(parts, "\x00"), nil
}

func cloneHTTPLinks(in []HttpLinkItem) []HttpLinkItem {
	if in == nil {
		return nil
	}
	out := make([]HttpLinkItem, len(in))
	copy(out, in)
	return out
}

func (s *MemoryLinkStore) Put(_ context.Context, org string, addr SubmissionAddress, timestamp string, links []HttpLinkItem) error {
	key, err := linkStoreKey(org, addr, timestamp)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byKey[key] = cloneHTTPLinks(links)
	return nil
}

func (s *MemoryLinkStore) Get(_ context.Context, org string, addr SubmissionAddress, timestamp string) ([]HttpLinkItem, error) {
	key, err := linkStoreKey(org, addr, timestamp)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	links, ok := s.byKey[key]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneHTTPLinks(links), nil
}
