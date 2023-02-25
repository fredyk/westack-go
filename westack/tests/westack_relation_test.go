package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"

	wst "github.com/fredyk/westack-go/westack/common"
)

func Test_ExtractLookups(t *testing.T) {

	// test nil filter
	lookups, err := noteModel.ExtractLookupsFromFilter(nil, false)
	assert.Nil(t, err)
	assert.Nil(t, lookups)

	// test empty filter
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{}, false)
	assert.Nil(t, err)
	// assert empty lookups
	assert.Equal(t, 0, len(*lookups))

	// test filter with order asc
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Order: &wst.Order{"title ASC"},
	}, false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(*lookups))
	assert.Contains(t, (*lookups)[0], "$sort")
	assert.Equal(t, (*lookups)[0]["$sort"].(bson.D)[0].Key, "title")
	assert.Equal(t, (*lookups)[0]["$sort"].(bson.D)[0].Value, 1)

	// test filter with order desc
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Order: &wst.Order{"created DESC"},
	}, false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(*lookups))
	assert.Contains(t, (*lookups)[0], "$sort")
	assert.Equal(t, (*lookups)[0]["$sort"].(bson.D)[0].Key, "created")
	assert.Equal(t, (*lookups)[0]["$sort"].(bson.D)[0].Value, -1)

	// test filter with invalid order
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Order: &wst.Order{"created INVALID-DIRECTION"},
	}, false)
	assert.NotNil(t, err)

	// test skip
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Skip: 10,
	}, false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(*lookups))
	assert.Contains(t, (*lookups)[0], "$skip")
	assert.Equal(t, int64(10), (*lookups)[0]["$skip"])

	// test include
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Include: &wst.Include{{Relation: "user"}},
	}, false)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(*lookups))
	assert.Contains(t, (*lookups)[0], "$lookup")
	assert.Equal(t, "user", (*lookups)[0]["$lookup"].(wst.M)["from"])
	assert.Equal(t, "user", (*lookups)[0]["$lookup"].(wst.M)["as"])
	assert.Equal(t, "$userId", (*lookups)[0]["$lookup"].(wst.M)["let"].(wst.M)["userId"])
	assert.Equal(t, "$_id", (*lookups)[0]["$lookup"].(wst.M)["pipeline"].(wst.A)[0]["$match"].(wst.M)["$expr"].(wst.M)["$and"].(wst.A)[0]["$eq"].([]string)[0])
	assert.Equal(t, "$$userId", (*lookups)[0]["$lookup"].(wst.M)["pipeline"].(wst.A)[0]["$match"].(wst.M)["$expr"].(wst.M)["$and"].(wst.A)[0]["$eq"].([]string)[1])

	assert.Contains(t, (*lookups)[1], "$unwind")
	assert.Equal(t, "$user", (*lookups)[1]["$unwind"].(wst.M)["path"])
	assert.Equal(t, true, (*lookups)[1]["$unwind"].(wst.M)["preserveNullAndEmptyArrays"])

	// test include with invalid relation 1
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Include: &wst.Include{{Relation: "invalid"}},
	}, false)
	assert.NotNil(t, err)

	// test include with invalid relation 2
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Include: &wst.Include{{Relation: "invalidRelation1"}},
	}, false)
	assert.NotNil(t, err)

	// test include with scope
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Include: &wst.Include{{Relation: "user", Scope: &wst.Filter{Where: &wst.Where{"name": "John"}}}},
	}, false)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(*lookups))
	assert.Contains(t, (*lookups)[0], "$lookup")
	assert.Equal(t, "user", (*lookups)[0]["$lookup"].(wst.M)["from"])
	assert.Equal(t, "user", (*lookups)[0]["$lookup"].(wst.M)["as"])
	assert.Equal(t, "$userId", (*lookups)[0]["$lookup"].(wst.M)["let"].(wst.M)["userId"])
	assert.Equal(t, "$_id", (*lookups)[0]["$lookup"].(wst.M)["pipeline"].(wst.A)[0]["$match"].(wst.M)["$expr"].(wst.M)["$and"].(wst.A)[0]["$eq"].([]string)[0])
	assert.Equal(t, "$$userId", (*lookups)[0]["$lookup"].(wst.M)["pipeline"].(wst.A)[0]["$match"].(wst.M)["$expr"].(wst.M)["$and"].(wst.A)[0]["$eq"].([]string)[1])
	assert.Equal(t, false, (*lookups)[0]["$lookup"].(wst.M)["pipeline"].(wst.A)[1]["$project"].(wst.M)["password"])
	//fmt.Printf("pipeline: %v\n", (*lookups)[0]["$lookup"].(wst.M)["pipeline"].(wst.A))
	assert.Equal(t, "John", (*lookups)[0]["$lookup"].(wst.M)["pipeline"].(wst.A)[2]["$match"].(wst.Where)["name"])

	assert.Contains(t, (*lookups)[1], "$unwind")
	assert.Equal(t, "$user", (*lookups)[1]["$unwind"].(wst.M)["path"])
	assert.Equal(t, true, (*lookups)[1]["$unwind"].(wst.M)["preserveNullAndEmptyArrays"])

	// test include hasMany
	lookups, err = userModel.ExtractLookupsFromFilter(&wst.Filter{
		Include: &wst.Include{{Relation: "notes"}},
	}, false)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(*lookups))
	assert.Contains(t, (*lookups)[0], "$lookup")
	assert.Equal(t, "Note", (*lookups)[0]["$lookup"].(wst.M)["from"])
	assert.Equal(t, "notes", (*lookups)[0]["$lookup"].(wst.M)["as"])
	assert.Equal(t, "$_id", (*lookups)[0]["$lookup"].(wst.M)["let"].(wst.M)["userId"])
	assert.Equal(t, "$userId", (*lookups)[0]["$lookup"].(wst.M)["pipeline"].(wst.A)[0]["$match"].(wst.M)["$expr"].(wst.M)["$and"].(wst.A)[0]["$eq"].([]string)[0])
	assert.Equal(t, "$$userId", (*lookups)[0]["$lookup"].(wst.M)["pipeline"].(wst.A)[0]["$match"].(wst.M)["$expr"].(wst.M)["$and"].(wst.A)[0]["$eq"].([]string)[1])

	// test invalid scope
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Include: &wst.Include{{Relation: "user", Scope: &wst.Filter{Include: &wst.Include{{Relation: "invalid"}}}}},
	}, false)
	assert.NotNil(t, err)

}
