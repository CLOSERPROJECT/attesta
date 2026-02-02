package main

import (
	"testing"
	"time"
)

func TestSSEHubSubscribeUnsubscribeClosesChannel(t *testing.T) {
	hub := newSSEHub()
	ch := hub.Subscribe("p1")

	hub.Unsubscribe("p1", ch)

	if _, ok := <-ch; ok {
		t.Fatal("expected unsubscribed channel to be closed")
	}
}

func TestSSEHubBroadcastDeliversMessage(t *testing.T) {
	hub := newSSEHub()
	ch1 := hub.Subscribe("p1")
	ch2 := hub.Subscribe("p1")
	t.Cleanup(func() {
		hub.Unsubscribe("p1", ch1)
		hub.Unsubscribe("p1", ch2)
	})

	hub.Broadcast("p1", "process-updated")

	if got := <-ch1; got != "process-updated" {
		t.Fatalf("expected message on subscriber 1, got %q", got)
	}
	if got := <-ch2; got != "process-updated" {
		t.Fatalf("expected message on subscriber 2, got %q", got)
	}
}

func TestSSEHubBroadcastDoesNotBlockOnFullBuffer(t *testing.T) {
	hub := newSSEHub()
	ch := hub.Subscribe("p1")
	t.Cleanup(func() { hub.Unsubscribe("p1", ch) })

	for i := 0; i < cap(ch); i++ {
		hub.Broadcast("p1", "msg")
	}

	done := make(chan struct{})
	go func() {
		hub.Broadcast("p1", "overflow")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected broadcast to drop overflow message without blocking")
	}
}
