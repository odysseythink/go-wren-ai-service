package mdl

// MDL is the top-level Model Definition Language structure.
type MDL struct {
	Models        []Model        `json:"models"`
	Relationships []Relationship `json:"relationships"`
	Views         []View         `json:"views"`
	Metrics       []Metric       `json:"metrics"`
}

// Model represents a data model in MDL.
type Model struct {
	Name       string     `json:"name"`
	Properties Properties `json:"properties,omitempty"`
	Columns    []Column   `json:"columns"`
	PrimaryKey string     `json:"primaryKey"`
}

// Column represents a model column.
type Column struct {
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	Properties   Properties `json:"properties,omitempty"`
	Relationship string     `json:"relationship,omitempty"`
	Expression   string     `json:"expression,omitempty"`
	IsCalculated bool       `json:"isCalculated,omitempty"`
}

// Properties holds arbitrary key-value metadata.
type Properties map[string]any

// Relationship defines a relationship between models.
type Relationship struct {
	Condition string   `json:"condition"`
	JoinType  string   `json:"joinType"`
	Models    []string `json:"models"`
}

// View represents a saved SQL view.
type View struct {
	Name       string     `json:"name"`
	Statement  string     `json:"statement"`
	Properties Properties `json:"properties,omitempty"`
}

// Metric represents an aggregated metric definition.
type Metric struct {
	Name       string      `json:"name"`
	BaseObject string      `json:"baseObject"`
	Dimension  []MetricDim `json:"dimension,omitempty"`
	Measure    []MetricMeas `json:"measure,omitempty"`
	Properties Properties  `json:"properties,omitempty"`
}

// MetricDim is a metric dimension.
type MetricDim struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// MetricMeas is a metric measure.
type MetricMeas struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Expression string `json:"expression"`
}
