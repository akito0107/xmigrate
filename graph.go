package xmigrate

import (
	"github.com/akito0107/xsqlparser/sqlast"

	"github.com/akito0107/xmigrate/toposort"
)

type DiffNode struct {
	Diff *SchemaDiff
	deps []string
}

func (d *DiffNode) Symbol() string {
	return d.Diff.Spec.ToSQLString()
}

func (d *DiffNode) GetDeps() []string {
	return d.deps
}

// Calculate Diff's dependencies
func CalcGraph(diffs []*SchemaDiff) *toposort.Graph {

	// table name / diffs which are associated it table
	tables := make(map[string][]string)

	// diffs / depends table name Diff
	deps := make(map[*SchemaDiff][]string)

	for _, d := range diffs {
		switch spec := d.Spec.(type) {
		case *AddTableSpec:
			for _, e := range spec.SQL.Elements {
				switch el := e.(type) {
				case *sqlast.ColumnDef:
					for _, c := range el.Constraints {
						ref, ok := c.Spec.(*sqlast.ReferencesColumnSpec)
						if !ok {
							continue
						}
						deps[d] = append(deps[d], ref.TableName.ToSQLString())
					}

				case *sqlast.TableConstraint:
					ref, ok := el.Spec.(*sqlast.ReferentialTableConstraint)
					if !ok {
						continue
					}
					deps[d] = append(deps[d], ref.KeyExpr.TableName.ToSQLString())
				}
			}

			// add table name to memo
			tables[spec.SQL.Name.ToSQLString()] = []string{spec.ToSQLString()}

		case *AddColumnSpec:
			for _, c := range spec.ColumnDef.Constraints {
				ref, ok := c.Spec.(*sqlast.ReferencesColumnSpec)
				if !ok {
					continue
				}
				deps[d] = append(deps[d], ref.TableName.ToSQLString())
			}
			tables[spec.TableName] = append(tables[spec.TableName], spec.ToSQLString())

		case *AddTableConstraintSpec:
			ref, ok := spec.ConstraintDef.Spec.(*sqlast.ReferentialTableConstraint)
			if ok {
				deps[d] = append(deps[d], ref.KeyExpr.TableName.ToSQLString())
			}
			tables[spec.TableName] = append(tables[spec.TableName], spec.ToSQLString())
		case *AddIndexSpec:
			deps[d] = append(deps[d], spec.Def.TableName.ToSQLString())
		}
	}

	var nodes []toposort.Node

	for _, d := range diffs {
		n := &DiffNode{
			Diff: d,
		}
		dependTables, ok := deps[d]

		if ok {
			for _, t := range dependTables {
				dependTableSpecs, ok := tables[t]
				if !ok {
					continue
				}
				n.deps = append(n.deps, dependTableSpecs...)
			}
		}

		nodes = append(nodes, n)
	}

	return &toposort.Graph{
		Nodes: nodes,
	}
}
