package router

import (
	"sync"

	"github.com/Kris-Adhikari/KV-Aware-Inference-Router/internal/radix"
	"github.com/Kris-Adhikari/KV-Aware-Inference-Router/internal/ring"
)

var _ Router = (*CacheAware)(nil)

type CacheAware struct {
	mu       sync.Mutex
	tree     *radix.Tree
	ring     *ring.Ring
	backends map[string]*Backend
}

func NewCacheAware(replicas int, backends []*Backend) *CacheAware {
	c := &CacheAware{
		tree:     radix.New(),
		ring:     ring.New(replicas),
		backends: make(map[string]*Backend, len(backends)),
	}
	for _, b := range backends {
		c.backends[b.Name] = b
		c.ring.Add(b.Name)
	}
	return c
}

func (c *CacheAware) Add(b *Backend) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.backends[b.Name]; ok {
		return
	}
	c.backends[b.Name] = b
	c.ring.Add(b.Name)
}

func (c *CacheAware) Remove(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.backends, name)
	c.ring.Remove(name)
}

func (c *CacheAware) Route(key string) *Backend {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.backends) == 0 {
		return nil
	}
	name, _, ok := c.tree.Match(key)
	if !ok || c.backends[name] == nil {
		name, ok = c.ring.Get(key)
		if !ok {
			return nil
		}
	}
	c.tree.Insert(key, name)
	return c.backends[name]
}
