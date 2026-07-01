package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kris-Adhikari/KV-Aware-Inference-Router/internal/mockvllm"
)

func main() {
	addr := flag.String("addr", ":8000", "listen address")
	pod := flag.String("pod", defaultPodName(), "pod/instance name reported in responses and logs")
	base := flag.Duration("base", 20*time.Millisecond, "base TTFT latency")
	prefill := flag.Duration("prefill", 200*time.Microsecond, "prefill cost per uncached token")
	flag.Parse()

	srv := mockvllm.NewServer(*pod, mockvllm.Params{Base: *base, Prefill: *prefill})
	httpSrv := &http.Server{Addr: *addr, Handler: srv.Handler()}

	go func() {
		log.Printf("mock vllm %q listening on %s (base=%s prefill=%s/tok)", *pod, *addr, *base, *prefill)
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

func defaultPodName() string {
	if v := os.Getenv("POD_NAME"); v != "" {
		return v
	}
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "mock-0"
}
