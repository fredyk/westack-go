package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fredyk/westack-go/westack/datasource"
)

func Test_Datasource_Initialize_InvalidDatasource(t *testing.T) {

	t.Parallel()

	ds := datasource.New("test2020", app.DsViper, context.Background())
	err := ds.Initialize()
	assert.NotNil(t, err)
	assert.Regexp(t, "invalid connector", err.Error(), "error message should be 'invalid connector'")

}

func Test_Datasource_Initialize_ConnectError(t *testing.T) {

	t.Parallel()

	prevHost := app.DsViper.GetString("db.host")
	ds := datasource.New("db0", app.DsViper, context.Background())
	ds.SubViper.Set("host", "<invalid host>")
	ds.Options = &datasource.Options{
		MongoDB: &datasource.MongoDBDatasourceOptions{
			Timeout: 3,
		},
	}
	err := ds.Initialize()
	assert.NotNil(t, err)
	assert.Regexp(t, "no such host", err.Error(), "error message should be 'no such host'")

	ds.SubViper.Set("host", prevHost)
	err = ds.Initialize()
	assert.Nil(t, err)

}

func Test_Datasource_Ping(t *testing.T) {

	t.Parallel()

	// Simply wait 3.2 seconds to cover datasource ping interval
	time.Sleep(3200 * time.Millisecond)

}
