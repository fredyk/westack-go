package entities

type M map[string]any

type Where M
type AggregationStage M

type IncludeItem struct {
	Relation string  `json:"relation"`
	Scope    *Filter `json:"scope"`
}

type Include []IncludeItem
type Order []string
type Fields []string

type Filter struct {
	Where       *Where             `json:"where"`
	Include     *Include           `json:"include"`
	Order       *Order             `json:"order"`
	Fields      *Fields            `json:"fields"`
	Skip        int64              `json:"skip"`
	Limit       int64              `json:"limit"`
	Aggregation []AggregationStage `json:"aggregation"`
}
