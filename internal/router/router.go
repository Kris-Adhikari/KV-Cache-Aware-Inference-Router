package router

import "sync/atomic"

type Backend struct {
	Name string

	URL string
}

type Router interface {
	Route(key string) *Backend
}

type RoundRobin struct {
	backends []*Backend

	next atomic.Uint64
}

func NewRoundRobin(backends []*Backend) *RoundRobin {
	return &RoundRobin{backends: backends}
}

func (r *RoundRobin) Route(key string) *Backend {
	n := len(r.backends)
	if n == 0 {
		return nil
	}
	i := (r.next.Add(1) - 1) % uint64(n)
	return r.backends[i]
}
