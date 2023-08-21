package tests

import (
	"fmt"
	"github.com/fredyk/westack-go/westack/model"
	"io"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"

	wst "github.com/fredyk/westack-go/westack/common"
)

func Test_ExtractLookups(t *testing.T) {

	t.Parallel()

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
	assert.Equal(t, "John", (*lookups)[0]["$lookup"].(wst.M)["pipeline"].(wst.A)[2]["$match"].(wst.M)["name"])

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

func Test_CustomerOrderStore(t *testing.T) {
	// Create a customer with a random name, using math
	nameN := 1000000 + rand.Intn(8999999)
	name := fmt.Sprintf("Customer %v", nameN)

	customer := wst.M{
		"name": name,
		"age":  30,
		"address": wst.M{
			"street": "Main",
			"city":   "New York",
		},
	}
	createdCustomer, err := customerModel.Create(customer, systemContext)
	assert.Nil(t, err)
	assert.NotNil(t, createdCustomer)

	// Create a store with a random name
	storeNameN := 1000000 + rand.Intn(8999999)
	storeName := fmt.Sprintf("Store %v", storeNameN)

	store := wst.M{
		"name": storeName,
		"address": wst.M{
			"street": "Main",
			"city":   "New York",
		},
	}
	createdStore, err := storeModel.Create(store, systemContext)
	assert.Nil(t, err)
	assert.NotNil(t, createdStore)

	// Create 2 orders with amount
	order := wst.M{
		"amount":     131.43,
		"customerId": createdCustomer.Id,
		"storeId":    createdStore.Id,
	}
	createdOrder, err := orderModel.Create(order, systemContext)
	assert.Nil(t, err)
	assert.NotNil(t, createdOrder)
	order["amount"] = 123.45
	createdOrder, err = orderModel.Create(order, systemContext)
	assert.Nil(t, err)
	assert.NotNil(t, createdOrder)

	// Create other 10k orders
	for i := 0; i < 20000; i++ {
		order := wst.M{
			"amount":     rand.Float64() * 1000,
			"customerId": nil,
			"storeId":    nil,
		}
		cratedOrder, err := orderModel.Create(order, systemContext)
		assert.Nil(t, err)
		assert.NotNil(t, cratedOrder)
	}

	// Get the customer including the orders and the store
	filter := &wst.Filter{
		Where: &wst.Where{"name": name},
		Include: &wst.Include{
			{Relation: "orders", Scope: &wst.Filter{Include: &wst.Include{{Relation: "store"}}}},
		},
	}
	start := time.Now()
	customersCursor := customerModel.FindMany(filter, systemContext)
	assert.NotNil(t, customersCursor)
	customers, err := customersCursor.All()
	assert.Nil(t, err)
	assert.NotNil(t, customers)
	delayed := time.Since(start)
	assert.Greater(t, delayed.Milliseconds(), int64(6))
	fmt.Printf("\n===\nDELAYED without cache: %v\n===\n", delayed.Milliseconds())

	assert.Equal(t, 1, len(customers))
	assert.Equal(t, name, customers[0].ToJSON()["name"])
	assert.Equal(t, 2, len(customers[0].GetMany("orders")))
	assert.Equal(t, storeName, customers[0].GetMany("orders")[0].GetOne("store").ToJSON()["name"])

	//// Wait 1 second for the cache to be created
	time.Sleep(1 * time.Second)

	// Get memorykv stats with http
	stats := requestStats(t, err)

	// Check that the cache has been used, present at stats["stats"]["datasorces"]["memorykv"]["Order"]
	assert.Greater(t, int(stats["stats"].(map[string]interface{})["datasources"].(map[string]interface{})["memorykv"].(map[string]interface{})["Order"].(map[string]interface{})["entries"].(float64)), 0)
	// Exactly 1 miss, because the cache was empty
	assert.Equal(t, 1, int(stats["stats"].(map[string]interface{})["datasources"].(map[string]interface{})["memorykv"].(map[string]interface{})["Order"].(map[string]interface{})["misses"].(float64)))

	// Get the customer including the orders and the store, again
	start = time.Now()
	customersCursor = customerModel.FindMany(filter, systemContext)
	assert.NotNil(t, customersCursor)
	customers, err = customersCursor.All()
	assert.Nil(t, err)
	assert.NotNil(t, customers)
	prevDelayed := delayed
	delayed = time.Since(start)
	assert.LessOrEqual(t, delayed.Milliseconds(), prevDelayed.Milliseconds())
	fmt.Printf("\n===\nDELAYED with cache: %v\n===\n", delayed.Milliseconds())

	assert.Equal(t, 1, len(customers))

	// Request stats again
	stats = requestStats(t, err)

	// Check that the cache has been used, present at stats["stats"]["datasorces"]["memorykv"]["Order"], with more hits
	assert.GreaterOrEqual(t, int(stats["stats"].(map[string]interface{})["datasources"].(map[string]interface{})["memorykv"].(map[string]interface{})["Order"].(map[string]interface{})["hits"].(float64)), 1)

	// Wait 11 seconds for the cache to expire
	time.Sleep(11 * time.Second)

}

func Test_Aggregations(t *testing.T) {

	t.Parallel()

	// Check this aggregation:
	/*
		Aggregation: [
			{
				$addFields: {
					"userUsername": "$user.username"
				}
			}
		],
		Where: {
			userUsername: {$gt: ""}
		}
	*/

	firstUser, err := userModel.FindOne(nil, systemContext)
	assert.Nil(t, err)
	assert.NotNil(t, firstUser)

	note, err := noteModel.Create(wst.M{
		"title":  "Note 1",
		"userId": firstUser.Id,
	}, systemContext)
	assert.Nil(t, err)
	assert.NotNil(t, note)

	// Get the note including the user
	filter := &wst.Filter{
		Where: &wst.Where{"userUsername": wst.M{"$gt": ""}, "_id": note.Id},
		Aggregation: []wst.AggregationStage{
			{
				"$addFields": map[string]interface{}{
					"userUsername": "$user.username",
				},
			},
		},
		Include: &wst.Include{
			{
				Relation: "user",
				Scope: &wst.Filter{
					Where: &wst.Where{"username": firstUser.ToJSON()["username"]},
				},
			},
		},
		Skip:  0,
		Limit: 30,
	}

	notesCursor := noteModel.FindMany(filter, systemContext)
	assert.NotNil(t, notesCursor)
	notes, err := notesCursor.All()
	assert.Nil(t, err)
	assert.NotNil(t, notes)
	assert.Equal(t, 1, len(notes))
	assert.Equal(t, "Note 1", notes[0].ToJSON()["title"])
	assert.Equal(t, firstUser.ToJSON()["username"], notes[0].ToJSON()["userUsername"])

}

func Test_AggregationsWithDirectNestedQuery(t *testing.T) {

	t.Parallel()

	var randomeUserName string
	// assign randomUserName with safe random string
	var randomN int64
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomN = 100000000 + rnd.Int63n(899999999)
	randomeUserName = fmt.Sprintf("testuser%d", randomN)
	randomUser, err := userModel.Create(wst.M{
		"username": randomeUserName,
		"password": "abcd1234.",
	}, systemContext)
	assert.Nil(t, err)

	var randomNoteTitle string
	// assign randomNoteTitle with safe random string
	randomN = 100000000 + rnd.Int63n(899999999)
	randomNoteTitle = fmt.Sprintf("testnote%d", randomN)

	note, err := noteModel.Create(wst.M{
		"title":  randomNoteTitle,
		"userId": randomUser.Id,
	}, systemContext)
	assert.Nil(t, err)
	assert.NotNil(t, note)

	filter := &wst.Filter{
		Where: &wst.Where{
			"title":         note.GetString("title"),
			"user.username": randomUser.ToJSON()["username"],
		},
		Include: &wst.Include{
			{
				Relation: "user",
			},
		},
	}

	notesCursor := noteModel.FindMany(filter, systemContext)
	assert.NotNil(t, notesCursor)
	notes, err := notesCursor.All()
	assert.Nil(t, err)
	assert.NotNil(t, notes)
	assert.Equal(t, 1, len(notes))
	assert.Equal(t, note.GetString("title"), notes[0].GetString("title"))
	assert.Equal(t, randomUser.ToJSON()["username"], notes[0].GetOne("user").ToJSON()["username"])

}

func Test_AggregationsLimitAfterLookups(t *testing.T) {

	t.Parallel()

	firstUser, err := userModel.FindOne(nil, systemContext)
	assert.Nil(t, err)
	assert.NotNil(t, firstUser)

	// we are adding a field that does not exist in the model, so skip and limit should be applied after the stage
	filter := &wst.Filter{
		Where: &wst.Where{"userUsername": wst.M{"$gt": ""}},
		Aggregation: []wst.AggregationStage{
			{
				"$addFields": map[string]interface{}{
					"userUsername": "$user.username",
				},
			},
		},
		Include: &wst.Include{
			{
				Relation: "user",
				Scope: &wst.Filter{
					Where: &wst.Where{"username": firstUser.ToJSON()["username"]},
				},
			},
		},
		Skip:  60,
		Limit: 30,
	}

	notesCursor := noteModel.FindMany(filter, systemContext)
	assert.NotNil(t, notesCursor)
	_, err = notesCursor.All()
	assert.Nil(t, err)

	// check that the limit was applied after the stage
	pipeline := notesCursor.(*model.ChannelCursor).UsedPipeline
	// find index for $stage
	lookupIndex := -1
	for i, stage := range *pipeline {
		if stage["$lookup"] != nil {
			lookupIndex = i
			break
		}
	}
	assert.GreaterOrEqual(t, lookupIndex, 0)
	assert.EqualValues(t, 60, (*pipeline)[lookupIndex+4]["$skip"]) // +1 is $unwind. +2 is addFields, +3 is $match, +4 is $skip
	assert.EqualValues(t, 30, (*pipeline)[lookupIndex+5]["$limit"])
}

func Test_AggregationsLimitBeforeLookups(t *testing.T) {

	t.Parallel()

	firstUser, err := userModel.FindOne(nil, systemContext)
	assert.Nil(t, err)
	assert.NotNil(t, firstUser)

	// this time, we are not adding new fields so skip and limit should be applied before the $lookup stage
	filter := &wst.Filter{
		Aggregation: []wst.AggregationStage{
			{
				"$addFields": map[string]interface{}{
					"foo": "bar",
				},
			},
		},
		Include: &wst.Include{
			{
				Relation: "user",
				Scope: &wst.Filter{
					Where: &wst.Where{"username": firstUser.ToJSON()["username"]},
				},
			},
		},
		Skip:  90,
		Limit: 30,
	}

	notesCursor := noteModel.FindMany(filter, systemContext)
	assert.NotNil(t, notesCursor)
	_, err = notesCursor.All()
	assert.Nil(t, err)

	// check that the limit was applied before the $lookup stage
	pipeline := notesCursor.(*model.ChannelCursor).UsedPipeline
	// find index for $lookup
	lookupIndex := -1
	for i, stage := range *pipeline {
		if stage["$lookup"] != nil {
			lookupIndex = i
			break
		}
	}
	assert.GreaterOrEqual(t, lookupIndex, 0)
	assert.EqualValues(t, 90, (*pipeline)[lookupIndex-2]["$skip"]) // -3 is $match
	assert.EqualValues(t, 30, (*pipeline)[lookupIndex-1]["$limit"])

}

func Test_AggregationsWithInvalidDatasource(t *testing.T) {

	t.Parallel()

	filter := &wst.Filter{
		Aggregation: []wst.AggregationStage{
			{
				"$addFields": map[string]interface{}{
					"footer2Title": "$footer2.title",
				},
			},
		},
		Include: &wst.Include{
			{
				Relation: "footer2",
			},
		},
	}

	notesCursor := noteModel.FindMany(filter, systemContext)
	assert.NotNil(t, notesCursor)
	notes, err := notesCursor.All()
	assert.NotNil(t, err)

	assert.Nil(t, notes)

	// check that type of error is westack error *wst.WeStackError
	assert.Equal(t, "*wst.WeStackError", fmt.Sprintf("%T", err))

	// check that the error code is 400
	assert.Equal(t, 400, err.(*wst.WeStackError).FiberError.Code)

	// check that the error message is "Invalid datasource"
	assert.Equal(t, "related model Footer at relation footer2 belongs to another datasource", err.(*wst.WeStackError).Details["message"])

}

func requestStats(t *testing.T, err error) wst.M {
	resp, err := http.Get("http://localhost:8020/system/memorykv/stats")
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.NotNil(t, body)

	fmt.Printf("cache stats response: %v\n", string(body))
	stats := wst.M{}
	err = json.Unmarshal(body, &stats)
	assert.Nil(t, err)
	assert.NotNil(t, stats)
	return stats
}
