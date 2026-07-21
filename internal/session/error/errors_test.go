package serror

import (
	"errors"
	"strings"
	"testing"
)

func TestSentinelErrors_Defined(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"ErrSessionAlreadyExists", ErrSessionAlreadyExists, "a session is already active"},
		{"ErrBashNotFound", ErrBashNotFound, "bash not found"},
		{"ErrSpawnFailed", ErrSpawnFailed, "failed to spawn bash"},
		{"ErrSessionNotFound", ErrSessionNotFound, "session not found"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.err == nil {
				t.Fatal("sentinel error is nil")
			}
			if !errors.Is(tt.err, tt.err) {
				t.Fatalf("errors.Is(%s, itself) = false, want true", tt.name)
			}
			if !strings.Contains(tt.err.Error(), tt.want) {
				t.Fatalf("%s.Error() = %q, want substring %q", tt.name, tt.err.Error(), tt.want)
			}
		})
	}
}

func TestSentinelErrors_Distinct(t *testing.T) {
	t.Parallel()
	all := []error{ErrSessionAlreadyExists, ErrBashNotFound, ErrSpawnFailed, ErrSessionNotFound}
	for i, a := range all {
		for j, b := range all {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Fatalf("errors.Is(sentinel[%d], sentinel[%d]) = true, want false", i, j)
			}
		}
	}
}

// TestWriteSentinels_Defined asserts the three write_terminal sentinels
// exist, are non-nil, satisfy errors.Is(self), and carry the expected
// substring in their message (REQ-WT-002, REQ-WT-003, REQ-WT-006).
func TestWriteSentinels_Defined(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"ErrSessionNotAlive", ErrSessionNotAlive, "not alive"},
		{"ErrWriteTooLarge", ErrWriteTooLarge, "maximum size"},
		{"ErrSessionClosed", ErrSessionClosed, "closed"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.err == nil {
				t.Fatalf("sentinel error %s is nil", tt.name)
			}
			if !errors.Is(tt.err, tt.err) {
				t.Fatalf("errors.Is(%s, itself) = false, want true", tt.name)
			}
			if !strings.Contains(tt.err.Error(), tt.want) {
				t.Fatalf("%s.Error() = %q, want substring %q", tt.name, tt.err.Error(), tt.want)
			}
		})
	}
}

// TestWriteSentinels_Distinct asserts none of the three new write sentinels
// collides (via errors.Is) with any existing sentinel or with each other.
func TestWriteSentinels_Distinct(t *testing.T) {
	t.Parallel()
	existing := []error{ErrSessionAlreadyExists, ErrBashNotFound, ErrSpawnFailed, ErrSessionNotFound}
	newSentinels := []error{ErrSessionNotAlive, ErrWriteTooLarge, ErrSessionClosed}
	for _, n := range newSentinels {
		for _, e := range existing {
			if errors.Is(n, e) {
				t.Fatalf("errors.Is(new sentinel %q, existing sentinel %q) = true, want false", n, e)
			}
		}
	}
	for i, a := range newSentinels {
		for j, b := range newSentinels {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Fatalf("errors.Is(new[%d], new[%d]) = true, want false", i, j)
			}
		}
	}
}
