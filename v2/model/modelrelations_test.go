package model

import (
	"reflect"
	"testing"

	wst "github.com/fredyk/westack-go/v2/common"
)

func newTestModel() *StatefulModel {
	// Minimal model setup to call ExtractLookupsFromFilter safely
	reg := make(map[string]*StatefulModel)
	rels := make(map[string]*Relation)
	return &StatefulModel{
		App: &wst.IApp{
			Debug: false,
			Bson:  wst.BsonOptions{Registry: wst.CreateDefaultMongoRegistry()},
		},
		Config:        &Config{Relations: &rels},
		modelRegistry: &reg,
	}
}

func TestExtractLookups_IncludeNested(t *testing.T) {
	m := newTestModel()
	fields := wst.Fields{"profile.name", "age"}
	filter := &wst.Filter{Fields: &fields}

	lookups, err := m.ExtractLookupsFromFilter(filter, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if lookups == nil {
		t.Fatalf("lookups is nil")
	}

	if len(*lookups) != 1 {
		t.Fatalf("expected exactly 1 stage, got %d", len(*lookups))
	}

	stage := (*lookups)[0]
	proj, ok := stage["$project"].(wst.M)
	if !ok {
		t.Fatalf("expected $project stage, got: %v", stage)
	}

	expected := wst.M{
		"profile": wst.M{"name": "$profile.name"},
		"age":     1,
		"_id":     1,
	}
	if !reflect.DeepEqual(proj, expected) {
		t.Fatalf("unexpected $project.\nexpected: %#v\n     got: %#v", expected, proj)
	}
}

func TestExtractLookups_ExcludeFieldsAndId(t *testing.T) {
	m := newTestModel()
	exclude := wst.Fields{"secret", "profile.ssn", "_id"}
	filter := &wst.Filter{ExcludeFields: &exclude}

	lookups, err := m.ExtractLookupsFromFilter(filter, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lookups == nil {
		t.Fatalf("lookups is nil")
	}
	if len(*lookups) != 2 {
		t.Fatalf("expected exactly 2 stages ($project + $unset), got %d", len(*lookups))
	}

	stage0 := (*lookups)[0]
	proj, ok := stage0["$project"].(wst.M)
	if !ok {
		t.Fatalf("expected first stage to be $project, got: %v", stage0)
	}
	expProj := wst.M{"_id": 0}
	if !reflect.DeepEqual(proj, expProj) {
		t.Fatalf("unexpected first $project. expected: %#v got: %#v", expProj, proj)
	}

	stage1 := (*lookups)[1]
	unset := stage1["$unset"]
	slice, ok := unset.([]string)
	if !ok {
		// driver may decode as []interface{} depending on usage; handle both
		if asIface, ok2 := unset.([]interface{}); ok2 {
			got := make([]string, len(asIface))
			for i, v := range asIface {
				got[i] = v.(string)
			}
			slice = got
		} else {
			t.Fatalf("expected second stage to be $unset array, got: %T -> %v", unset, stage1)
		}
	}

	expectedUnset := []string{"secret", "profile.ssn"}
	if !reflect.DeepEqual(slice, expectedUnset) {
		t.Fatalf("unexpected $unset. expected: %#v got: %#v", expectedUnset, slice)
	}
}
