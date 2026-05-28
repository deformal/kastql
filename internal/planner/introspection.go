package planner

import (
	"sort"

	"github.com/vektah/gqlparser/v2/ast"
)

// Schema returns the current merged *ast.Schema, or nil if no services are loaded.
func (p *Planner) Schema() *ast.Schema {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.merged == nil {
		return nil
	}
	return p.merged.Schema
}

// BuildIntrospectionResponse constructs the standard GraphQL introspection
// response data from the merged schema. The returned map is the value of the
// top-level "data" key in the response JSON.
func BuildIntrospectionResponse(schema *ast.Schema) map[string]any {
	b := &introspectionBuilder{schema: schema}
	return map[string]any{"__schema": b.buildSchema()}
}

type introspectionBuilder struct {
	schema *ast.Schema
}

func (b *introspectionBuilder) buildSchema() map[string]any {
	result := map[string]any{"description": nil}

	if b.schema.Query != nil {
		result["queryType"] = map[string]any{"name": b.schema.Query.Name}
	} else {
		result["queryType"] = nil
	}
	if b.schema.Mutation != nil {
		result["mutationType"] = map[string]any{"name": b.schema.Mutation.Name}
	} else {
		result["mutationType"] = nil
	}
	if b.schema.Subscription != nil {
		result["subscriptionType"] = map[string]any{"name": b.schema.Subscription.Name}
	} else {
		result["subscriptionType"] = nil
	}

	// Collect and sort type names for stable output.
	names := make([]string, 0, len(b.schema.Types))
	for name := range b.schema.Types {
		names = append(names, name)
	}
	sort.Strings(names)

	types := make([]map[string]any, 0, len(names))
	for _, name := range names {
		if federationScalarNames[name] {
			continue
		}
		types = append(types, b.buildType(b.schema.Types[name]))
	}
	result["types"] = types

	// Directives
	dirNames := make([]string, 0, len(b.schema.Directives))
	for name := range b.schema.Directives {
		dirNames = append(dirNames, name)
	}
	sort.Strings(dirNames)

	directives := make([]map[string]any, 0, len(dirNames))
	for _, name := range dirNames {
		if federationDirectiveNames[name] {
			continue
		}
		directives = append(directives, b.buildDirective(b.schema.Directives[name]))
	}
	result["directives"] = directives

	return result
}

func (b *introspectionBuilder) buildType(def *ast.Definition) map[string]any {
	result := map[string]any{
		"name":           def.Name,
		"description":    nilStr(def.Description),
		"specifiedByURL": nil,
	}

	switch def.Kind {
	case ast.Scalar:
		result["kind"] = "SCALAR"
		result["fields"] = nil
		result["inputFields"] = nil
		result["interfaces"] = nil
		result["enumValues"] = nil
		result["possibleTypes"] = nil

	case ast.Object:
		result["kind"] = "OBJECT"
		result["fields"] = b.buildFields(def.Fields)
		result["inputFields"] = nil
		result["interfaces"] = b.buildNamedRefs(def.Interfaces)
		result["enumValues"] = nil
		result["possibleTypes"] = nil

	case ast.Interface:
		result["kind"] = "INTERFACE"
		result["fields"] = b.buildFields(def.Fields)
		result["inputFields"] = nil
		result["interfaces"] = nil
		result["enumValues"] = nil
		result["possibleTypes"] = b.buildPossibleTypes(def.Name)

	case ast.Union:
		result["kind"] = "UNION"
		result["fields"] = nil
		result["inputFields"] = nil
		result["interfaces"] = nil
		result["enumValues"] = nil
		result["possibleTypes"] = b.buildNamedRefs(def.Types)

	case ast.Enum:
		result["kind"] = "ENUM"
		result["fields"] = nil
		result["inputFields"] = nil
		result["interfaces"] = nil
		result["enumValues"] = b.buildEnumValues(def.EnumValues)
		result["possibleTypes"] = nil

	case ast.InputObject:
		result["kind"] = "INPUT_OBJECT"
		result["fields"] = nil
		result["inputFields"] = b.buildInputFields(def.Fields)
		result["interfaces"] = nil
		result["enumValues"] = nil
		result["possibleTypes"] = nil
	}

	return result
}

func (b *introspectionBuilder) buildTypeRef(t *ast.Type) map[string]any {
	if t == nil {
		return nil
	}
	if t.Elem != nil {
		// List wrapper
		inner := map[string]any{
			"kind":   "LIST",
			"name":   nil,
			"ofType": b.buildTypeRef(t.Elem),
		}
		if t.NonNull {
			return map[string]any{"kind": "NON_NULL", "name": nil, "ofType": inner}
		}
		return inner
	}
	// Named type
	named := map[string]any{
		"kind":   b.kindOf(t.NamedType),
		"name":   t.NamedType,
		"ofType": nil,
	}
	if t.NonNull {
		return map[string]any{"kind": "NON_NULL", "name": nil, "ofType": named}
	}
	return named
}

func (b *introspectionBuilder) kindOf(name string) string {
	def, ok := b.schema.Types[name]
	if !ok {
		return "SCALAR"
	}
	switch def.Kind {
	case ast.Scalar:
		return "SCALAR"
	case ast.Object:
		return "OBJECT"
	case ast.Interface:
		return "INTERFACE"
	case ast.Union:
		return "UNION"
	case ast.Enum:
		return "ENUM"
	case ast.InputObject:
		return "INPUT_OBJECT"
	}
	return "SCALAR"
}

func (b *introspectionBuilder) buildFields(fields ast.FieldList) []map[string]any {
	result := make([]map[string]any, 0, len(fields))
	for _, f := range fields {
		dep := f.Directives.ForName("deprecated")
		isDeprecated := dep != nil
		var deprecationReason any
		if isDeprecated {
			if a := dep.Arguments.ForName("reason"); a != nil {
				deprecationReason = valueToString(a.Value)
			}
		}
		result = append(result, map[string]any{
			"name":              f.Name,
			"description":       nilStr(f.Description),
			"args":              b.buildArgDefs(f.Arguments),
			"type":              b.buildTypeRef(f.Type),
			"isDeprecated":      isDeprecated,
			"deprecationReason": deprecationReason,
		})
	}
	return result
}

func (b *introspectionBuilder) buildInputFields(fields ast.FieldList) []map[string]any {
	result := make([]map[string]any, 0, len(fields))
	for _, f := range fields {
		var defaultValue any
		if f.DefaultValue != nil {
			defaultValue = valueToString(f.DefaultValue)
		}
		result = append(result, map[string]any{
			"name":              f.Name,
			"description":       nilStr(f.Description),
			"type":              b.buildTypeRef(f.Type),
			"defaultValue":      defaultValue,
			"isDeprecated":      false,
			"deprecationReason": nil,
		})
	}
	return result
}

func (b *introspectionBuilder) buildArgDefs(args ast.ArgumentDefinitionList) []map[string]any {
	result := make([]map[string]any, 0, len(args))
	for _, arg := range args {
		var defaultValue any
		if arg.DefaultValue != nil {
			defaultValue = valueToString(arg.DefaultValue)
		}
		result = append(result, map[string]any{
			"name":              arg.Name,
			"description":       nilStr(arg.Description),
			"type":              b.buildTypeRef(arg.Type),
			"defaultValue":      defaultValue,
			"isDeprecated":      false,
			"deprecationReason": nil,
		})
	}
	return result
}

func (b *introspectionBuilder) buildEnumValues(values ast.EnumValueList) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, v := range values {
		dep := v.Directives.ForName("deprecated")
		isDeprecated := dep != nil
		var deprecationReason any
		if isDeprecated {
			if a := dep.Arguments.ForName("reason"); a != nil {
				deprecationReason = valueToString(a.Value)
			}
		}
		result = append(result, map[string]any{
			"name":              v.Name,
			"description":       nilStr(v.Description),
			"isDeprecated":      isDeprecated,
			"deprecationReason": deprecationReason,
		})
	}
	return result
}

func (b *introspectionBuilder) buildNamedRefs(names []string) []map[string]any {
	result := make([]map[string]any, 0, len(names))
	for _, name := range names {
		result = append(result, map[string]any{
			"kind":   b.kindOf(name),
			"name":   name,
			"ofType": nil,
		})
	}
	return result
}

func (b *introspectionBuilder) buildPossibleTypes(interfaceName string) []map[string]any {
	result := make([]map[string]any, 0)
	// Collect and sort for stable output.
	var implementors []string
	for _, def := range b.schema.Types {
		if def.Kind != ast.Object {
			continue
		}
		for _, iface := range def.Interfaces {
			if iface == interfaceName {
				implementors = append(implementors, def.Name)
				break
			}
		}
	}
	sort.Strings(implementors)
	for _, name := range implementors {
		result = append(result, map[string]any{
			"kind": "OBJECT", "name": name, "ofType": nil,
		})
	}
	return result
}

func (b *introspectionBuilder) buildDirective(d *ast.DirectiveDefinition) map[string]any {
	locs := make([]string, 0, len(d.Locations))
	for _, loc := range d.Locations {
		locs = append(locs, string(loc))
	}
	return map[string]any{
		"name":         d.Name,
		"description":  nilStr(d.Description),
		"isRepeatable": d.IsRepeatable,
		"locations":    locs,
		"args":         b.buildArgDefs(d.Arguments),
	}
}

func nilStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
