package router

import (
	"github.com/serialx/hashring"
)

type Router struct {
	ring  *hashring.HashRing
	nodes []string
	self  string
}

func New(self string) *Router {
	return &Router{
		self:  self,
		ring:  hashring.New([]string{}),
		nodes: []string{},
	}
}

func (r *Router) GetHostByKey(key string) string {
	node, _ := r.ring.GetNode(key)
	return node
}

func (r *Router) IsSelf(node string) bool {
	return r.self == node
}

func (r *Router) Nodes() []string {
	return r.nodes
}

func (r *Router) SetNodes(nodes []string) {
	r.ring = hashring.New(nodes)
	r.nodes = nodes
}
