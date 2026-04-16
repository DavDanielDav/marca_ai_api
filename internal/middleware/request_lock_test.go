package middleware

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAcquireEmailRequestSlotSerializesSameEmail(t *testing.T) {
	resetRequestGates()

	releaseFirst, err := AcquireEmailRequestSlot(context.Background(), "Pessoa@Test.com")
	if err != nil {
		t.Fatalf("unexpected error acquiring first slot: %v", err)
	}

	acquiredSecond := make(chan func(), 1)
	errCh := make(chan error, 1)

	go func() {
		releaseSecond, err := AcquireEmailRequestSlot(context.Background(), " pessoa@test.com ")
		if err != nil {
			errCh <- err
			return
		}

		acquiredSecond <- releaseSecond
	}()

	select {
	case err := <-errCh:
		t.Fatalf("unexpected error while waiting for second slot: %v", err)
	case releaseSecond := <-acquiredSecond:
		releaseSecond()
		t.Fatal("second request should wait until the first one finishes")
	case <-time.After(75 * time.Millisecond):
	}

	releaseFirst()

	select {
	case err := <-errCh:
		t.Fatalf("unexpected error after releasing first slot: %v", err)
	case releaseSecond := <-acquiredSecond:
		releaseSecond()
	case <-time.After(time.Second):
		t.Fatal("second request did not acquire the slot after the first one finished")
	}

	assertNoRequestGates(t)
}

func TestAcquireEmailRequestSlotHonorsContextCancellation(t *testing.T) {
	resetRequestGates()

	releaseFirst, err := AcquireEmailRequestSlot(context.Background(), "cancel@test.com")
	if err != nil {
		t.Fatalf("unexpected error acquiring first slot: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = AcquireEmailRequestSlot(ctx, "cancel@test.com")
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}

	releaseFirst()

	assertNoRequestGates(t)
}

func resetRequestGates() {
	requestGates.mu.Lock()
	defer requestGates.mu.Unlock()

	requestGates.byKey = make(map[string]*requestGate)
}

func assertNoRequestGates(t *testing.T) {
	t.Helper()

	requestGates.mu.Lock()
	defer requestGates.mu.Unlock()

	if len(requestGates.byKey) != 0 {
		t.Fatalf("expected request gates to be cleaned up, found %d", len(requestGates.byKey))
	}
}
