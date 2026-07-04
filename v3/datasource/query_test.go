package datasource

import "testing"

func TestFilter_DefaultLogicalOp(t *testing.T) {
	f := Filter{
		Conditions: []Condition{
			{Field: "name", Op: "eq", Value: "Alice"},
			{Field: "age", Op: "gt", Value: 18},
		},
	}
	if f.LogicalOp != "" {
		t.Errorf("Default LogicalOp should be empty (and/or), got %q", f.LogicalOp)
	}
}

func TestQuery_WithLimitOffsetSort(t *testing.T) {
	q := Query{
		Collection: "users",
		Filter: &Filter{
			Conditions: []Condition{
				{Field: "active", Op: "eq", Value: true},
			},
		},
		Sort:   []SortField{{Field: "created_at", Order: "desc"}},
		Limit:  10,
		Offset: 20,
	}
	if q.Limit != 10 {
		t.Errorf("Limit = %d, want 10", q.Limit)
	}
	if q.Offset != 20 {
		t.Errorf("Offset = %d, want 20", q.Offset)
	}
	if len(q.Sort) != 1 || q.Sort[0].Field != "created_at" {
		t.Error("Sort not set correctly")
	}
}

func TestQuery_WithJoin(t *testing.T) {
	q := Query{
		Collection: "expedientes",
		Joins: []Join{
			{
				Collection: "clientes",
				Type:       "inner",
				On:         JoinCondition{LeftField: "cliente_id", RightField: "id"},
			},
		},
	}
	if len(q.Joins) != 1 {
		t.Fatalf("Expected 1 join, got %d", len(q.Joins))
	}
	if q.Joins[0].On.LeftField != "cliente_id" {
		t.Errorf("Join LeftField = %q, want cliente_id", q.Joins[0].On.LeftField)
	}
}

func TestCondition_SubField(t *testing.T) {
	c := Condition{Field: "address", Op: "eq", SubField: "city", Value: "Madrid"}
	if c.SubField != "city" {
		t.Errorf("SubField = %q, want city", c.SubField)
	}
}
