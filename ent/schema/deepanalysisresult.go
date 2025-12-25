package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// DeepAnalysisResult holds the schema definition for the DeepAnalysisResult entity.
type DeepAnalysisResult struct {
	ent.Schema
}

// Fields of the DeepAnalysisResult.
func (DeepAnalysisResult) Fields() []ent.Field {
	return []ent.Field{
		field.Int("run_id").Optional(),
		field.String("macro_trends").Optional(),
		field.String("opportunities").Optional(),
		field.String("risks").Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the DeepAnalysisResult.
func (DeepAnalysisResult) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("report_run", ReportRun.Type).
			Ref("deep_analysis_results").
			Field("run_id").
			Unique(),
		edge.To("action_guides", ActionGuide.Type),
	}
}
