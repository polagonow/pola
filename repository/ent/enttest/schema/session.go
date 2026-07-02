package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Session holds the schema definition for the Session entity. It mirrors what
// the model generator emits for uuid-keyed entities: a string ID with a uuid
// DefaultFunc.
type Session struct {
	ent.Schema
}

// Fields of the Session.
func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").DefaultFunc(func() string { return uuid.New().String() }),
		field.String("note"),
	}
}
