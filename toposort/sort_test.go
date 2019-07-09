package toposort

import (
	"strings"
	"testing"
)

type NodeImpl struct {
	name string
	deps []*NodeImpl
}

func NewNodeImpl(name string, deps ...*NodeImpl) *NodeImpl {
	return &NodeImpl{
		name: name,
		deps: deps,
	}
}

func (n *NodeImpl) GetDeps() []string {
	var deps []string

	for _, d := range n.deps {
		deps = append(deps, d.Symbol())
	}

	return deps
}

func (n *NodeImpl) Symbol() string {
	return n.String()
}

func (n *NodeImpl) String() string {
	return n.name
}

func TestResolveGraph(t *testing.T) {
	nodeA := NewNodeImpl("A")
	nodeB := NewNodeImpl("B")
	nodeC := NewNodeImpl("C", nodeA)
	nodeD := NewNodeImpl("D", nodeB)
	nodeE := NewNodeImpl("E", nodeC, nodeD)
	nodeF := NewNodeImpl("F", nodeA, nodeB)
	nodeG := NewNodeImpl("G", nodeE, nodeF)
	nodeH := NewNodeImpl("H", nodeG)
	nodeI := NewNodeImpl("I", nodeA)
	nodeJ := NewNodeImpl("J", nodeB)
	nodeK := NewNodeImpl("K")
	workingGraph := NewGraph(nodeA, nodeB, nodeC, nodeD, nodeE, nodeF, nodeG, nodeH,
		nodeI, nodeJ, nodeK)

	result, err := ResolveGraph(workingGraph)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	var resolved []string
	for _, node := range result.Nodes {
		deps := node.GetDeps()
		for _, d := range deps {
			if !strings.Contains(strings.Join(resolved, ""), d) {
				t.Errorf("node %s dep %s is not resolved", node.Symbol(), d)
			}
		}
		resolved = append(resolved, node.Symbol())
	}
}
