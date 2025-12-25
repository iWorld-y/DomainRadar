package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// DomainReport holds the schema definition for the DomainReport entity.
type DomainReport struct {
	ent.Schema
}

// Fields of the DomainReport.
func (DomainReport) Fields() []ent.Field {
	return []ent.Field{
		field.Int("run_id").Optional(),
		field.String("domain_name"),
		field.String("overview").Optional(),
		field.String("trends").Optional(),
		field.Int("score").Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the DomainReport.
func (DomainReport) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("report_run", ReportRun.Type).
			Ref("domain_reports").
			Field("run_id").
			Unique(),
		edge.To("articles", Article.Type),
		edge.To("key_events", KeyEvent.Type),
	}
}
