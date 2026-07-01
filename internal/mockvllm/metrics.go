package mockvllm

import "github.com/prometheus/client_golang/prometheus"

type metrics struct {
	registry     *prometheus.Registry
	promptTokens prometheus.Counter
	genTokens    prometheus.Counter
	cacheQueries prometheus.Counter
	cacheHits    prometheus.Counter
	running      prometheus.Gauge
	ttft         prometheus.Histogram
}

func newMetrics() *metrics {
	m := &metrics{
		registry: prometheus.NewRegistry(),
		promptTokens: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "vllm:prompt_tokens_total",
			Help: "Number of prefill (prompt) tokens processed.",
		}),
		genTokens: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "vllm:generation_tokens_total",
			Help: "Number of generation tokens produced.",
		}),
		cacheQueries: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "vllm:gpu_prefix_cache_queries_total",
			Help: "Prompt tokens looked up in the prefix cache.",
		}),
		cacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "vllm:gpu_prefix_cache_hits_total",
			Help: "Prompt tokens served from the prefix cache.",
		}),
		running: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "vllm:num_requests_running",
			Help: "Requests currently being processed.",
		}),
		ttft: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "vllm:time_to_first_token_seconds",
			Help:    "Time to first token in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}),
	}
	m.registry.MustRegister(m.promptTokens, m.genTokens, m.cacheQueries, m.cacheHits, m.running, m.ttft)
	return m
}
