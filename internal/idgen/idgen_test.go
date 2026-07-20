package idgen

import (
	"regexp"
	"sync"
	"testing"
)

// idFormat matches the spec: "term_" prefix + 16 lowercase hex chars.
var idFormat = regexp.MustCompile(`^term_[0-9a-f]{16}$`)

func TestNew_Format(t *testing.T) {
	t.Parallel()
	id := New()
	if !idFormat.MatchString(id) {
		t.Fatalf("New() = %q, want match %s", id, idFormat.String())
	}
}

func TestNew_NonEmptyAndPrefixed(t *testing.T) {
	t.Parallel()
	id := New()
	if id == "" {
		t.Fatal("New() returned empty string")
	}
	if len(id) < len("term_") {
		t.Fatalf("New() = %q too short", id)
	}
	if id[:len("term_")] != "term_" {
		t.Fatalf("New() = %q, want term_ prefix", id)
	}
}

func TestNew_UniquenessParallel(t *testing.T) {
	t.Parallel()
	const n = 1000
	ids := make(chan string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids <- New()
		}()
	}
	wg.Wait()
	close(ids)

	seen := make(map[string]struct{}, n)
	count := 0
	for id := range ids {
		count++
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id generated: %q", id)
		}
		seen[id] = struct{}{}
		if !idFormat.MatchString(id) {
			t.Fatalf("id %q does not match format", id)
		}
	}
	if count != n {
		t.Fatalf("got %d ids, want %d", count, n)
	}
}