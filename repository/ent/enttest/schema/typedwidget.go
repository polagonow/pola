package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// TypedWidget holds the schema definition for the TypedWidget entity. It
// carries one field per scalar kind the repository generator emits, so tests
// exercise every SetField type-dispatch path.
type TypedWidget struct {
	ent.Schema
}

// Fields of the TypedWidget.
func (TypedWidget) Fields() []ent.Field {
	return []ent.Field{
		field.String("title"),
		field.Int("count"),
		field.Float("rate"),
		field.Bool("done"),
	}
}
