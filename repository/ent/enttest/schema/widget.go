// Package schema holds the ent schemas for the repository/ent test fixture.
// The generated client (../ent) is committed so framework tests run without
// invoking ent codegen; regenerate with:
//
//	go run -mod=mod entgo.io/ent/cmd/ent generate --target repository/ent/enttest/ent ./repository/ent/enttest/schema
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Widget holds the schema definition for the Widget entity. It mirrors the
// conformance suite's Widget (uint-keyed, one string field).
type Widget struct {
	ent.Schema
}

// Fields of the Widget.
func (Widget) Fields() []ent.Field {
	return []ent.Field{
		field.String("name"),
	}
}
