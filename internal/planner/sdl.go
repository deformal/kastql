package planner

import (
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

var federationDirectiveNames = map[string]bool{
	"key": true, "external": true, "requires": true, "provides": true,
	"shareable": true, "inaccessible": true, "override": true,
	"link": true, "extends": true, "tag": true,
}

var federationScalarNames = map[string]bool{
	"_Any": true, "_FieldSet": true, "FieldSet": true,
	"_Service": true, "link__Import": true, "link__Purpose": true,
}

// definitionToSDL renders a *ast.Definition to SDL text, stripping federation
// directives from the output.
func definitionToSDL(b *strings.Builder, def *ast.Definition) {
	if def.Kind == ast.Scalar {
		if def.Description != "" {
			fmt.Fprintf(b, "\"\"\"%s\"\"\"\n", def.Description)
		}
		fmt.Fprintf(b, "scalar %s\n\n", def.Name)
		return
	}

	if def.Description != "" {
		fmt.Fprintf(b, "\"\"\"%s\"\"\"\n", def.Description)
	}

	switch def.Kind {
	case ast.Object:
		b.WriteString("type ")
	case ast.Interface:
		b.WriteString("interface ")
	case ast.InputObject:
		b.WriteString("input ")
	case ast.Union:
		b.WriteString("union ")
	case ast.Enum:
		b.WriteString("enum ")
	default:
		return
	}

	b.WriteString(def.Name)

	if len(def.Interfaces) > 0 {
		b.WriteString(" implements ")
		b.WriteString(strings.Join(def.Interfaces, " & "))
	}

	if def.Kind == ast.Union {
		b.WriteString(" = ")
		b.WriteString(strings.Join(def.Types, " | "))
		b.WriteString("\n\n")
		return
	}

	b.WriteString(" {\n")

	if def.Kind == ast.Enum {
		for _, v := range def.EnumValues {
			if v.Description != "" {
				fmt.Fprintf(b, "  \"\"\"%s\"\"\"\n", v.Description)
			}
			fmt.Fprintf(b, "  %s", v.Name)
			if v.Directives.ForName("deprecated") != nil {
				b.WriteString(" @deprecated")
			}
			b.WriteString("\n")
		}
	} else {
		for _, f := range def.Fields {
			fieldDefToSDL(b, f, "  ")
		}
	}

	b.WriteString("}\n\n")
}

// fieldDefToSDL writes an *ast.FieldDefinition (or InputFieldDefinition) as SDL.
func fieldDefToSDL(b *strings.Builder, f *ast.FieldDefinition, indent string) {
	if f.Description != "" {
		fmt.Fprintf(b, "%s\"\"\"%s\"\"\"\n", indent, f.Description)
	}
	fmt.Fprintf(b, "%s%s", indent, f.Name)

	if len(f.Arguments) > 0 {
		b.WriteString("(")
		for i, arg := range f.Arguments {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%s: %s", arg.Name, astTypeToSDL(arg.Type))
			if arg.DefaultValue != nil {
				fmt.Fprintf(b, " = %s", arg.DefaultValue.Raw)
			}
		}
		b.WriteString(")")
	}

	fmt.Fprintf(b, ": %s", astTypeToSDL(f.Type))

	for _, dir := range f.Directives {
		if federationDirectiveNames[dir.Name] {
			continue
		}
		writeDirectiveSDL(b, dir)
	}

	b.WriteString("\n")
}

func writeDirectiveSDL(b *strings.Builder, dir *ast.Directive) {
	fmt.Fprintf(b, " @%s", dir.Name)
	if len(dir.Arguments) > 0 {
		b.WriteString("(")
		for i, arg := range dir.Arguments {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%s: %s", arg.Name, arg.Value.Raw)
		}
		b.WriteString(")")
	}
}

// astTypeToSDL converts an *ast.Type node to its SDL string representation.
func astTypeToSDL(t *ast.Type) string {
	if t == nil {
		return ""
	}
	if t.NamedType != "" {
		if t.NonNull {
			return t.NamedType + "!"
		}
		return t.NamedType
	}
	elem := astTypeToSDL(t.Elem)
	if t.NonNull {
		return "[" + elem + "]!"
	}
	return "[" + elem + "]"
}

// namedTypeName unwraps NonNull/List wrappers and returns the bare named type.
func namedTypeName(t *ast.Type) string {
	if t == nil {
		return ""
	}
	if t.NamedType != "" {
		return t.NamedType
	}
	return namedTypeName(t.Elem)
}

// selectionToQueryString serialises an ast.SelectionSet back to a query
// fragment string (without outer braces). extraFields are injected at the
// top of the output (used to ensure @key fields are present).
func selectionToQueryString(sel ast.SelectionSet, extraFields []string, indent string) string {
	var b strings.Builder
	for _, f := range extraFields {
		fmt.Fprintf(&b, "%s%s\n", indent, f)
	}
	writeSelectionSet(&b, sel, indent)
	return b.String()
}

func writeSelectionSet(b *strings.Builder, sel ast.SelectionSet, indent string) {
	for _, s := range sel {
		switch f := s.(type) {
		case *ast.Field:
			alias := ""
			if f.Alias != f.Name {
				alias = f.Alias + ": "
			}
			fmt.Fprintf(b, "%s%s%s", indent, alias, f.Name)
			if len(f.Arguments) > 0 {
				b.WriteString("(")
				for i, arg := range f.Arguments {
					if i > 0 {
						b.WriteString(", ")
					}
					fmt.Fprintf(b, "%s: %s", arg.Name, valueToString(arg.Value))
				}
				b.WriteString(")")
			}
			if len(f.SelectionSet) > 0 {
				b.WriteString(" {\n")
				writeSelectionSet(b, f.SelectionSet, indent+"  ")
				fmt.Fprintf(b, "%s}", indent)
			}
			b.WriteString("\n")
		case *ast.InlineFragment:
			fmt.Fprintf(b, "%s... on %s {\n", indent, f.TypeCondition)
			writeSelectionSet(b, f.SelectionSet, indent+"  ")
			fmt.Fprintf(b, "%s}\n", indent)
		case *ast.FragmentSpread:
			fmt.Fprintf(b, "%s...%s\n", indent, f.Name)
		}
	}
}

// valueToString serialises an *ast.Value back to valid GraphQL syntax.
// ast.Value.Raw is only populated for scalars; objects and lists must be
// reconstructed from their Children.
func valueToString(v *ast.Value) string {
	if v == nil {
		return "null"
	}
	switch v.Kind {
	case ast.Variable:
		return "$" + v.Raw
	case ast.IntValue, ast.FloatValue, ast.EnumValue:
		return v.Raw
	case ast.StringValue:
		// Re-quote, escaping any embedded quotes.
		return `"` + strings.ReplaceAll(v.Raw, `"`, `\"`) + `"`
	case ast.BlockValue:
		return `"""` + v.Raw + `"""`
	case ast.BooleanValue, ast.NullValue:
		return v.Raw
	case ast.ListValue:
		parts := make([]string, 0, len(v.Children))
		for _, child := range v.Children {
			parts = append(parts, valueToString(child.Value))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case ast.ObjectValue:
		parts := make([]string, 0, len(v.Children))
		for _, child := range v.Children {
			parts = append(parts, child.Name+": "+valueToString(child.Value))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		if v.Raw != "" {
			return v.Raw
		}
		return "null"
	}
}

// varDefsToSDL writes variable definitions like ($id: ID!, $limit: Int).
func varDefsToSDL(defs ast.VariableDefinitionList) string {
	if len(defs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(defs))
	for _, d := range defs {
		s := "$" + d.Variable + ": " + astTypeToSDL(d.Type)
		if d.DefaultValue != nil {
			s += " = " + d.DefaultValue.Raw
		}
		parts = append(parts, s)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
