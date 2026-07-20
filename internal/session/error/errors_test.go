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
