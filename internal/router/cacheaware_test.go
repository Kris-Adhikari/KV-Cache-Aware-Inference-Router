package router

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func abcd() []*Backend {
	return []*Backend{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}}
}

func TestCacheAwareEmpty(t *testing.T) {
	c := NewCacheAware(50, nil)
	if got := c.Route("k"); got != nil {
		t.Errorf("empty: got %v, want nil", got)
	}
}

func TestCacheAwareStableForSameKey(t *testing.T) {
	c := NewCacheAware(50, abcd())
	first := c.Route("session-1")
	if first == nil {
		t.Fatal("got nil")
	}
	for range 20 {
		if got := c.Route("session-1"); got != first {
			t.Fatalf("unstable: got %v, want %v", got, first)
		}
	}
}

func TestCacheAwareFollowUpSticksToWarmPod(t *testing.T) {
	c := NewCacheAware(50, abcd())
	base := c.Route("conv")
	if base == nil {
		t.Fatal("got nil")
	}
	// A follow-up that extends the same prefix must hit the warm pod.
	if got := c.Route("conv|assistant:hi|user:more"); got != base {
		t.Errorf("follow-up got %v, want warm pod %v", got, base)
	}
}

func TestCacheAwareFallbackSpreads(t *testing.T) {
	c := NewCacheAware(100, abcd())
	seen := map[string]bool{}
	for i := range 200 {
		seen[c.Route(fmt.Sprintf("key-%d", i)).Name] = true
	}
	if len(seen) < 2 {
		t.Errorf("fallback used %d pods, want >= 2", len(seen))
	}
}

func TestCacheAwareReroutesAfterRemove(t *testing.T) {
	c := NewCacheAware(50, []*Backend{{Name: "a"}, {Name: "b"}})
	first := c.Route("x")
	if first == nil {
		t.Fatal("got nil")
	}
	c.Remove(first.Name)

	second := c.Route("x")
	if second == nil {
		t.Fatal("got nil after remove")
	}
	if second.Name == first.Name {
		t.Errorf("still routed to removed pod %q", first.Name)
	}
}

func TestCacheAwareConcurrent(t *testing.T) {
	c := NewCacheAware(50, abcd())
	var nils atomic.Int64
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for i := range 500 {
				if c.Route(fmt.Sprintf("key-%d", i)) == nil {
					nils.Add(1)
				}
			}
		})
	}
	wg.Wait()
	if n := nils.Load(); n != 0 {
		t.Errorf("got %d nil routes with backends present", n)
	}
}
