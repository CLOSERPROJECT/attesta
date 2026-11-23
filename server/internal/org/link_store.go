package org

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// SubmissionAddress keys a Submission path (job / instance / task).
// LinkStore also requires Submission.Timestamp so many-submit Commits under the
// same address keep distinct link docs.
type SubmissionAddress struct {
	JobID         string
	JobInstanceID string
	TaskID        string
}

func (a SubmissionAddress) normalized() SubmissionAddress {
	return SubmissionAddress{
		JobID:         strings.TrimSpace(a.JobID),
		JobInstanceID: strings.TrimSpace(a.JobInstanceID),
		TaskID:        strings.TrimSpace(a.TaskID),
	}
}

func (a SubmissionAddress) valid() bool {
	a = a.normalized()
	return a.JobID != "" && a.JobInstanceID != "" && a.TaskID != ""
}

// LinkStore persists post-Commit HTTP links separately from Submission facts.
// Put/Get are keyed by Scope + Submission address + timestamp (no address-only fallback).
type LinkStore interface {
	Put(ctx context.Context, scope Scope, addr SubmissionAddress, timestamp string, links []HttpLinkItem) error
	Get(ctx context.Context, scope Scope, addr SubmissionAddress, timestamp string) ([]HttpLinkItem, error)
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

func linkStoreKey(scope Scope, addr SubmissionAddress, timestamp string) (string, error) {
	scope, err := scope.Normalize()
	if err != nil {
		return "", err
	}
	addr = addr.normalized()
	timestamp = strings.TrimSpace(timestamp)
	if !addr.valid() || timestamp == "" {
		return "", fmt.Errorf("org: link store key requires scope, submission address, and timestamp")
	}
	parts := []string{scope.Org, scope.Stream, addr.JobID, addr.JobInstanceID, addr.TaskID, timestamp}
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

func (s *MemoryLinkStore) Put(_ context.Context, scope Scope, addr SubmissionAddress, timestamp string, links []HttpLinkItem) error {
	key, err := linkStoreKey(scope, addr, timestamp)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byKey[key] = cloneHTTPLinks(links)
	return nil
}

func (s *MemoryLinkStore) Get(_ context.Context, scope Scope, addr SubmissionAddress, timestamp string) ([]HttpLinkItem, error) {
	key, err := linkStoreKey(scope, addr, timestamp)
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
