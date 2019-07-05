package xmigrate

import (
	"github.com/akito0107/xsqlparser/sqlast"

	"github.com/akito0107/xmigrate/toposort"
)

type DiffNode struct {
	diff *SchemaDiff
	deps []*DiffNode
}

func (d *DiffNode) Symbol() string {
	return d.diff.Spec.ToSQLString()
}

func (d *DiffNode) GetDeps() map[string]toposort.Node {
	deps := make(map[string]toposort.Node)
	for _, dep := range d.deps {
		deps[dep.Symbol()] = dep
	}

	return deps
}

func CalcGraph(diffs []*SchemaDiff) *toposort.Graph {
	var nodes []toposort.Node
	for _, d := range diffs {
		switch tp := d.Spec.(type) {
		case *AddTableSpec:
			var needsother bool
			for _, e := range tp.SQL.Elements {
				switch el := e.(type) {
				case *sqlast.SQLColumnDef:
					for _, c := range el.Constraints {
						ref, ok := c.Spec.(*sqlast.ReferencesColumnSpec)
						if !ok {
							continue
						}
					}
				case *sqlast.TableConstraint:

				}
			}
		case *AddColumnSpec:
		case *AddTableConstraintSpec:
		default:
			nodes = append(nodes, &DiffNode{
				diff: d,
			})
		}
	}

	return &toposort.Graph{
		Nodes: nodes,
	}
}
