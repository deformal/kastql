package planner

import (
	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator/rules"
)

// QueryAnalysis holds structural metrics computed from an AST before execution.
type QueryAnalysis struct {
	Depth      int
	Complexity int
	Aliases    int
	Directives int
	IsMutation bool
}

// Analyze parses the query against the merged schema and returns structural
// metrics. Returns nil if the schema is not loaded or the query is invalid.
func (p *Planner) Analyze(query string) *QueryAnalysis {
	p.mu.RLock()
	merged := p.merged
	p.mu.RUnlock()
	if merged == nil {
		return nil
	}

	doc, errs := gqlparser.LoadQueryWithRules(merged.Schema, query, rules.NewDefaultRules())
	if errs != nil || doc == nil {
		return nil
	}

	a := &QueryAnalysis{}
	for _, op := range doc.Operations {
		if op.Operation == ast.Mutation {
			a.IsMutation = true
		}
		a.Directives += len(op.Directives)
		walkSelections(op.SelectionSet, 1, a)
	}
	return a
}

func walkSelections(sel ast.SelectionSet, depth int, a *QueryAnalysis) {
	if depth > a.Depth {
		a.Depth = depth
	}
	for _, s := range sel {
		switch f := s.(type) {
		case *ast.Field:
			a.Complexity++
			a.Directives += len(f.Directives)
			if f.Alias != "" && f.Alias != f.Name {
				a.Aliases++
			}
			walkSelections(f.SelectionSet, depth+1, a)
		case *ast.InlineFragment:
			a.Directives += len(f.Directives)
			walkSelections(f.SelectionSet, depth, a)
		case *ast.FragmentSpread:
			a.Directives += len(f.Directives)
		}
	}
}
