package router

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestRoundRobinCyclesInOrder(t *testing.T) {
	backends := []*Backend{
		{Name: "pod-a", URL: "http://a"},
		{Name: "pod-b", URL: "http://b"},
		{Name: "pod-c", URL: "http://c"},
	}
	rr := NewRoundRobin(backends)

	// Two full cycles must return a, b, c, a, b, c in order.
	want := []string{"pod-a", "pod-b", "pod-c", "pod-a", "pod-b", "pod-c"}
	for i, name := range want {
		got := rr.Route("ignored-key")
		if got == nil {
			t.Fatalf("call %d: got nil backend", i)
		}
		if got.Name != name {
			t.Errorf("call %d: got %q, want %q", i, got.Name, name)
		}
	}
}

func TestRoundRobinEmpty(t *testing.T) {
	rr := NewRoundRobin(nil)
	if got := rr.Route("k"); got != nil {
		t.Errorf("empty round-robin: got %v, want nil", got)
	}
}

func TestRoundRobinConcurrentEvenSpread(t *testing.T) {
	backends := []*Backend{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}}
	rr := NewRoundRobin(backends)

	const goroutines = 8
	const perGoroutine = 1000
	total := goroutines * perGoroutine

	counts := make([]atomic.Int64, len(backends))
	index := make(map[*Backend]int, len(backends))
	for i, b := range backends {
		index[b] = i
	}

	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for range perGoroutine {
				counts[index[rr.Route("")]].Add(1)
			}
		})
	}
	wg.Wait()

	want := int64(total / len(backends))
	for i := range counts {
		if got := counts[i].Load(); got != want {
			t.Errorf("backend %d: got %d picks, want %d", i, got, want)
		}
	}
}
