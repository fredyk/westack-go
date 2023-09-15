package tests

import (
	"github.com/fredyk/westack-go/westack/memorykv"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_MemoryKv(t *testing.T) {

	t.Parallel()

	db := memorykv.NewMemoryKvDb(memorykv.Options{
		Name: "exampleMemoryKv",
	})
	bucket := db.GetBucket("testBucket")
	err := bucket.SetEx("foo", [][]byte{[]byte("bar")}, 3*time.Second)
	assert.NoError(t, err)
	val, err := bucket.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{[]byte("bar")}, val)
	// set again
	bucket.Set("foo", [][]byte{[]byte("bar2")})
	val, err = bucket.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{[]byte("bar2")}, val)
	// set other
	err = bucket.SetEx("foo2", [][]byte{[]byte("bar2")}, 5*time.Second)
	assert.NoError(t, err)
	// and other
	err = bucket.SetEx("foo3", [][]byte{[]byte("bar3")}, 7*time.Second)
	assert.NoError(t, err)

	time.Sleep(4 * time.Second)
	val, err = bucket.Get("foo")
	assert.NoError(t, err)
	assert.Nil(t, val)

}

func Test_MemoryKvExpireInvalid(t *testing.T) {

	t.Parallel()

	db := memorykv.NewMemoryKvDb(memorykv.Options{
		Name: "exampleMemoryKv",
	})
	bucket := db.GetBucket("testBucket")
	err := bucket.Expire("foo", 3*time.Second)
	assert.EqualError(t, err, "key not found")
}

func Test_MemoryKvDelete(t *testing.T) {

	t.Parallel()

	db := memorykv.NewMemoryKvDb(memorykv.Options{
		Name: "exampleMemoryKv",
	})
	bucket := db.GetBucket("testBucket")
	err := bucket.SetEx("foo", [][]byte{[]byte("bar")}, 3*time.Second)
	assert.NoError(t, err)
	val, err := bucket.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{[]byte("bar")}, val)

	err = bucket.Delete("foo")
	assert.NoError(t, err)
	val, err = bucket.Get("foo")
	assert.NoError(t, err)
	assert.Nil(t, val)

}

func Test_MemoryKvFlush(t *testing.T) {

	t.Parallel()

	db := memorykv.NewMemoryKvDb(memorykv.Options{
		Name: "exampleMemoryKv",
	})
	bucket := db.GetBucket("testBucket")
	err := bucket.SetEx("foo", [][]byte{[]byte("bar")}, 3*time.Second)
	assert.NoError(t, err)

	bucket.Flush()
	assert.NoError(t, err)

}

func Test_MemoryKvPurge(t *testing.T) {

	t.Parallel()

	db := memorykv.NewMemoryKvDb(memorykv.Options{
		Name: "exampleMemoryKv",
	})
	bucket := db.GetBucket("testBucket")
	err := bucket.SetEx("foo", [][]byte{[]byte("bar")}, 3*time.Second)
	assert.NoError(t, err)

	err = db.Purge()
	assert.NoError(t, err)

}

func Test_MemoryKvReuseBucket(t *testing.T) {

	t.Parallel()

	db := memorykv.NewMemoryKvDb(memorykv.Options{
		Name: "exampleMemoryKv",
	})
	bucket := db.GetBucket("testBucket")
	err := bucket.SetEx("foo", [][]byte{[]byte("bar")}, 3*time.Second)
	assert.NoError(t, err)

	bucket = db.GetBucket("testBucket")
	val, err := bucket.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, [][]byte{[]byte("bar")}, val)

}
