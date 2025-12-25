package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ReportRun holds the schema definition for the ReportRun entity.
type ReportRun struct {
	ent.Schema
}

// Fields of the ReportRun.
func (ReportRun) Fields() []ent.Field {
	return []ent.Field{
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the ReportRun.
func (ReportRun) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("domain_reports", DomainReport.Type),
		edge.To("deep_analysis_results", DeepAnalysisResult.Type),
	}
}
