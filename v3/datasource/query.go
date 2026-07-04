package datasource

// Filter represents a generic where/condition clause agnostic of any query language.
type Filter struct {
	// Conditions is a list of field-level predicates (e.g. field="name", op="eq", value="John").
	Conditions []Condition
	// LogicalOp is the conjunction between conditions: "and" (default) or "or".
	LogicalOp string
}

// Condition is a single predicate on a field.
type Condition struct {
	Field    string
	Op       string // eq, neq, gt, gte, lt, lte, in, nin, like, is_null
	Value    interface{}
	SubField string // for nested fields (e.g. "address.city")
}

// SortField defines a single sort entry.
type SortField struct {
	Field string
	Order string // "asc" or "desc"
}

// Query is a generic query representation replacing the BSON pipeline wst.A.
// It is agnostic of Mongo, Postgres, or any specific backend.
type Query struct {
	Collection string
	Filter     *Filter
	Fields     []string  // projected fields (nil = all)
	Sort       []SortField
	Limit      int64
	Offset     int64
	Joins      []Join
}

// Join represents a relational join.
type Join struct {
	Collection string
	Type       string // inner, left, right
	On         JoinCondition
}

// JoinCondition defines the ON clause for a join.
type JoinCondition struct {
	LeftField  string
	RightField string
}
