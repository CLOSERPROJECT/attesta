package org

import "sync"

// DomainChangeNotifier hears that a Commit wrote a new Domain revision, after the
// Commit's side records (DPP, HTTP links) are stored.
type DomainChangeNotifier interface {
	DomainChanged(scope Scope, revision int64)
}

// DomainChangeFeed delivers Domain revisions for one Org stream to an open event stream.
type DomainChangeFeed interface {
	// Subscribe returns a channel holding at most the newest unseen revision.
	// cancel unsubscribes and closes the channel; calling it twice is safe.
	Subscribe(scope Scope) (revisions <-chan int64, cancel func())
}

// DomainChangeHub is the in-process DomainChangeNotifier and DomainChangeFeed.
// Another Attesta process never sees this hub's revisions; boards catch up on reconnect.
type DomainChangeHub struct {
	mu   sync.Mutex
	subs map[string]map[chan int64]struct{}
}

var (
	_ DomainChangeNotifier = (*DomainChangeHub)(nil)
	_ DomainChangeFeed     = (*DomainChangeHub)(nil)
)

func NewDomainChangeHub() *DomainChangeHub {
	return &DomainChangeHub{subs: map[string]map[chan int64]struct{}{}}
}

func (h *DomainChangeHub) Subscribe(scope Scope) (<-chan int64, func()) {
	normalized, err := scope.Normalize()
	if err != nil {
		closed := make(chan int64)
		close(closed)
		return closed, func() {}
	}
	key := scopeKey(normalized)
	ch := make(chan int64, 1)

	h.mu.Lock()
	if h.subs[key] == nil {
		h.subs[key] = map[chan int64]struct{}{}
	}
	h.subs[key][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			subs := h.subs[key]
			delete(subs, ch)
			if len(subs) == 0 {
				delete(h.subs, key)
			}
			close(ch)
		})
	}
	return ch, cancel
}

func (h *DomainChangeHub) DomainChanged(scope Scope, revision int64) {
	normalized, err := scope.Normalize()
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[scopeKey(normalized)] {
		rev := revision
		select {
		case ch <- rev:
		default:
			// Slot full: keep only the newest revision; boards re-Load on any bump.
			select {
			case old := <-ch:
				if old > rev {
					rev = old
				}
			default:
			}
			ch <- rev
		}
	}
}

func (h *DomainChangeHub) subscriberCount(scope Scope) int {
	normalized, err := scope.Normalize()
	if err != nil {
		return 0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs[scopeKey(normalized)])
}
