package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Kris-Adhikari/KV-Aware-Inference-Router/internal/proxy"
	"github.com/Kris-Adhikari/KV-Aware-Inference-Router/internal/router"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	backendsFlag := flag.String("backends", "", "comma-separated name=url pairs, e.g. a=http://host:8000,b=http://host:8001")
	strategy := flag.String("strategy", "cache-aware", "routing strategy: cache-aware | round-robin")
	replicas := flag.Int("replicas", 100, "consistent-hash virtual nodes per backend (cache-aware only)")
	flag.Parse()

	backends, err := parseBackends(*backendsFlag)
	if err != nil {
		log.Fatalf("backends: %v", err)
	}
	if len(backends) == 0 {
		log.Fatal("no backends configured (use -backends name=url,...)")
	}

	r, err := buildRouter(*strategy, *replicas, backends)
	if err != nil {
		log.Fatal(err)
	}

	httpSrv := &http.Server{Addr: *addr, Handler: proxy.New(r)}

	go func() {
		log.Printf("router listening on %s strategy=%s backends=%d", *addr, *strategy, len(backends))
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func buildRouter(strategy string, replicas int, backends []*router.Backend) (router.Router, error) {
	switch strategy {
	case "cache-aware":
		return router.NewCacheAware(replicas, backends), nil
	case "round-robin":
		return router.NewRoundRobin(backends), nil
	default:
		return nil, fmt.Errorf("unknown strategy %q (want cache-aware or round-robin)", strategy)
	}
}

func parseBackends(s string) ([]*router.Backend, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []*router.Backend
	for pair := range strings.SplitSeq(s, ",") {
		name, url, ok := strings.Cut(pair, "=")
		name, url = strings.TrimSpace(name), strings.TrimSpace(url)
		if !ok || name == "" || url == "" {
			return nil, fmt.Errorf("invalid backend %q, want name=url", pair)
		}
		out = append(out, &router.Backend{Name: name, URL: url})
	}
	return out, nil
}
