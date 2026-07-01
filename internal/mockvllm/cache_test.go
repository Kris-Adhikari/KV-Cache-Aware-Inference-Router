package mockvllm

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

var testParams = Params{Base: 10 * time.Millisecond, Prefill: 2 * time.Millisecond}

func TestColdPromptPaysFullPrefill(t *testing.T) {
	c := NewCache()
	got := c.Process("a b c", testParams)

	want := Result{
		TTFT:           10*time.Millisecond + 3*2*time.Millisecond, // 16ms
		TotalTokens:    3,
		CachedTokens:   0,
		UncachedTokens: 3,
		Hit:            false,
	}
	if got != want {
		t.Errorf("cold: got %+v, want %+v", got, want)
	}
}

func TestRepeatPromptSkipsPrefill(t *testing.T) {
	c := NewCache()
	c.Process("a b c", testParams)
	got := c.Process("a b c", testParams)

	if !got.Hit || got.CachedTokens != 3 || got.UncachedTokens != 0 {
		t.Fatalf("repeat: got %+v", got)
	}
	if got.TTFT != testParams.Base {
		t.Errorf("repeat TTFT = %v, want base %v", got.TTFT, testParams.Base)
	}
}

func TestFollowUpPaysOnlyForNewTokens(t *testing.T) {
	c := NewCache()
	c.Process("a b c", testParams)
	got := c.Process("a b c d e", testParams)

	if got.CachedTokens != 3 || got.UncachedTokens != 2 {
		t.Fatalf("follow-up: got %+v", got)
	}
	want := testParams.Base + 2*testParams.Prefill // 14ms
	if got.TTFT != want {
		t.Errorf("follow-up TTFT = %v, want %v", got.TTFT, want)
	}
}

func TestDivergentPromptIsCold(t *testing.T) {
	c := NewCache()
	c.Process("a b c", testParams)
	got := c.Process("x y", testParams)

	if got.Hit || got.CachedTokens != 0 || got.UncachedTokens != 2 {
		t.Errorf("divergent: got %+v", got)
	}
}

func TestTokenize(t *testing.T) {
	got := tokenize("  hello   world ")
	if len(got) != 2 || got[0] != "hello" || got[1] != "world" {
		t.Errorf("tokenize = %q", got)
	}
}

func TestProcessConcurrent(t *testing.T) {
	c := NewCache()
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 200 {
				c.Process(fmt.Sprintf("g%d tok%d", g, i), testParams)
			}
		})
	}
	wg.Wait()
}
