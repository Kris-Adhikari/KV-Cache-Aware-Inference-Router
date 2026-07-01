package main

import (
	"testing"
	"time"
)

func TestSummarize(t *testing.T) {
	var rs []result
	for i := 1; i <= 100; i++ {
		rs = append(rs, result{latency: time.Duration(i) * time.Millisecond, hit: i%2 == 0, ok: true})
	}
	s := summarize(rs)
	if s.count != 100 {
		t.Errorf("count = %d, want 100", s.count)
	}
	if s.hits != 50 {
		t.Errorf("hits = %d, want 50", s.hits)
	}
	if s.avg != 50500*time.Microsecond {
		t.Errorf("avg = %v, want 50.5ms", s.avg)
	}
	if s.p50 != 51*time.Millisecond {
		t.Errorf("p50 = %v, want 51ms", s.p50)
	}
	if s.p95 != 96*time.Millisecond {
		t.Errorf("p95 = %v, want 96ms", s.p95)
	}
}

func TestSummarizeErrors(t *testing.T) {
	rs := []result{{ok: true}, {ok: false}, {ok: false}}
	if s := summarize(rs); s.errors != 2 {
		t.Errorf("errors = %d, want 2", s.errors)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	if s := summarize(nil); s.count != 0 || s.avg != 0 {
		t.Errorf("empty summarize = %+v", s)
	}
}

func TestPercentileEmpty(t *testing.T) {
	if got := percentile(nil, 50); got != 0 {
		t.Errorf("percentile(nil) = %v, want 0", got)
	}
}
