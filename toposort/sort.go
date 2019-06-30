package toposrt

import (
	"errors"
	"fmt"
	"io"
)

type Node interface {
	Symbol() string
	GetDeps() map[string]Node
}

type Graph struct {
	Nodes []Node
}

func NewGraph(nodes ...Node) *Graph {
	return &Graph{
		Nodes: nodes,
	}
}

func DisplayGraph(w io.Writer, graph *Graph) {
	for _, node := range graph.Nodes {
		for _, dep := range node.GetDeps() {
			fmt.Fprintf(w, "%s -> %s\n", node, dep)
		}
	}
}

func ResolveGraph(graph *Graph) (Graph, error) {
	nodeNames := make(map[string]Node)
	nodeDependencies := make(map[string]map[string]Node)

	for _, node := range graph.Nodes {
		nodeNames[node.Symbol()] = node
		nodeDependencies[node.Symbol()] = node.GetDeps()
	}

	var resolved Graph

	for len(nodeDependencies) != 0 {
		readySet := make(map[string]struct{})

		for name, deps := range nodeDependencies {
			if len(deps) == 0 {
				readySet[name] = struct{}{}
			}
		}
		if len(readySet) == 0 {
			var g Graph
			for name := range nodeDependencies {
				g.Nodes = append(g.Nodes, nodeNames[name])
			}
			return g, errors.New("Circular Dependency")
		}

		for name, _ := range readySet {
			delete(nodeDependencies, name)
			resolved.Nodes = append(resolved.Nodes, nodeNames[name])
		}

		for name, deps := range nodeDependencies {
			diff := make(map[string]Node)
			for k, v := range deps {
				if _, ok := readySet[k]; !ok {
					diff[k] = v
				}
			}
			nodeDependencies[name] = diff
		}
	}

	return resolved, nil
}
