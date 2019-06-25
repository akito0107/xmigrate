package toposort

import (
	"errors"
	"fmt"
	"log"

	mapset "github.com/deckarep/golang-set"
)

func main() {
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

	fmt.Printf(">>> A working dependency graph\n")
	displayGraph(workingGraph)

	resolved, err := resolveGraph(workingGraph)
	if err != nil {
		log.Println(resolved)
		log.Fatal(err)
	} else {
		fmt.Println("resolved success")
	}

	for _, node := range resolved.Nodes {
		fmt.Println(node)
	}

}

type Node interface {
	Symbol() string
	GetDeps() []Node
}

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

func (n *NodeImpl) GetDeps() []Node {
	var deps []Node
	for _, d := range n.deps {
		deps = append(deps, d)
	}

	return deps
}

func (n *NodeImpl) Symbol() string {
	return n.String()
}

func (n *NodeImpl) String() string {
	return n.name
}

type Graph struct {
	Nodes []Node
}

func NewGraph(nodes ...Node) *Graph {
	return &Graph{
		Nodes: nodes,
	}
}

func displayGraph(graph *Graph) {
	for _, node := range graph.Nodes {
		for _, dep := range node.GetDeps() {
			fmt.Printf("%s -> %s\n", node, dep)
		}
	}
}

func resolveGraph(graph *Graph) (Graph, error) {
	nodeNames := make(map[string]Node)
	nodeDependencies := make(map[string]mapset.Set)

	for _, node := range graph.Nodes {
		nodeNames[node.Symbol()] = node
		dependencySet := mapset.NewSet()

		for _, dep := range node.GetDeps() {
			dependencySet.Add(dep)
		}

		nodeDependencies[node.Symbol()] = dependencySet
	}

	var resolved Graph

	for len(nodeDependencies) != 0 {
		readySet := mapset.NewSet()

		for name, deps := range nodeDependencies {
			if deps.Cardinality() == 0 {
				readySet.Add(name)
			}
		}
		if readySet.Cardinality() == 0 {
			var g Graph
			for name := range nodeDependencies {
				g.Nodes = append(g.Nodes, nodeNames[name])
			}
			return g, errors.New("Circular Dependency")
		}

		for name := range readySet.Iter() {
			delete(nodeDependencies, name.(string))
			resolved.Nodes = append(resolved.Nodes, nodeNames[name.(string)])
		}

		log.Println(nodeDependencies)
		for name, deps := range nodeDependencies {
			diff := deps.Difference(readySet)
			log.Println(name)
			log.Println(deps)
			log.Println(readySet)
			log.Printf("diff: %s", diff)
			nodeDependencies[name] = diff
		}
		log.Println(nodeDependencies)
	}

	return resolved, nil
}
