package ring

import (
	"hash/crc32"
	"slices"
	"strconv"
)

// Ring is a consistent-hash ring over string members (backend names). Each
// member is placed at `replicas` points to smooth out the distribution.
type Ring struct {
	replicas int
	keys     []uint32
	points   map[uint32]string
	members  map[string]struct{}
}

func New(replicas int) *Ring {
	return &Ring{
		replicas: replicas,
		points:   make(map[uint32]string),
		members:  make(map[string]struct{}),
	}
}

func (r *Ring) Len() int { return len(r.members) }

func hashKey(s string) uint32 { return crc32.ChecksumIEEE([]byte(s)) }

func (r *Ring) Add(member string) {
	if _, ok := r.members[member]; ok {
		return
	}
	r.members[member] = struct{}{}
	for i := range r.replicas {
		r.points[hashKey(member+"#"+strconv.Itoa(i))] = member
	}
	r.rebuild()
}

func (r *Ring) Remove(member string) {
	if _, ok := r.members[member]; !ok {
		return
	}
	delete(r.members, member)
	for i := range r.replicas {
		delete(r.points, hashKey(member+"#"+strconv.Itoa(i)))
	}
	r.rebuild()
}

func (r *Ring) rebuild() {
	r.keys = r.keys[:0]
	for h := range r.points {
		r.keys = append(r.keys, h)
	}
	slices.Sort(r.keys)
}

// Get returns the member owning key: the first ring point clockwise from
// hash(key), wrapping past the end. ok is false only on an empty ring.
func (r *Ring) Get(key string) (member string, ok bool) {
	if len(r.keys) == 0 {
		return "", false
	}
	h := hashKey(key)
	i, _ := slices.BinarySearch(r.keys, h)
	if i == len(r.keys) {
		i = 0
	}
	return r.points[r.keys[i]], true
}
