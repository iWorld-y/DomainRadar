package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id").SchemaType(map[string]string{
			dialect.Postgres: "serial",
		}),
		field.String("username").Unique(),
		field.String("password_hash"),
		field.String("persona").Optional().Comment("User persona for deep analysis"),
		field.JSON("domains", []string{}).Optional().Comment("User interested domains"),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}
