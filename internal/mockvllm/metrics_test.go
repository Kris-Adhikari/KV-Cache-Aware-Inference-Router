package mockvllm

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsExposesVLLMNames(t *testing.T) {
	srv := httptest.NewServer(NewServer("pod-x", Params{}).Handler())
	defer srv.Close()

	postChat(t, srv.URL, `{"model":"m","messages":[{"role":"user","content":"hello there"}]}`).Body.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	text := string(body)

	for _, name := range []string{
		"vllm:prompt_tokens_total",
		"vllm:generation_tokens_total",
		"vllm:gpu_prefix_cache_queries_total",
		"vllm:gpu_prefix_cache_hits_total",
		"vllm:num_requests_running",
		"vllm:time_to_first_token_seconds",
	} {
		if !strings.Contains(text, name) {
			t.Errorf("metrics missing %q", name)
		}
	}

	// "user: hello there" = 3 tokens, all uncached on the first request.
	if !strings.Contains(text, "vllm:prompt_tokens_total 3") {
		t.Errorf("prompt_tokens_total != 3 in:\n%s", text)
	}
	if !strings.Contains(text, "vllm:gpu_prefix_cache_hits_total 0") {
		t.Errorf("gpu_prefix_cache_hits_total != 0 in:\n%s", text)
	}
}
