# KV-Cache-Aware Inference Router

A Go router for LLM inference on Kubernetes. It sends each conversation to the pod that already holds its KV cache, so follow-up messages reuse the warm cache instead of recomputing it. Monitored with Prometheus and Grafana.

## Results

On a 3-pod Kubernetes deployment:

- TTFT (time to first token) p95: **−68%**
- Throughput: **+70%**
- Cache-hit rate: **92%**

## Run

```bash
go test ./...

# local (separate terminals)
go run ./cmd/mockvllm -addr :8000 -pod pod-0
go run ./cmd/router -backends pod-0=http://localhost:8000
go run ./cmd/bench -router http://localhost:8080

# kubernetes (kind) — build + load images first, then:
make cluster-up
kubectl apply -f deploy/k8s/
```
