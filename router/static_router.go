package router

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// StaticRouter keeps routing table: which node is responsible for which account
type StaticRouter struct {
	sync.Mutex

	shards []uint64
	nodes  []string
	self   string
}

func NewStatic(self string) *StaticRouter {
	return &StaticRouter{
		self:  self,
		nodes: []string{},
	}
}

// GetHostByKey returns host for given key. Key is usually accountID
func (r *StaticRouter) GetHostByKey(key string) string {
	defer r.Unlock()
	r.Lock()

	id, err := strconv.ParseUint(key, 10, 64)
	if err != nil {
		panic(err)
	}

	l := len(r.shards)

	idx := sort.Search(l, func(i int) bool {
		return id < r.shards[i]
	})
	idx = (idx - 1 + l) % l

	return r.nodes[idx]
}

// IsSelf checks if current node is equal to given
func (r *StaticRouter) IsSelf(node string) bool {
	defer r.Unlock()
	r.Lock()

	return r.self == node
}

func (r *StaticRouter) Nodes() []string {
	defer r.Unlock()
	r.Lock()

	nodes := make([]string, 0, len(r.nodes))

	for i, n := range r.nodes {
		nodes = append(nodes, fmt.Sprintf("%d=%s", r.shards[i], n))
	}

	return nodes
}

func (r *StaticRouter) SetNodes(shardNodes []string) {
	defer r.Unlock()
	r.Lock()

	r.nodes = make([]string, 0, len(shardNodes))
	r.shards = make([]uint64, 0, len(shardNodes))

	equalPart := (1 << 63) / uint64(len(shardNodes))
	equalPart <<= 1

	for i, n := range shardNodes {
		if n == "" {
			continue
		}

		s := strings.Split(n, "=")
		switch len(s) {
		case 2:
			shard, err := strconv.ParseUint(s[0], 10, 64)
			if err != nil {
				// don't omit errors
				panic("format error: point parsing error")
			}

			r.shards = append(r.shards, shard)
			r.nodes = append(r.nodes, s[1])
		case 1:
			shard := uint64(i) * equalPart
			r.shards = append(r.shards, shard)
			r.nodes = append(r.nodes, n)
		default:
			panic("format error: must be '{host}' or '{id=host}'")
		}
	}
}

func (r *StaticRouter) Self() string {
	defer r.Unlock()
	r.Lock()

	return r.self
}

func (r *StaticRouter) SetSelf(s string) {
	defer r.Unlock()
	r.Lock()

	r.self = s
}
