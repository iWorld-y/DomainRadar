package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ActionGuide holds the schema definition for the ActionGuide entity.
type ActionGuide struct {
	ent.Schema
}

// Fields of the ActionGuide.
func (ActionGuide) Fields() []ent.Field {
	return []ent.Field{
		field.Int("deep_analysis_id").Optional(),
		field.String("guide_content").Optional(),
	}
}

// Edges of the ActionGuide.
func (ActionGuide) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("deep_analysis_result", DeepAnalysisResult.Type).
			Ref("action_guides").
			Field("deep_analysis_id").
			Unique(),
	}
}
