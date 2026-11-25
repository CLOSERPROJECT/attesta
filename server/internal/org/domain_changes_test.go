package org

import (
	"sync"
	"testing"
	"time"
)

func TestDomainChangeHubDeliversRevision(t *testing.T) {
	hub := NewDomainChangeHub()
	revisions, cancel := hub.Subscribe(FeragameScope())
	defer cancel()

	hub.DomainChanged(FeragameScope(), 7)

	select {
	case got := <-revisions:
		if got != 7 {
			t.Fatalf("revision = %d, want 7", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the published revision")
	}
}

func TestDomainChangeHubIsolatesScopes(t *testing.T) {
	hub := NewDomainChangeHub()
	recycling, cancelRecycling := hub.Subscribe(FeragameScope())
	defer cancelRecycling()
	other, cancelOther := hub.Subscribe(Scope{Org: "feragame", Stream: "composting"})
	defer cancelOther()

	hub.DomainChanged(FeragameScope(), 2)

	select {
	case got := <-recycling:
		if got != 2 {
			t.Fatalf("recycling revision = %d, want 2", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the recycling revision")
	}
	select {
	case got := <-other:
		t.Fatalf("other scope received revision %d, want nothing", got)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestDomainChangeHubKeepsNewestRevisionWhenSlotFull(t *testing.T) {
	tests := []struct {
		name     string
		publish  []int64
		expected int64
	}{
		{name: "in order", publish: []int64{1, 2, 3}, expected: 3},
		{name: "out of order", publish: []int64{5, 3, 4}, expected: 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hub := NewDomainChangeHub()
			revisions, cancel := hub.Subscribe(FeragameScope())
			defer cancel()

			for _, revision := range tc.publish {
				hub.DomainChanged(FeragameScope(), revision)
			}

			if got := <-revisions; got != tc.expected {
				t.Fatalf("revision = %d, want %d", got, tc.expected)
			}
			select {
			case got := <-revisions:
				t.Fatalf("second buffered revision = %d, want exactly one value", got)
			default:
			}
		})
	}
}

func TestDomainChangeHubNeverBlocksWithUnreadSubscriber(t *testing.T) {
	hub := NewDomainChangeHub()
	_, cancel := hub.Subscribe(FeragameScope())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for revision := int64(1); revision <= 1000; revision++ {
			hub.DomainChanged(FeragameScope(), revision)
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("DomainChanged blocked on an unread subscriber")
	}
}

func TestDomainChangeHubCancelTwiceIsSafe(t *testing.T) {
	hub := NewDomainChangeHub()
	revisions, cancel := hub.Subscribe(FeragameScope())
	if got := hub.subscriberCount(FeragameScope()); got != 1 {
		t.Fatalf("subscriberCount = %d, want 1", got)
	}

	cancel()
	cancel()

	if got := hub.subscriberCount(FeragameScope()); got != 0 {
		t.Fatalf("subscriberCount after cancel = %d, want 0", got)
	}
	if _, open := <-revisions; open {
		t.Fatal("channel stays open after cancel")
	}
}

func TestDomainChangeHubInvalidScopeIsIgnored(t *testing.T) {
	hub := NewDomainChangeHub()
	hub.DomainChanged(Scope{}, 9)
	if got := hub.subscriberCount(Scope{}); got != 0 {
		t.Fatalf("subscriberCount for invalid scope = %d, want 0", got)
	}
}

func TestDomainChangeHubInvalidScopeReturnsClosedChannel(t *testing.T) {
	hub := NewDomainChangeHub()
	revisions, cancel := hub.Subscribe(Scope{Org: "", Stream: "pv-recovery"})
	defer cancel()

	if _, open := <-revisions; open {
		t.Fatal("invalid scope channel is open, want closed")
	}
	if got := hub.subscriberCount(FeragameScope()); got != 0 {
		t.Fatalf("subscriberCount = %d, want 0", got)
	}
}

func TestDomainChangeHubConcurrentPublishAndCancel(t *testing.T) {
	hub := NewDomainChangeHub()
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, cancel := hub.Subscribe(FeragameScope())
			go func() {
				for range ch {
				}
			}()
			cancel()
		}()
	}
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		revision := int64(1)
		for {
			select {
			case <-stop:
				return
			default:
				hub.DomainChanged(FeragameScope(), revision)
				revision++
			}
		}
	}()
	time.Sleep(10 * time.Millisecond)
	close(stop)
	wg.Wait()
}
