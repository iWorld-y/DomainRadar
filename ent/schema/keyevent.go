package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// KeyEvent holds the schema definition for the KeyEvent entity.
type KeyEvent struct {
	ent.Schema
}

// Fields of the KeyEvent.
func (KeyEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Int("domain_report_id").Optional(),
		field.String("event_content").Optional(),
	}
}

// Edges of the KeyEvent.
func (KeyEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("domain_report", DomainReport.Type).
			Ref("key_events").
			Field("domain_report_id").
			Unique(),
	}
}
