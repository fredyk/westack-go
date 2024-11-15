package tests

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/fredyk/westack-go/v2/model"
	"github.com/mailru/easyjson"
	"go.mongodb.org/mongo-driver/bson"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/stretchr/testify/assert"
)

func Test_ExtractLookups(t *testing.T) {

	t.Parallel()

	// test nil filter
	lookups, err := noteModel.ExtractLookupsFromFilter(nil, false)
	assert.NoError(t, err)
	assert.Nil(t, lookups)

	// test empty filter
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{}, false)
	assert.NoError(t, err)
	// assert empty lookups
	assert.Equal(t, 0, len(*lookups))

	// test filter with order asc
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Order: &wst.Order{"title ASC"},
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(*lookups))
	assert.Equal(t, "title", wst.GetTypedItem[bson.D](lookups.GetAt(0), "$sort")[0].Key)
	assert.Equal(t, 1, wst.GetTypedItem[bson.D](lookups.GetAt(0), "$sort")[0].Value)

	// test filter with order desc
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Order: &wst.Order{"created DESC"},
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(*lookups))
	assert.Equal(t, "created", wst.GetTypedItem[bson.D](lookups.GetAt(0), "$sort")[0].Key)
	assert.Equal(t, -1, wst.GetTypedItem[bson.D](lookups.GetAt(0), "$sort")[0].Value)

	// test filter with invalid order
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Order: &wst.Order{"created INVALID-DIRECTION"},
	}, false)
	assert.Error(t, err)
	assert.Nil(t, lookups)

	// test skip
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Skip: 10,
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(*lookups))
	assert.Equal(t, 10, lookups.GetAt(0).GetInt("$skip"))

	// test include
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Include: &wst.Include{{Relation: "account"}},
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(*lookups))
	assert.Equal(t, "account", lookups.GetM("[0].$lookup").GetString("from"))
	assert.Equal(t, "account", lookups.GetM("[0].$lookup").GetString("as"))
	assert.Equal(t, "$accountId", lookups.GetM("[0].$lookup.let").GetString("accountId"))
	assert.Equal(t, "$_id", wst.GetTypedList[string](lookups.GetM("[0].$lookup.pipeline.[0].$match.$expr.$and.[0]"), "$eq")[0])
	assert.Equal(t, "$$accountId", wst.GetTypedList[string](lookups.GetM("[0].$lookup.pipeline.[0].$match.$expr.$and.[0]"), "$eq")[1])

	assert.Equal(t, "$account", lookups.GetM("[1].$unwind").GetString("path"))
	assert.Equal(t, true, lookups.GetM("[1]").GetBoolean("$unwind.preserveNullAndEmptyArrays"))

	// test include with invalid relation 1
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Include: &wst.Include{{Relation: "invalid"}},
	}, false)
	assert.Error(t, err)
	assert.Nil(t, lookups)

	// test include with scope
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Include: &wst.Include{{Relation: "account", Scope: &wst.Filter{Where: &wst.Where{"name": "John"}}}},
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(*lookups))
	assert.Contains(t, *lookups.GetAt(0), "$lookup")
	assert.Equal(t, "Account", lookups.GetM("[0].$lookup").GetString("from"))
	assert.Equal(t, "account", lookups.GetM("[0].$lookup").GetString("as"))
	assert.Equal(t, "$accountId", lookups.GetM("[0].$lookup.let").GetString("accountId"))
	assert.Equal(t, "$_id", wst.GetTypedList[string](lookups.GetM("[0].$lookup.pipeline.[0].$match.$expr.$and.[0]"), "$eq")[0])
	assert.Equal(t, "$$accountId", wst.GetTypedList[string](lookups.GetM("[0].$lookup.pipeline.[0].$match.$expr.$and.[0]"), "$eq")[1])
	assert.Equal(t, false, lookups.GetM("[0].$lookup.pipeline.[1].$project").GetBoolean("password"))
	assert.Equal(t, "John", lookups.GetString("[0].$lookup.pipeline.[2].$match.name"))

	assert.Contains(t, *lookups.GetAt(1), "$unwind")
	assert.Equal(t, "$account", lookups.GetM("[1].$unwind").GetString("path"))
	assert.Equal(t, true, lookups.GetM("[1].$unwind").GetBoolean("preserveNullAndEmptyArrays"))

	// test include hasMany
	lookups, err = userModel.ExtractLookupsFromFilter(&wst.Filter{
		Include: &wst.Include{{Relation: "notes"}},
	}, false)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(*lookups))
	assert.Contains(t, *lookups.GetAt(0), "$lookup")
	assert.Equal(t, "Note", lookups.GetM("[0].$lookup").GetString("from"))
	assert.Equal(t, "notes", lookups.GetM("[0].$lookup").GetString("as"))
	assert.Equal(t, "$_id", lookups.GetM("[0].$lookup.let").GetString("accountId"))
	assert.Equal(t, "$accountId", wst.GetTypedList[string](lookups.GetM("[0].$lookup.pipeline.[0].$match.$expr.$and.[0]"), "$eq")[0])
	assert.Equal(t, "$$accountId", wst.GetTypedList[string](lookups.GetAt(0), "$lookup.pipeline.[0].$match.$expr.$and.[0].$eq")[1])

	// test invalid scope
	lookups, err = noteModel.ExtractLookupsFromFilter(&wst.Filter{
		Include: &wst.Include{{Relation: "account", Scope: &wst.Filter{Include: &wst.Include{{Relation: "invalid"}}}}},
	}, false)
	assert.Error(t, err)
	assert.Nil(t, lookups)

}

func Test_CustomerOrderStore(t *testing.T) {

	t.Parallel()

	// Create a customer with a random name, using math
	nameN := createRandomInt()
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
	assert.NoError(t, err)
	assert.NotNil(t, createdCustomer)

	// Create a store with a random name
	storeNameN := createRandomInt()
	storeName := fmt.Sprintf("Store %v", storeNameN)

	store := wst.M{
		"name": storeName,
		"address": wst.M{
			"street": "Main",
			"city":   "New York",
		},
	}
	createdStore, err := storeModel.Create(store, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, createdStore)

	// Create 2 orders with amount
	order := wst.M{
		"amount":     131.43,
		"customerId": createdCustomer.GetID(),
		"storeId":    createdStore.GetID(),
	}
	createdOrder, err := orderModel.Create(order, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, createdOrder)
	order["amount"] = 123.45
	createdOrder, err = orderModel.Create(order, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, createdOrder)

	// Create a waiting group using sync.WaitGroup
	var wg sync.WaitGroup
	// Create many orders
	orderCountToCreate := 200
	wg.Add(orderCountToCreate)
	creationInit := time.Now()
	for i := 0; i < orderCountToCreate; i++ {
		go func() {
			order := wst.M{
				"amount":     createRandomFloat(0, 1000.0),
				"customerId": nil,
				"storeId":    nil,
			}
			cratedOrder, err := orderModel.Create(order, systemContext)
			assert.NoError(t, err)
			assert.NotNil(t, cratedOrder)
			wg.Done()
		}()
	}
	wg.Wait()
	creationDelay := time.Since(creationInit)
	fmt.Printf("\n===\nCREATION DELAY: %v ms for %d orders\n===\n", creationDelay.Milliseconds(), orderCountToCreate)

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
	assert.NoError(t, err)
	assert.NotNil(t, customers)
	delayed := time.Since(start)
	assert.Greater(t, delayed.Milliseconds(), int64(0))
	fmt.Printf("\n===\nDELAYED without cache: %v\n===\n", delayed.Milliseconds())

	assert.Equal(t, 1, len(customers))
	assert.Equal(t, name, customers[0].ToJSON()["name"])
	assert.Equal(t, 2, len(customers[0].GetMany("orders")))
	assert.Equal(t, storeName, customers[0].GetMany("orders")[0].GetOne("store").ToJSON()["name"])

	//// Wait 1 second for the cache to be created
	time.Sleep(1 * time.Second)

	// Get memorykv stats with http
	stats := requestStats(t)

	// Check that the cache has been used, present at stats["stats"]["datasorces"]["memorykv"]["Order"]
	//allStats := stats["stats"].(map[string]interface{})
	//datasourcesStats := allStats["datasources"].(map[string]interface{})
	//memoryKvStats := datasourcesStats["memorykv"].(map[string]interface{})
	//orderStats := memoryKvStats["Order"].(map[string]interface{})
	//assert.Greater(t, int(orderStats["entries"].(float64)), 0)
	assert.Greater(t, stats.GetInt("stats.datasources.memorykv.Order.entries"), 0)
	// Exactly 1 miss, because the cache was empty
	//assert.Equal(t, 1, int(orderStats["misses"].(float64)))
	assert.EqualValues(t, 1, stats.GetInt("stats.datasources.memorykv.Order.misses"))

	// Get the customer including the orders and the store, again
	start = time.Now()
	customersCursor = customerModel.FindMany(filter, systemContext)
	assert.NotNil(t, customersCursor)
	customers, err = customersCursor.All()
	assert.NoError(t, err)
	assert.NotNil(t, customers)
	prevDelayed := delayed
	delayed = time.Since(start)
	assert.LessOrEqual(t, delayed.Milliseconds(), prevDelayed.Milliseconds())
	fmt.Printf("\n===\nDELAYED with cache: %v\n===\n", delayed.Milliseconds())

	assert.Equal(t, 1, len(customers))

	// Request stats again
	stats = requestStats(t)

	// Check that the cache has been used, present at stats["stats"]["datasorces"]["memorykv"]["Order"], with more hits
	//assert.GreaterOrEqual(t, int(stats["stats"].(map[string]interface{})["datasources"].(map[string]interface{})["memorykv"].(map[string]interface{})["Order"].(map[string]interface{})["hits"].(float64)), 1)
	assert.GreaterOrEqual(t, stats.GetInt("stats.datasources.memorykv.Order.hits"), 1)

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
					"userUsername": "$account.username"
				}
			}
		],
		Where: {
			userUsername: {$gt: ""}
		}
	*/

	randomUserName := fmt.Sprintf("testuser%d", createRandomInt())
	randomUser := createAccount(t, wst.M{
		"username":  randomUserName,
		"password":  "abcd1234.",
		"firstName": "John",
		"lastName":  "Doe",
	})

	noteTitle := "Note 1"
	note, err := noteModel.Create(wst.M{
		"title":     noteTitle,
		"accountId": randomUser.GetString("id"),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, note)

	// Get the note including the user
	filter := &wst.Filter{
		Where: &wst.Where{"userUsername": wst.M{"$gt": ""}, "_id": note.GetID()},
		Aggregation: []wst.AggregationStage{
			{
				"$addFields": map[string]interface{}{
					"userUsername": "$account.username",
					"fullUserName": map[string]interface{}{
						"$concat": []string{"$account.firstName", " ", "$account.lastName"},
					},
				},
			},
		},
		Include: &wst.Include{
			{
				Relation: "account",
				Scope: &wst.Filter{
					Where: &wst.Where{"username": randomUserName},
				},
			},
		},
		Skip:  0,
		Limit: 30,
	}

	notesCursor := noteModel.FindMany(filter, systemContext)
	assert.NotNil(t, notesCursor)
	notes, err := notesCursor.All()
	assert.NoError(t, err)
	assert.NotNil(t, notes)
	assert.Equal(t, 1, len(notes))
	assert.Equal(t, noteTitle, notes[0].ToJSON()["title"])
	assert.Equal(t, randomUserName, notes[0].ToJSON()["userUsername"])
	assert.Equal(t, "John Doe", notes[0].ToJSON()["fullUserName"])

}

func Test_AggregationsWithDirectNestedQuery(t *testing.T) {

	t.Parallel()

	randomNoteTitle := fmt.Sprintf("testnote%d", createRandomInt())

	note, err := noteModel.Create(wst.M{
		"title":     randomNoteTitle,
		"accountId": randomAccount.GetString("id"),
	}, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, note)

	filter := &wst.Filter{
		Where: &wst.Where{
			"title":            note.GetString("title"),
			"account.username": randomAccount.GetString("username"),
		},
		Include: &wst.Include{
			{
				Relation: "account",
			},
		},
	}

	notesCursor := noteModel.FindMany(filter, systemContext)
	assert.NotNil(t, notesCursor)
	notes, err := notesCursor.All()
	assert.NoError(t, err)
	assert.NotNil(t, notes)
	assert.Equal(t, 1, len(notes))
	assert.Equal(t, note.GetString("title"), notes[0].GetString("title"))
	assert.Equal(t, randomAccount.GetString("username"), notes[0].GetOne("account").ToJSON()["username"])

}

func Test_AggregationsLimitAfterLookups(t *testing.T) {

	t.Parallel()

	firstUser, err := userModel.FindOne(nil, systemContext)
	assert.NoError(t, err)
	assert.NotNil(t, firstUser)

	// we are adding a field that does not exist in the model, so skip and limit should be applied after the stage
	filter := &wst.Filter{
		Where: &wst.Where{"userUsername": wst.M{"$gt": ""}},
		Aggregation: []wst.AggregationStage{
			{
				"$addFields": map[string]interface{}{
					"userUsername": "$account.username",
				},
			},
		},
		Include: &wst.Include{
			{
				Relation: "account",
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
	assert.NoError(t, err)

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
	assert.NoError(t, err)
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
				Relation: "account",
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
	assert.NoError(t, err)

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
	assert.Error(t, err)

	assert.Nil(t, notes)

	// check that type of error is westack error *wst.WeStackError
	assert.Equal(t, "*wst.WeStackError", fmt.Sprintf("%T", err))

	// check that the error code is 400
	assert.Equal(t, 400, err.(*wst.WeStackError).FiberError.Code)

	// check that the error message is "Invalid datasource"
	assert.Equal(t, "related model Footer at relation footer2 belongs to another datasource", err.(*wst.WeStackError).Details["message"])

}

// Tries to fetch a relation that not exists
func Test_AggregationWithNonExistentRelation(t *testing.T) {

	t.Parallel()

	filter := &wst.Filter{
		Aggregation: []wst.AggregationStage{
			{
				"$addFields": map[string]interface{}{
					"footer2Title": "$invalidRelation.title",
				},
			},
		},
	}

	notesCursor := noteModel.FindMany(filter, systemContext)
	assert.NotNil(t, notesCursor)
	notes, err := notesCursor.All()
	assert.Error(t, err)

	assert.Nil(t, notes)

	// check that type of error is westack error *wst.WeStackError
	assert.Equal(t, "*wst.WeStackError", fmt.Sprintf("%T", err))

	// check that the error code is 400
	assert.Equal(t, 400, err.(*wst.WeStackError).FiberError.Code)

	// check that the error message matches format "relation %v not found for model %v"
	assert.Equal(t, fmt.Sprintf("relation %v not found for model %v", "invalidRelation", "Note"), err.(*wst.WeStackError).Details["message"])

}

func Test_AggregationWithInvalidStage(t *testing.T) {

	t.Parallel()

	filter := &wst.Filter{
		Aggregation: []wst.AggregationStage{
			{
				"$out": "SomeCollection",
			},
		},
	}

	notesCursor := noteModel.FindMany(filter, systemContext)
	assert.NotNil(t, notesCursor)
	notes, err := notesCursor.All()
	assert.Error(t, err)

	assert.Nil(t, notes)

	// check that type of error is westack error *wst.WeStackError
	assert.Equal(t, "*wst.WeStackError", fmt.Sprintf("%T", err))

	// check that the error code is 400
	assert.Equal(t, 400, err.(*wst.WeStackError).FiberError.Code)

	// check that the error message matches format "%s aggregation stage not allowed"
	assert.Equal(t, fmt.Sprintf("%s aggregation stage not allowed", "$out"), err.(*wst.WeStackError).Details["message"])

}

func requestStats(t *testing.T) wst.M {
	req, err := http.NewRequest("GET", "/system/memorykv/stats", nil)
	assert.NoError(t, err)
	resp, err := app.Server.Test(req, 45000)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	//goland:noinspection GoUnhandledErrorResult
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NotNil(t, body)

	fmt.Printf("cache stats response: %v\n", string(body))
	stats := wst.M{}
	err = easyjson.Unmarshal(body, &stats)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	return stats
}

func Test_RelationWithoutAuth(t *testing.T) {

	t.Parallel()

	// Create a note
	note, err := invokeApiAsRandomAccount(t, "POST", "/notes", wst.M{
		"title": "Note 1",
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, note, "id")

	// Create a footer
	footer, err := invokeApiAsRandomAccount(t, "POST", "/footers", wst.M{
		"title":        "Public Footer 1",
		"publicNoteId": note.GetString("id"),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, footer, "id")

	// Get the note including the footer
	noteWithFooter, err := invokeApiAsRandomAccount(t, "GET", "/notes/"+note.GetString("id")+"?filter=%7B%22include%22%3A%5B%7B%22relation%22%3A%22publicFooter%22%7D%5D%7D", nil, nil)
	assert.NoError(t, err)
	assert.Contains(t, noteWithFooter, "id")
	assert.Contains(t, noteWithFooter, "publicFooter")
	assert.Equal(t, footer.GetString("id"), noteWithFooter.GetM("publicFooter").GetString("id"))
	assert.Equal(t, footer.GetString("title"), noteWithFooter.GetM("publicFooter").GetString("title"))
}
