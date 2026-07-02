package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

type config struct {
	routerURL     string
	conversations int
	turns         int
	concurrency   int
	tokensPerTurn int
	label         string
}

type result struct {
	latency time.Duration
	hit     bool
	ok      bool
}

type reqMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type reqBody struct {
	Model    string   `json:"model"`
	Messages []reqMsg `json:"messages"`
}

func main() {
	var cfg config
	flag.StringVar(&cfg.routerURL, "router", "http://127.0.0.1:18080", "router base URL")
	flag.IntVar(&cfg.conversations, "conversations", 40, "number of conversations")
	flag.IntVar(&cfg.turns, "turns", 6, "turns per conversation")
	flag.IntVar(&cfg.concurrency, "concurrency", 8, "parallel workers")
	flag.IntVar(&cfg.tokensPerTurn, "tokens", 20, "new tokens added per turn")
	flag.StringVar(&cfg.label, "label", "", "label for the report")
	flag.Parse()

	start := time.Now()
	results := runLoad(cfg)
	printReport(cfg, results, time.Since(start))
}

func runLoad(cfg config) []result {
	jobs := make(chan int, cfg.conversations)
	for c := range cfg.conversations {
		jobs <- c
	}
	close(jobs)

	results := make(chan result, cfg.conversations*cfg.turns)
	client := &http.Client{Timeout: 30 * time.Second}

	var wg sync.WaitGroup
	for range cfg.concurrency {
		wg.Go(func() {
			for c := range jobs {
				runConversation(client, cfg, c, results)
			}
		})
	}
	wg.Wait()
	close(results)

	all := make([]result, 0, cfg.conversations*cfg.turns)
	for r := range results {
		all = append(all, r)
	}
	return all
}

func runConversation(client *http.Client, cfg config, conv int, out chan<- result) {
	session := fmt.Sprintf("conv-%d", conv)
	var b strings.Builder
	for turn := range cfg.turns {
		for i := range cfg.tokensPerTurn {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			fmt.Fprintf(&b, "c%dt%dw%d", conv, turn, i)
		}
		out <- doRequest(client, cfg, session, b.String())
	}
}

func doRequest(client *http.Client, cfg config, session, content string) result {
	body, _ := json.Marshal(reqBody{Model: "bench", Messages: []reqMsg{{Role: "user", Content: content}}})
	req, err := http.NewRequest(http.MethodPost, cfg.routerURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return result{}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-Id", session)

	start := time.Now()
	resp, err := client.Do(req)
	lat := time.Since(start)
	if err != nil {
		return result{latency: lat}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return result{
		latency: lat,
		hit:     resp.Header.Get("X-Cache-Hit") == "true",
		ok:      resp.StatusCode == http.StatusOK,
	}
}

type stats struct {
	count, hits, errors int
	avg, p50, p95, p99  time.Duration
}

func summarize(rs []result) stats {
	s := stats{count: len(rs)}
	if len(rs) == 0 {
		return s
	}
	lats := make([]time.Duration, 0, len(rs))
	var sum time.Duration
	for _, r := range rs {
		if !r.ok {
			s.errors++
		}
		if r.hit {
			s.hits++
		}
		lats = append(lats, r.latency)
		sum += r.latency
	}
	slices.Sort(lats)
	s.avg = sum / time.Duration(len(lats))
	s.p50 = percentile(lats, 50)
	s.p95 = percentile(lats, 95)
	s.p99 = percentile(lats, 99)
	return s
}

func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := p * len(sorted) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func printReport(cfg config, rs []result, wall time.Duration) {
	s := summarize(rs)
	fmt.Printf("== %s ==\n", cfg.label)
	fmt.Printf("requests:  %d  (errors: %d)\n", s.count, s.errors)
	if s.count > 0 {
		fmt.Printf("hit rate:  %.1f%%\n", 100*float64(s.hits)/float64(s.count))
	}
	fmt.Printf("TTFT avg:  %v\n", s.avg)
	fmt.Printf("TTFT p50:  %v\n", s.p50)
	fmt.Printf("TTFT p95:  %v\n", s.p95)
	fmt.Printf("TTFT p99:  %v\n", s.p99)
	if wall > 0 {
		fmt.Printf("wall:      %v  (%.0f req/s)\n", wall, float64(s.count)/wall.Seconds())
	}
}
