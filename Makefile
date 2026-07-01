CLUSTER_NAME ?= kv-router
KIND_CONFIG  ?= kind-config.yaml

.PHONY: cluster-up cluster-down

## Create the local kind cluster (1 control-plane + 2 workers)
cluster-up:
	kind create cluster --name $(CLUSTER_NAME) --config $(KIND_CONFIG)
	kubectl cluster-info --context kind-$(CLUSTER_NAME)

## Delete the local kind cluster
cluster-down:
	kind delete cluster --name $(CLUSTER_NAME)
