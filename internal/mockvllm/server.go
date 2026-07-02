package mockvllm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	podName string
	params  Params
	cache   *Cache
	metrics *metrics
}

func NewServer(podName string, params Params) *Server {
	return &Server{podName: podName, params: params, cache: NewCache(), metrics: newMetrics()}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", s.handleChat)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /reset", func(w http.ResponseWriter, _ *http.Request) {
		s.cache.Reset()
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("GET /metrics", promhttp.HandlerFor(s.metrics.registry, promhttp.HandlerOpts{}))
	return mux
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type choice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	s.metrics.running.Inc()
	defer s.metrics.running.Dec()

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	res := s.cache.Process(buildPrompt(req.Messages), s.params)
	s.metrics.promptTokens.Add(float64(res.TotalTokens))
	s.metrics.cacheQueries.Add(float64(res.TotalTokens))
	s.metrics.cacheHits.Add(float64(res.CachedTokens))
	s.metrics.ttft.Observe(res.TTFT.Seconds())

	select {
	case <-time.After(res.TTFT):
	case <-r.Context().Done():
		return
	}
	s.metrics.genTokens.Add(1)

	resp := chatResponse{
		ID:      "chatcmpl-mock-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []choice{{
			Index: 0,
			Message: chatMessage{
				Role:    "assistant",
				Content: fmt.Sprintf("[mock:%s] %d tokens, %d cached", s.podName, res.TotalTokens, res.CachedTokens),
			},
			FinishReason: "stop",
		}},
		Usage: usage{PromptTokens: res.TotalTokens, CompletionTokens: 1, TotalTokens: res.TotalTokens + 1},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Mock-Pod", s.podName)
	w.Header().Set("X-Cache-Hit", strconv.FormatBool(res.Hit))
	json.NewEncoder(w).Encode(resp)
}

func buildPrompt(msgs []chatMessage) string {
	var b strings.Builder
	for i, m := range msgs {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(m.Role)
		b.WriteString(": ")
		b.WriteString(m.Content)
	}
	return b.String()
}
