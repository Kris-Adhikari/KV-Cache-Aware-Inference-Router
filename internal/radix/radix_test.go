package radix

import "testing"

type matchCase struct {
	query   string
	wantPod string
	wantLen int
	wantOK  bool
}

func run(t *testing.T, tr *Tree, cases []matchCase) {
	t.Helper()
	for _, c := range cases {
		pod, n, ok := tr.Match(c.query)
		if pod != c.wantPod || n != c.wantLen || ok != c.wantOK {
			t.Errorf("Match(%q) = (%q,%d,%v), want (%q,%d,%v)",
				c.query, pod, n, ok, c.wantPod, c.wantLen, c.wantOK)
		}
	}
}

func TestMatchEmptyTree(t *testing.T) {
	tr := New()
	if _, _, ok := tr.Match("anything"); ok {
		t.Fatal("empty tree matched")
	}
}

func TestExactAndPrefix(t *testing.T) {
	tr := New()
	tr.Insert("hello", "A")
	tr.Insert("hello world", "B")
	run(t, tr, []matchCase{
		{"hello world foo", "B", 11, true},
		{"hello there", "A", 5, true},
		{"hello", "A", 5, true},
		{"hel", "", 0, false},
		{"xyz", "", 0, false},
	})
}

func TestEdgeSplit(t *testing.T) {
	tr := New()
	tr.Insert("team", "A")
	tr.Insert("test", "B")
	run(t, tr, []matchCase{
		{"team", "A", 4, true},
		{"test", "B", 4, true},
		{"teamwork", "A", 4, true},
		{"te", "", 0, false},
		{"tea", "", 0, false},
	})
}

func TestOverwriteOwner(t *testing.T) {
	tr := New()
	tr.Insert("abc", "A")
	tr.Insert("abc", "B")
	if pod, _, ok := tr.Match("abc"); !ok || pod != "B" {
		t.Errorf("Match(abc) = (%q,%v), want (B,true)", pod, ok)
	}
}

func TestConversationFollowUp(t *testing.T) {
	tr := New()
	turn1 := "sys:helpful|user:hi"
	tr.Insert(turn1, "A")

	turn2 := turn1 + "|assistant:hello|user:bye"
	pod, matched, ok := tr.Match(turn2)
	if !ok || pod != "A" {
		t.Fatalf("follow-up routed to (%q,%v), want A,true", pod, ok)
	}
	if matched != len(turn1) {
		t.Errorf("matched=%d, want %d (the cached turn-1 prefix)", matched, len(turn1))
	}
}
