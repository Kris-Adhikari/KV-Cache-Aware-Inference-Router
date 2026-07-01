package ring

import (
	"fmt"
	"testing"
)

func TestGetEmpty(t *testing.T) {
	r := New(10)
	if _, ok := r.Get("x"); ok {
		t.Fatal("empty ring returned ok=true")
	}
}

func TestGetDeterministic(t *testing.T) {
	r := New(50)
	r.Add("a")
	r.Add("b")
	r.Add("c")

	first, ok := r.Get("session-42")
	if !ok {
		t.Fatal("expected a member")
	}
	for range 100 {
		if got, _ := r.Get("session-42"); got != first {
			t.Fatalf("nondeterministic: %q vs %q", got, first)
		}
	}
}

func TestDistributionRoughlyEven(t *testing.T) {
	r := New(200)
	members := []string{"a", "b", "c", "d"}
	for _, m := range members {
		r.Add(m)
	}

	const n = 20000
	counts := map[string]int{}
	for i := range n {
		m, _ := r.Get(fmt.Sprintf("key-%d", i))
		counts[m]++
	}

	fair := n / len(members)
	for _, m := range members {
		if counts[m] < fair/2 || counts[m] > fair*3/2 {
			t.Errorf("member %q got %d keys, want within [%d,%d]", m, counts[m], fair/2, fair*3/2)
		}
	}
}

func TestRemoveMinimalDisruption(t *testing.T) {
	r := New(200)
	for _, m := range []string{"a", "b", "c", "d"} {
		r.Add(m)
	}

	const n = 20000
	before := make(map[string]string, n)
	for i := range n {
		k := fmt.Sprintf("key-%d", i)
		before[k], _ = r.Get(k)
	}

	r.Remove("c")

	moved := 0
	for k, prev := range before {
		now, ok := r.Get(k)
		if !ok {
			t.Fatal("ring unexpectedly empty")
		}
		if prev == "c" {
			if now == "c" {
				t.Errorf("key %q still routed to removed member c", k)
			}
			moved++
			continue
		}
		if now != prev {
			t.Errorf("key %q moved from %q to %q even though %q survived", k, prev, now, prev)
		}
	}
	if moved == 0 {
		t.Fatal("expected keys previously on c to move")
	}
}
