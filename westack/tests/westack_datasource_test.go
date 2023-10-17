package tests

import (
	"context"
	"fmt"
	wst "github.com/fredyk/westack-go/westack/common"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fredyk/westack-go/westack/datasource"
)

func Test_Datasource_Initialize_InvalidDatasource(t *testing.T) {

	t.Parallel()

	ds := datasource.New(&wst.IApp{}, "test2020", app.DsViper, context.Background())
	err := ds.Initialize()
	assert.Error(t, err)
	assert.Regexp(t, "invalid connector", err.Error(), "error message should be 'invalid connector'")

}

func Test_Datasource_Initialize_ConnectError(t *testing.T) {

	t.Parallel()

	ds := datasource.New(&wst.IApp{}, "db0", app.DsViper, context.Background())
	prevHost := ds.SubViper.GetString("url")
	ds.SubViper.Set("url", "<invalid url>")
	ds.Options = &datasource.Options{
		MongoDB: &datasource.MongoDBDatasourceOptions{
			Timeout: 3,
		},
	}
	err := ds.Initialize()
	assert.Error(t, err)
	assert.Regexp(t, `error parsing uri: scheme must be "mongodb" or "mongodb\+srv"`, err.Error(), "error message should be 'error parsing uri: scheme must be \"mongodb\" or \"mongodb+srv\"'")

	ds.SubViper.Set("url", prevHost)
	err = ds.Initialize()
	assert.NoError(t, err)

}

func Test_DatasourceClose(t *testing.T) {

	t.Parallel()

	ds, err := app.FindDatasource("db_expected_to_be_closed")
	assert.NoError(t, err)
	assert.NotNil(t, ds)

	err = ds.Close()
	assert.NoError(t, err)

	// Based on suggestion from @tanryberdi: https://github.com/fredyk/westack-go/pull/480#discussion_r1312634782
	// Attempt to perform a query. We don't mind the queried collection because the client
	// is disconnected anyway
	result, err := ds.FindMany("unknownCollection", nil)
	assert.Errorf(t, err, "client is disconnected")
	assert.Nil(t, result)

}

func Test_Datasource_Ping(t *testing.T) {

	t.Parallel()

	// Simply wait 3.2 seconds to cover datasource ping interval
	time.Sleep(3200 * time.Millisecond)

}

func Test_Datasource_Ping_Will_Fail(t *testing.T) {

	t.Parallel()

	db, err := app.FindDatasource("db_expected_to_fail")
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Wait 0.1 seconds, then change host and expect to fail
	time.Sleep(100 * time.Millisecond)

	db.SetTimeout(0.1)

	// Wait 5.1 seconds to cover datasource ping interval
	time.Sleep(5100 * time.Millisecond)

	err = db.Close()
	assert.NoError(t, err)

}

func Test_DatasourceDeleteManyNilWhere(t *testing.T) {

	t.Parallel()

	ds, err := app.FindDatasource("db0")
	assert.NoError(t, err)
	assert.NotNil(t, ds)

	result, err := ds.DeleteMany("InvalidModel", nil)
	assert.Error(t, err)
	assert.EqualValuesf(t, 0, result.DeletedCount, "result: %v", result)
	assert.EqualErrorf(t, err, "whereLookups cannot be nil", err.Error())

}

func Test_DatasourceDeleteManyMultipleLookups(t *testing.T) {

	t.Parallel()

	ds, err := app.FindDatasource("db0")
	assert.NoError(t, err)
	assert.NotNil(t, ds)

	result, err := ds.DeleteMany("InvalidModel", &wst.A{
		{},
		{},
	})
	assert.Error(t, err)
	assert.EqualValuesf(t, 0, result.DeletedCount, "result: %v", result)
	assert.EqualErrorf(t, err, "whereLookups must have exactly one element as a $match stage", err.Error())

}

func Test_DatasourceDeleteManyNilLookupEntry(t *testing.T) {

	t.Parallel()

	ds, err := app.FindDatasource("db0")
	assert.NoError(t, err)
	assert.NotNil(t, ds)

	result, err := ds.DeleteMany("InvalidModel", &wst.A{
		nil,
	})
	assert.Error(t, err)
	assert.EqualValuesf(t, 0, result.DeletedCount, "result: %v", result)
	assert.EqualErrorf(t, err, "whereLookups cannot have nil elements", err.Error())

}

func Test_DatasourceDeleteManyInvalidLookupEntry(t *testing.T) {

	t.Parallel()

	ds, err := app.FindDatasource("db0")
	assert.NoError(t, err)
	assert.NotNil(t, ds)

	result, err := ds.DeleteMany("InvalidModel", &wst.A{
		{"$foo": "bar"},
	})
	assert.Error(t, err)
	assert.EqualValuesf(t, 0, result.DeletedCount, "result: %v", result)
	assert.EqualErrorf(t, err, "first element of whereLookups must be a $match stage", err.Error())

}

func Test_DatasourceDeleteManyLookupEntryWithMultipleFields(t *testing.T) {

	t.Parallel()

	ds, err := app.FindDatasource("db0")
	assert.NoError(t, err)
	assert.NotNil(t, ds)

	result, err := ds.DeleteMany("InvalidModel", &wst.A{
		{"$match": "<unfound>", "$foo": "bar"},
	})
	assert.Error(t, err)
	assert.EqualValuesf(t, 0, result.DeletedCount, "result: %v", result)
	assert.EqualErrorf(t, err, "first element of whereLookups must be a single $match stage", err.Error())

}

func Test_DatasourceDeleteManyLookupEntryWithEmptyMatch(t *testing.T) {

	t.Parallel()

	ds, err := app.FindDatasource("db0")
	assert.NoError(t, err)
	assert.NotNil(t, ds)

	result, err := ds.DeleteMany("InvalidModel", &wst.A{
		{"$match": wst.M{}},
	})
	assert.Error(t, err)
	assert.EqualValuesf(t, 0, result.DeletedCount, "result: %v", result)
	assert.EqualErrorf(t, err, "first element of whereLookups must be a single and non-empty $match stage", err.Error())

}

func Test_DatasourceDeleteManyOK(t *testing.T) {

	t.Parallel()

	note1, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		"title": fmt.Sprintf("Note %v", createRandomInt()),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, note1, "id")

	note2, err := invokeApiAsRandomUser(t, "POST", "/notes", wst.M{
		"title": fmt.Sprintf("Note %v", createRandomInt()),
	}, wst.M{
		"Content-Type": "application/json",
	})
	assert.NoError(t, err)
	assert.Contains(t, note2, "id")

	result, err := noteModel.DeleteMany(&wst.Where{
		"_id": wst.M{
			"$in": []interface{}{
				note1.GetString("id"),
				note2.GetString("id"),
			},
		},
	}, systemContext)
	assert.NoError(t, err)
	assert.EqualValuesf(t, 2, result.DeletedCount, "result: %v", result)

}

func Test_ReplaceObjectIdsNil(t *testing.T) {

	t.Parallel()

	result, err := datasource.ReplaceObjectIds(nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func Test_ReplaceObjectIdsInvalidDate(t *testing.T) {

	t.Parallel()

	result, err := datasource.ReplaceObjectIds("2023-02-30T12:34:56.789Z")
	assert.NoError(t, err)
	assert.EqualValues(t, int64(-62135596800), result.(time.Time).Unix())
}

func Test_ReplaceObjectIdsInvalidInput(t *testing.T) {

	t.Parallel()

	data := make(chan int)
	result, err := datasource.ReplaceObjectIds(data)
	assert.NoError(t, err)
	assert.Equal(t, data, result)
}

func Test_ReplaceObjectInterfaceList(t *testing.T) {

	t.Parallel()

	data := wst.M{
		"foo": []interface{}{
			wst.M{
				"bar": "baz",
			},
		},
	}
	result, err := datasource.ReplaceObjectIds(data)
	assert.NoError(t, err)
	assert.EqualValues(t, data, result)
}

func Test_ReplaceObjectMList(t *testing.T) {

	t.Parallel()

	data := wst.M{
		"foo": []wst.M{
			{
				"bar": "baz",
			},
		},
	}
	result, err := datasource.ReplaceObjectIds(data)
	assert.NoError(t, err)
	assert.EqualValues(t, data, result)
}
