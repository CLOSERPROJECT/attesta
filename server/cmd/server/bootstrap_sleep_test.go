package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResolveBootstrapSleep(t *testing.T) {
	customErr := errors.New("custom sleep")
	customCalled := false
	custom := func(context.Context, time.Duration) error {
		customCalled = true
		return customErr
	}
	if err := resolveBootstrapSleep(custom)(context.Background(), time.Hour); !errors.Is(err, customErr) {
		t.Fatalf("custom sleep = %v", err)
	}
	if !customCalled {
		t.Fatal("custom sleep was not used")
	}

	sleep := resolveBootstrapSleep(nil)
	if err := sleep(context.Background(), 0); err != nil {
		t.Fatalf("zero duration = %v", err)
	}
	if err := sleep(context.Background(), time.Millisecond); err != nil {
		t.Fatalf("short duration = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleep(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context = %v", err)
	}
	if err := sleep(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled zero duration = %v", err)
	}
}
