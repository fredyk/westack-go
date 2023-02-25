package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	wst "github.com/fredyk/westack-go/westack/common"
)

func Test_AFromGenericSlice_ValidEntries(t *testing.T) {
	slice := &[]interface{}{wst.M{"a": 1}, wst.M{"a": 2}, wst.M{"a": 3}}
	var a = *wst.AFromGenericSlice(slice)
	assert.Equal(t, 3, len(a))
	assert.Equal(t, 1, a[0]["a"])
	assert.Equal(t, 2, a[1]["a"])
	assert.Equal(t, 3, a[2]["a"])
}

func Test_AFromGenericSlice_NilInput(t *testing.T) {
	var a = wst.AFromGenericSlice(nil)
	assert.Nil(t, a)
}

func Test_AFromGenericSlice_WrongType(t *testing.T) {
	slice := &[]interface{}{wst.M{"a": 1}, map[string]interface{}{"a": 3}}
	var a = *wst.AFromGenericSlice(slice)
	assert.Equal(t, 2, len(a))
	assert.Equal(t, 1, a[0]["a"])
	// Ensure the second 1 is an empty wst.M
	assert.Equal(t, 0, len(a[1]))
}

func Test_AFromPrimitiveSlice_ValidEntries(t *testing.T) {
	var slice *primitive.A = &primitive.A{primitive.M{"a": 1}, primitive.M{"a": 2}, primitive.M{"a": 3}}
	var a = *wst.AFromPrimitiveSlice(slice)
	assert.Equal(t, 3, len(a))
	assert.Equal(t, 1, a[0]["a"])
	assert.Equal(t, 2, a[1]["a"])
	assert.Equal(t, 3, a[2]["a"])
}

func Test_AFromPrimitiveSlice_ValidEntriesAsM(t *testing.T) {
	var slice *primitive.A = &primitive.A{wst.M{"a": 1}, wst.M{"a": 2}, wst.M{"a": 3}}
	var a = *wst.AFromPrimitiveSlice(slice)
	assert.Equal(t, 3, len(a))
	assert.Equal(t, 1, a[0]["a"])
	assert.Equal(t, 2, a[1]["a"])
	assert.Equal(t, 3, a[2]["a"])
}

func Test_AFromPrimitiveSlice_NilInput(t *testing.T) {
	var a = wst.AFromPrimitiveSlice(nil)
	assert.Nil(t, a)
}

func Test_AFromPrimitiveSlice_WrongType(t *testing.T) {
	var slice *primitive.A = &primitive.A{primitive.M{"a": 1}, wst.M{"a": 2}, map[string]interface{}{"a": 3}}
	var a = *wst.AFromPrimitiveSlice(slice)
	assert.Equal(t, 3, len(a))
	assert.Equal(t, 1, a[0]["a"])
	assert.Equal(t, 2, a[1]["a"])
	// Ensure the second 1 is an empty wst.M
	assert.Equal(t, 0, len(a[2]))
}
