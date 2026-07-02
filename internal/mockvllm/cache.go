package mockvllm

import (
	"strings"
	"sync"
	"time"
)

type Params struct {
	Base    time.Duration
	Prefill time.Duration
}

// Result is the outcome of admitting one prompt.
type Result struct {
	TTFT           time.Duration
	TotalTokens    int
	CachedTokens   int
	UncachedTokens int
	Hit            bool
}

type Cache struct {
	mu   sync.Mutex
	root *cacheNode
}

type cacheNode struct {
	children map[string]*cacheNode
}

func NewCache() *Cache {
	return &Cache{root: &cacheNode{children: map[string]*cacheNode{}}}
}

func (c *Cache) Reset() {
	c.mu.Lock()
	c.root = &cacheNode{children: map[string]*cacheNode{}}
	c.mu.Unlock()
}

func (c *Cache) Process(prompt string, p Params) Result {
	tokens := tokenize(prompt)

	c.mu.Lock()
	cached := c.cachedPrefixLen(tokens)
	c.add(tokens)
	c.mu.Unlock()

	uncached := len(tokens) - cached
	return Result{
		TTFT:           p.Base + time.Duration(uncached)*p.Prefill,
		TotalTokens:    len(tokens),
		CachedTokens:   cached,
		UncachedTokens: uncached,
		Hit:            cached > 0,
	}
}

func (c *Cache) cachedPrefixLen(tokens []string) int {
	n := c.root
	i := 0
	for ; i < len(tokens); i++ {
		next, ok := n.children[tokens[i]]
		if !ok {
			break
		}
		n = next
	}
	return i
}

func (c *Cache) add(tokens []string) {
	n := c.root
	for _, tok := range tokens {
		next, ok := n.children[tok]
		if !ok {
			next = &cacheNode{children: map[string]*cacheNode{}}
			n.children[tok] = next
		}
		n = next
	}
}

func tokenize(s string) []string { return strings.Fields(s) }
