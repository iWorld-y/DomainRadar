package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Article holds the schema definition for the Article entity.
type Article struct {
	ent.Schema
}

// Fields of the Article.
func (Article) Fields() []ent.Field {
	return []ent.Field{
		field.Int("domain_report_id").Optional(),
		field.String("title").Optional(),
		field.String("link").Optional(),
		field.String("source").Optional(),
		field.String("pub_date").Optional(),
		field.String("content").Optional(),
	}
}

// Edges of the Article.
func (Article) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("domain_report", DomainReport.Type).
			Ref("articles").
			Field("domain_report_id").
			Unique(),
	}
}
