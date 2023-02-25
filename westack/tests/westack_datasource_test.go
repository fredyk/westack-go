package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fredyk/westack-go/westack/datasource"
)

func Test_Datasource_Initialize_InvalidConnector(t *testing.T) {

	ds := datasource.New("test2020", app.DsViper, context.Background())
	err := ds.Initialize()
	assert.NotNil(t, err)
	assert.Regexp(t, "invalid connector", err.Error(), "error message should be 'invalid connector'")

}

func Test_Datasource_Initialize_ConnectError(t *testing.T) {

	prevHost := app.DsViper.GetString("db.host")
	app.DsViper.Set("db.host", "<invalid host>")
	ds := datasource.New("db", app.DsViper, context.Background())
	ds.Options = &datasource.Options{
		MongoDB: &datasource.MongoDBDatasourceOptions{
			Timeout: 3,
		},
	}
	err := ds.Initialize()
	assert.NotNil(t, err)
	assert.Regexp(t, "no such host", err.Error(), "error message should be 'no such host'")

	app.DsViper.Set("db.host", prevHost)
	err = ds.Initialize()
	assert.Nil(t, err)

}
