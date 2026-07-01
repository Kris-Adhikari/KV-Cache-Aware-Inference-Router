package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Kris-Adhikari/KV-Aware-Inference-Router/internal/router"
)

func TestProxyForwardsToBackend(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("backend got path %q, want /v1/chat/completions", r.URL.Path)
		}
		w.Header().Set("X-Backend", "b1")
		io.WriteString(w, "hello from backend")
	}))
	defer backend.Close()

	p := New(router.NewRoundRobin([]*router.Backend{{Name: "b1", URL: backend.URL}}))
	front := httptest.NewServer(p)
	defer front.Close()

	resp, err := http.Post(front.URL+"/v1/chat/completions", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello from backend" {
		t.Errorf("body = %q, want %q", body, "hello from backend")
	}
	if resp.Header.Get("X-Backend") != "b1" {
		t.Errorf("X-Backend = %q, want b1", resp.Header.Get("X-Backend"))
	}
}

func TestProxyNoBackends(t *testing.T) {
	p := New(router.NewRoundRobin(nil))
	front := httptest.NewServer(p)
	defer front.Close()

	resp, err := http.Get(front.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
}

func TestProxyRoundRobinsAcrossBackends(t *testing.T) {
	newBackend := func(name string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, name)
		}))
	}
	b1 := newBackend("one")
	defer b1.Close()
	b2 := newBackend("two")
	defer b2.Close()

	p := New(router.NewRoundRobin([]*router.Backend{
		{Name: "b1", URL: b1.URL},
		{Name: "b2", URL: b2.URL},
	}))
	front := httptest.NewServer(p)
	defer front.Close()

	got := make([]string, 4)
	for i := range got {
		resp, err := http.Get(front.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		got[i] = string(body)
	}

	want := []string{"one", "two", "one", "two"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("req %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
