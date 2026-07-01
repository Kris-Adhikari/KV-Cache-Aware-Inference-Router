package main

import "testing"

func TestParseBackends(t *testing.T) {
	got, err := parseBackends("a=http://x:1, b=http://y:2")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "a" || got[0].URL != "http://x:1" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[1].Name != "b" || got[1].URL != "http://y:2" {
		t.Errorf("got[1] = %+v", got[1])
	}
}

func TestParseBackendsEmpty(t *testing.T) {
	got, err := parseBackends("   ")
	if err != nil || got != nil {
		t.Errorf("got = %v, err = %v", got, err)
	}
}

func TestParseBackendsInvalid(t *testing.T) {
	for _, s := range []string{"noequals", "=http://x", "name="} {
		if _, err := parseBackends(s); err == nil {
			t.Errorf("parseBackends(%q): expected error", s)
		}
	}
}

func TestBuildRouterUnknownStrategy(t *testing.T) {
	if _, err := buildRouter("bogus", 100, nil); err == nil {
		t.Error("expected error for unknown strategy")
	}
}
