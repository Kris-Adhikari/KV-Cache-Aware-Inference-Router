package mockvllm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func postChat(t *testing.T, base, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(base+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestChatReturnsOpenAIShape(t *testing.T) {
	srv := httptest.NewServer(NewServer("pod-x", Params{}).Handler())
	defer srv.Close()

	resp := postChat(t, srv.URL, `{"model":"m","messages":[{"role":"user","content":"hello there"}]}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Mock-Pod") != "pod-x" {
		t.Errorf("X-Mock-Pod = %q", resp.Header.Get("X-Mock-Pod"))
	}

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		t.Fatal(err)
	}
	if len(cr.Choices) != 1 || cr.Choices[0].Message.Role != "assistant" {
		t.Fatalf("unexpected choices: %+v", cr.Choices)
	}
	if cr.Choices[0].FinishReason != "stop" {
		t.Errorf("finish_reason = %q", cr.Choices[0].FinishReason)
	}
	// "user: hello there" -> 3 whitespace tokens
	if cr.Usage.PromptTokens != 3 {
		t.Errorf("prompt_tokens = %d, want 3", cr.Usage.PromptTokens)
	}
}

func TestChatCacheHitHeader(t *testing.T) {
	srv := httptest.NewServer(NewServer("pod-x", Params{}).Handler())
	defer srv.Close()

	body := `{"model":"m","messages":[{"role":"user","content":"hello there"}]}`

	r1 := postChat(t, srv.URL, body)
	r1.Body.Close()
	if r1.Header.Get("X-Cache-Hit") != "false" {
		t.Errorf("first request X-Cache-Hit = %q, want false", r1.Header.Get("X-Cache-Hit"))
	}

	r2 := postChat(t, srv.URL, body)
	r2.Body.Close()
	if r2.Header.Get("X-Cache-Hit") != "true" {
		t.Errorf("repeat request X-Cache-Hit = %q, want true", r2.Header.Get("X-Cache-Hit"))
	}
}

func TestChatSimulatesLatency(t *testing.T) {
	srv := httptest.NewServer(NewServer("pod-x", Params{Base: 30 * time.Millisecond}).Handler())
	defer srv.Close()

	start := time.Now()
	resp := postChat(t, srv.URL, `{"model":"m","messages":[{"role":"user","content":"a b c"}]}`)
	resp.Body.Close()
	if elapsed := time.Since(start); elapsed < 25*time.Millisecond {
		t.Errorf("elapsed %v, want >= ~30ms (Base)", elapsed)
	}
}

func TestChatBadJSON(t *testing.T) {
	srv := httptest.NewServer(NewServer("pod-x", Params{}).Handler())
	defer srv.Close()

	resp := postChat(t, srv.URL, `{not json`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestChatWrongMethod(t *testing.T) {
	srv := httptest.NewServer(NewServer("pod-x", Params{}).Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}

func TestHealthz(t *testing.T) {
	srv := httptest.NewServer(NewServer("pod-x", Params{}).Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestBuildPrompt(t *testing.T) {
	got := buildPrompt([]chatMessage{
		{Role: "system", Content: "be nice"},
		{Role: "user", Content: "hi"},
	})
	want := "system: be nice\nuser: hi"
	if got != want {
		t.Errorf("buildPrompt = %q, want %q", got, want)
	}
}
