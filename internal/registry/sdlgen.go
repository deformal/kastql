package registry

import (
	"fmt"
	"strings"
)

// introspection response types

type introspectionSchema struct {
	QueryType        *namedType         `json:"queryType"`
	MutationType     *namedType         `json:"mutationType"`
	SubscriptionType *namedType         `json:"subscriptionType"`
	Types            []introspectedType `json:"types"`
	Directives       []introspectedDir  `json:"directives"`
}

type namedType struct {
	Name string `json:"name"`
}

type introspectedType struct {
	Kind          string              `json:"kind"`
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	Fields        []introspectedField `json:"fields"`
	InputFields   []introspectedArg   `json:"inputFields"`
	Interfaces    []typeRef           `json:"interfaces"`
	EnumValues    []enumValue         `json:"enumValues"`
	PossibleTypes []typeRef           `json:"possibleTypes"`
}

type introspectedField struct {
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	Args              []introspectedArg `json:"args"`
	Type              typeRef           `json:"type"`
	IsDeprecated      bool              `json:"isDeprecated"`
	DeprecationReason string            `json:"deprecationReason"`
}

type introspectedArg struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Type         typeRef `json:"type"`
	DefaultValue *string `json:"defaultValue"`
}

type typeRef struct {
	Kind   string   `json:"kind"`
	Name   string   `json:"name"`
	OfType *typeRef `json:"ofType"`
}

type enumValue struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	IsDeprecated      bool   `json:"isDeprecated"`
	DeprecationReason string `json:"deprecationReason"`
}

type introspectedDir struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Locations   []string          `json:"locations"`
	Args        []introspectedArg `json:"args"`
}

var builtinTypes = map[string]bool{
	"String": true, "Boolean": true, "Int": true, "Float": true, "ID": true,
	"__Schema": true, "__Type": true, "__Field": true, "__InputValue": true,
	"__EnumValue": true, "__Directive": true, "__DirectiveLocation": true,
}

var builtinDirectives = map[string]bool{
	"skip": true, "include": true, "deprecated": true, "specifiedBy": true,
}

func schemaToSDL(s *introspectionSchema) string {
	var b strings.Builder

	// schema block (only if non-standard root names)
	writeSchemaBlock(&b, s)

	for _, t := range s.Types {
		if builtinTypes[t.Name] || strings.HasPrefix(t.Name, "__") {
			continue
		}
		switch t.Kind {
		case "OBJECT":
			writeObject(&b, t)
		case "INTERFACE":
			writeInterface(&b, t)
		case "UNION":
			writeUnion(&b, t)
		case "ENUM":
			writeEnum(&b, t)
		case "INPUT_OBJECT":
			writeInput(&b, t)
		case "SCALAR":
			writeScalar(&b, t)
		}
	}

	for _, d := range s.Directives {
		if builtinDirectives[d.Name] {
			continue
		}
		writeDirective(&b, d)
	}

	return b.String()
}

func writeSchemaBlock(b *strings.Builder, s *introspectionSchema) {
	qName := "Query"
	mName := "Mutation"
	subName := "Subscription"
	if s.QueryType != nil {
		qName = s.QueryType.Name
	}
	if s.MutationType != nil {
		mName = s.MutationType.Name
	}
	if s.SubscriptionType != nil {
		subName = s.SubscriptionType.Name
	}

	nonStandard := qName != "Query" || (s.MutationType != nil && mName != "Mutation") ||
		(s.SubscriptionType != nil && subName != "Subscription")

	if nonStandard {
		b.WriteString("schema {\n")
		fmt.Fprintf(b, "  query: %s\n", qName)
		if s.MutationType != nil {
			fmt.Fprintf(b, "  mutation: %s\n", mName)
		}
		if s.SubscriptionType != nil {
			fmt.Fprintf(b, "  subscription: %s\n", subName)
		}
		b.WriteString("}\n\n")
	}
}

func writeObject(b *strings.Builder, t introspectedType) {
	writeDescription(b, t.Description, "")
	b.WriteString("type ")
	b.WriteString(t.Name)
	if len(t.Interfaces) > 0 {
		b.WriteString(" implements ")
		for i, iface := range t.Interfaces {
			if i > 0 {
				b.WriteString(" & ")
			}
			b.WriteString(iface.Name)
		}
	}
	b.WriteString(" {\n")
	for _, f := range t.Fields {
		writeField(b, f)
	}
	b.WriteString("}\n\n")
}

func writeInterface(b *strings.Builder, t introspectedType) {
	writeDescription(b, t.Description, "")
	b.WriteString("interface ")
	b.WriteString(t.Name)
	b.WriteString(" {\n")
	for _, f := range t.Fields {
		writeField(b, f)
	}
	b.WriteString("}\n\n")
}

func writeUnion(b *strings.Builder, t introspectedType) {
	writeDescription(b, t.Description, "")
	b.WriteString("union ")
	b.WriteString(t.Name)
	b.WriteString(" = ")
	for i, pt := range t.PossibleTypes {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(pt.Name)
	}
	b.WriteString("\n\n")
}

func writeEnum(b *strings.Builder, t introspectedType) {
	writeDescription(b, t.Description, "")
	b.WriteString("enum ")
	b.WriteString(t.Name)
	b.WriteString(" {\n")
	for _, v := range t.EnumValues {
		writeDescription(b, v.Description, "  ")
		b.WriteString("  ")
		b.WriteString(v.Name)
		if v.IsDeprecated {
			b.WriteString(" @deprecated")
			if v.DeprecationReason != "" && v.DeprecationReason != "No longer supported" {
				fmt.Fprintf(b, `(reason: "%s")`, v.DeprecationReason)
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("}\n\n")
}

func writeInput(b *strings.Builder, t introspectedType) {
	writeDescription(b, t.Description, "")
	b.WriteString("input ")
	b.WriteString(t.Name)
	b.WriteString(" {\n")
	for _, f := range t.InputFields {
		writeDescription(b, f.Description, "  ")
		fmt.Fprintf(b, "  %s: %s", f.Name, typeRefToString(f.Type))
		if f.DefaultValue != nil {
			fmt.Fprintf(b, " = %s", *f.DefaultValue)
		}
		b.WriteString("\n")
	}
	b.WriteString("}\n\n")
}

func writeScalar(b *strings.Builder, t introspectedType) {
	writeDescription(b, t.Description, "")
	fmt.Fprintf(b, "scalar %s\n\n", t.Name)
}

func writeDirective(b *strings.Builder, d introspectedDir) {
	writeDescription(b, d.Description, "")
	fmt.Fprintf(b, "directive @%s", d.Name)
	if len(d.Args) > 0 {
		b.WriteString("(")
		for i, a := range d.Args {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%s: %s", a.Name, typeRefToString(a.Type))
			if a.DefaultValue != nil {
				fmt.Fprintf(b, " = %s", *a.DefaultValue)
			}
		}
		b.WriteString(")")
	}
	b.WriteString(" on ")
	b.WriteString(strings.Join(d.Locations, " | "))
	b.WriteString("\n\n")
}

func writeField(b *strings.Builder, f introspectedField) {
	writeDescription(b, f.Description, "  ")
	b.WriteString("  ")
	b.WriteString(f.Name)
	if len(f.Args) > 0 {
		b.WriteString("(")
		for i, a := range f.Args {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%s: %s", a.Name, typeRefToString(a.Type))
			if a.DefaultValue != nil {
				fmt.Fprintf(b, " = %s", *a.DefaultValue)
			}
		}
		b.WriteString(")")
	}
	fmt.Fprintf(b, ": %s", typeRefToString(f.Type))
	if f.IsDeprecated {
		b.WriteString(" @deprecated")
		if f.DeprecationReason != "" && f.DeprecationReason != "No longer supported" {
			fmt.Fprintf(b, `(reason: "%s")`, f.DeprecationReason)
		}
	}
	b.WriteString("\n")
}

func typeRefToString(t typeRef) string {
	if t.Kind == "NON_NULL" && t.OfType != nil {
		return typeRefToString(*t.OfType) + "!"
	}
	if t.Kind == "LIST" && t.OfType != nil {
		return "[" + typeRefToString(*t.OfType) + "]"
	}
	return t.Name
}

func writeDescription(b *strings.Builder, desc, indent string) {
	if desc == "" {
		return
	}
	fmt.Fprintf(b, "%s\"\"\"%s\"\"\"\n", indent, desc)
}
