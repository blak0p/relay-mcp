package output

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBrokerSnapshotPreservesOutputAndSupportsContinuation(t *testing.T) {
	t.Parallel()
	broker := New(16)
	broker.Append([]byte("abcdef"))
	for _, tt := range []struct {
		name   string
		cursor int64
		want   Snapshot
	}{
		{"first snapshot is bounded", 0, snapshot("abc", 0, 3, 0, StatusRunning)},
		{"same cursor repeats immutable data", 0, snapshot("abc", 0, 3, 0, StatusRunning)},
		{"next cursor continues without replay", 3, snapshot("def", 3, 6, 0, StatusRunning)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assertSnapshot(t, read(t, broker, tt.cursor, 3, 0), tt.want)
		})
	}
}

func TestBroker_OverflowReportsExactGap(t *testing.T) {
	t.Parallel()
	broker := New(4)
	broker.Append([]byte("abcdefgh"))
	assertSnapshot(t, read(t, broker, 1, 4, 0), snapshot("efgh", 4, 8, 3, StatusRunning))
}

func TestBrokerSnapshotReadsAcrossRingWrap(t *testing.T) {
	t.Parallel()
	broker := New(5)
	broker.Append([]byte("abc"))
	broker.Append([]byte("def"))
	assertSnapshot(t, read(t, broker, 1, 5, 0), snapshot("bcdef", 1, 6, 0, StatusRunning))
}

func TestBrokerSnapshotWaitContracts(t *testing.T) {
	t.Parallel()
	t.Run("zero wait returns immediately", func(t *testing.T) {
		started := time.Now()
		assertSnapshot(t, read(t, New(16), 0, 16, 0), snapshot("", 0, 0, 0, StatusRunning))
		if elapsed := time.Since(started); elapsed > 50*time.Millisecond {
			t.Errorf("zero wait took %s, want less than 50ms", elapsed)
		}
	})
	for _, tt := range []struct {
		name string
		wait time.Duration
	}{{"default wait wakes for output", DefaultWait}, {"maximum wait wakes for output", MaxWait}} {
		t.Run(tt.name, func(t *testing.T) {
			broker, done := New(16), make(chan readResult, 1)
			go func() {
				got, err := broker.Snapshot(context.Background(), 0, 16, tt.wait)
				done <- readResult{got, err}
			}()
			select {
			case result := <-done:
				t.Fatalf("Snapshot() returned before output: %v", result.err)
			case <-time.After(20 * time.Millisecond):
			}
			broker.Append([]byte("ready"))
			select {
			case result := <-done:
				if result.err != nil {
					t.Fatal(result.err)
				}
				assertSnapshot(t, result.snapshot, snapshot("ready", 0, 5, 0, StatusRunning))
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Snapshot() did not wake after Append")
			}
		})
	}
}

func TestBrokerSnapshotWakesForTerminalStateAndContextCancellation(t *testing.T) {
	t.Parallel()
	t.Run("terminal status wakes empty read", func(t *testing.T) {
		broker, done := New(16), make(chan readResult, 1)
		go func() {
			got, err := broker.Snapshot(context.Background(), 0, 16, MaxWait)
			done <- readResult{got, err}
		}()
		time.Sleep(10 * time.Millisecond)
		broker.SetStatus(StatusExited)
		select {
		case result := <-done:
			if result.err != nil {
				t.Fatal(result.err)
			}
			assertSnapshot(t, result.snapshot, snapshot("", 0, 0, 0, StatusExited))
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Snapshot() did not wake after terminal status change")
		}
	})
	t.Run("context cancellation stops wait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := New(16).Snapshot(ctx, 0, 16, MaxWait); !errors.Is(err, context.Canceled) {
			t.Fatalf("Snapshot() error = %v, want context.Canceled", err)
		}
	})
}

func TestBrokerSnapshotReturnsImmutableOutput(t *testing.T) {
	t.Parallel()
	broker := New(16)
	broker.Append([]byte("output"))
	first := read(t, broker, 0, 16, 0)
	first.Output[0] = 'X'
	assertSnapshot(t, read(t, broker, 0, 16, 0), snapshot("output", 0, 6, 0, StatusRunning))
}

type readResult struct {
	snapshot Snapshot
	err      error
}

func read(t *testing.T, broker *Broker, cursor int64, maxBytes int, wait time.Duration) Snapshot {
	t.Helper()
	got, err := broker.Snapshot(context.Background(), cursor, maxBytes, wait)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func snapshot(output string, cursor, next, dropped int64, status Status) Snapshot {
	return Snapshot{[]byte(output), cursor, next, dropped, status}
}

func assertSnapshot(t *testing.T, got, want Snapshot) {
	t.Helper()
	if string(got.Output) != string(want.Output) || got.Cursor != want.Cursor || got.NextCursor != want.NextCursor || got.DroppedBytes != want.DroppedBytes || got.Status != want.Status {
		t.Errorf("Snapshot() = %+v, want %+v", got, want)
	}
}
