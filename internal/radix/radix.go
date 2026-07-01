package radix

import "strings"

type node struct {
	edge     string
	children map[byte]*node
	owner    string
	hasOwner bool
}

// Tree is a radix (compressed prefix) tree mapping stored keys to an owner pod.
// It answers: which pod holds the longest stored prefix of this query?
// Not safe for concurrent use; the caller synchronizes.
type Tree struct {
	root *node
}

func New() *Tree {
	return &Tree{root: &node{children: map[byte]*node{}}}
}

func commonPrefixLen(a, b string) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

// Insert records that pod holds key (and therefore every prefix of key).
func (t *Tree) Insert(key, pod string) {
	cur := t.root
	for {
		if len(key) == 0 {
			cur.owner, cur.hasOwner = pod, true
			return
		}
		child, ok := cur.children[key[0]]
		if !ok {
			cur.children[key[0]] = &node{edge: key, children: map[byte]*node{}, owner: pod, hasOwner: true}
			return
		}
		cp := commonPrefixLen(child.edge, key)
		if cp == len(child.edge) {
			cur, key = child, key[cp:]
			continue
		}
		split := &node{edge: child.edge[:cp], children: map[byte]*node{}}
		child.edge = child.edge[cp:]
		split.children[child.edge[0]] = child
		cur.children[split.edge[0]] = split
		if cp == len(key) {
			split.owner, split.hasOwner = pod, true
		} else {
			rest := key[cp:]
			split.children[rest[0]] = &node{edge: rest, children: map[byte]*node{}, owner: pod, hasOwner: true}
		}
		return
	}
}

// Match returns the owner of the longest stored key that is a prefix of query,
// and how many bytes of query that key covers. ok is false when no stored key
// is a prefix of query.
func (t *Tree) Match(query string) (pod string, matched int, ok bool) {
	cur := t.root
	if cur.hasOwner {
		pod, matched, ok = cur.owner, 0, true
	}
	consumed := 0
	for len(query) > 0 {
		child, exists := cur.children[query[0]]
		if !exists || !strings.HasPrefix(query, child.edge) {
			break
		}
		consumed += len(child.edge)
		query = query[len(child.edge):]
		cur = child
		if cur.hasOwner {
			pod, matched, ok = cur.owner, consumed, true
		}
	}
	return
}
