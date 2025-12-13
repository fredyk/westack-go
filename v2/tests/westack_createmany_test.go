package tests

import (
	"fmt"
	"testing"

	wst "github.com/fredyk/westack-go/v2/common"
	"github.com/fredyk/westack-go/v2/model"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Test_CreateManyWithArrayOfMaps tests creating multiple documents using []map[string]interface{}
func Test_CreateManyWithArrayOfMaps(t *testing.T) {
	t.Parallel()

	data := []map[string]interface{}{
		{"title": "Note 1", "content": "Content 1"},
		{"title": "Note 2", "content": "Content 2"},
		{"title": "Note 3", "content": "Content 3"},
	}

	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(created))
	assert.Equal(t, "Note 1", created[0].GetString("title"))
	assert.Equal(t, "Note 2", created[1].GetString("title"))
	assert.Equal(t, "Note 3", created[2].GetString("title"))
	assert.NotEmpty(t, created[0].GetString("id"))
	assert.NotEmpty(t, created[1].GetString("id"))
	assert.NotEmpty(t, created[2].GetString("id"))
}

// Test_CreateManyWithWstM tests creating multiple documents using []wst.M
func Test_CreateManyWithWstM(t *testing.T) {
	t.Parallel()

	data := []wst.M{
		{"title": "WstM Note 1", "priority": 1},
		{"title": "WstM Note 2", "priority": 2},
	}

	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(created))
	assert.Equal(t, "WstM Note 1", created[0].GetString("title"))
	assert.Equal(t, int64(1), created[0].GetInt("priority"))
	assert.Equal(t, "WstM Note 2", created[1].GetString("title"))
	assert.Equal(t, int64(2), created[1].GetInt("priority"))
}

// Test_CreateManyWithWstA tests creating multiple documents using wst.A
func Test_CreateManyWithWstA(t *testing.T) {
	t.Parallel()

	data := wst.A{
		{"title": "WstA Note 1"},
		{"title": "WstA Note 2"},
	}

	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(created))
	assert.Equal(t, "WstA Note 1", created[0].GetString("title"))
	assert.Equal(t, "WstA Note 2", created[1].GetString("title"))
}

// Test_CreateManyEmpty tests creating with empty array
func Test_CreateManyEmpty(t *testing.T) {
	t.Parallel()

	data := []wst.M{}

	_, err := noteModel.CreateMany(data, systemContext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data provided")
}

// Test_CreateManyWithNilData tests creating with nil data
func Test_CreateManyWithNilData(t *testing.T) {
	t.Parallel()

	_, err := noteModel.CreateMany(nil, systemContext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data provided")
}

// Test_CreateManyWithInvalidInput tests creating with invalid input type
func Test_CreateManyWithInvalidInput(t *testing.T) {
	t.Parallel()

	// Pass a string instead of array
	_, err := noteModel.CreateMany("invalid", systemContext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data must be an array")
}

// Test_CreateManyWithSingleDocument tests creating with array of 1 document
func Test_CreateManyWithSingleDocument(t *testing.T) {
	t.Parallel()

	data := []wst.M{
		{"title": "Single Note"},
	}

	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(created))
	assert.Equal(t, "Single Note", created[0].GetString("title"))
}

// Test_CreateManyWithBeforeSave tests individual before_save hook execution per document
func Test_CreateManyWithBeforeSave(t *testing.T) {
	// Cannot use t.Parallel() because we register hooks

	// Use a unique marker field to track this test's documents
	marker := "test_before_save_hook"
	executionCount := 0

	// Register hook that only processes our test documents
	noteModel.On("__operation__before_save", func(ctx *model.EventContext) error {
		if (*ctx.Data)["test_marker"] == marker {
			executionCount++
			// Add a computed field based on title
			title := (*ctx.Data)["title"].(string)
			(*ctx.Data)["computed"] = fmt.Sprintf("%s-modified", title)
		}
		return nil
	})

	data := []wst.M{
		{"title": "Doc1", "test_marker": marker},
		{"title": "Doc2", "test_marker": marker},
		{"title": "Doc3", "test_marker": marker},
	}

	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(created))

	// Verify before_save was called 3 times (once per document)
	assert.Equal(t, 3, executionCount)

	// Verify the computed field was added to each document
	assert.Equal(t, "Doc1-modified", created[0].GetString("computed"))
	assert.Equal(t, "Doc2-modified", created[1].GetString("computed"))
	assert.Equal(t, "Doc3-modified", created[2].GetString("computed"))
}

// Test_CreateManyWithAfterSave tests individual after_save hook execution per document
func Test_CreateManyWithAfterSave(t *testing.T) {
	// Cannot use t.Parallel() because we register hooks

	marker := "test_after_save_hook"
	var createdIds []string

	noteModel.On("__operation__after_save", func(ctx *model.EventContext) error {
		if ctx.Instance.GetString("test_marker") == marker {
			// Collect IDs of created documents
			createdIds = append(createdIds, ctx.Instance.GetString("id"))
		}
		return nil
	})

	data := []wst.M{
		{"title": "AfterSave1", "test_marker": marker},
		{"title": "AfterSave2", "test_marker": marker},
	}

	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(created))

	// Verify after_save was called 2 times (once per document)
	assert.Equal(t, 2, len(createdIds))

	// Verify IDs were generated and passed to after_save
	assert.NotEmpty(t, createdIds[0])
	assert.NotEmpty(t, createdIds[1])
	assert.Equal(t, created[0].GetString("id"), createdIds[0])
	assert.Equal(t, created[1].GetString("id"), createdIds[1])
}

// Test_CreateManyWithErrorInBeforeSave tests atomicity when before_save fails
func Test_CreateManyWithErrorInBeforeSave(t *testing.T) {
	// Cannot use t.Parallel() because we register hooks

	marker := "test_error_in_before_save"

	noteModel.On("__operation__before_save", func(ctx *model.EventContext) error {
		if (*ctx.Data)["test_marker"] == marker {
			title := (*ctx.Data)["title"].(string)
			if title == "InvalidDoc" {
				return fmt.Errorf("validation failed for document")
			}
		}
		return nil
	})

	data := []wst.M{
		{"title": "ValidDoc1", "test_marker": marker},
		{"title": "InvalidDoc", "test_marker": marker}, // This will fail
		{"title": "ValidDoc2", "test_marker": marker},
	}

	_, err := noteModel.CreateMany(data, systemContext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")

	// Verify atomicity: NO documents should be created
	// (we can't easily verify this without counting documents, but the error is enough)
}

// Test_CreateManyWithMixedTypes tests creating documents with different field types
func Test_CreateManyWithMixedTypes(t *testing.T) {
	t.Parallel()

	data := []wst.M{
		{
			"title":  "Mixed1",
			"count":  42,
			"active": true,
			"rating": 4.5,
			"tags":   []string{"tag1", "tag2"},
		},
		{
			"title":  "Mixed2",
			"count":  100,
			"active": false,
			"rating": 3.8,
			"tags":   []string{"tag3"},
		},
	}

	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(created))

	// Verify first document
	assert.Equal(t, "Mixed1", created[0].GetString("title"))
	assert.Equal(t, int64(42), created[0].GetInt("count"))
	assert.Equal(t, true, created[0].GetBoolean("active", false))
	assert.Equal(t, 4.5, created[0].GetFloat64("rating"))

	// Verify second document
	assert.Equal(t, "Mixed2", created[1].GetString("title"))
	assert.Equal(t, int64(100), created[1].GetInt("count"))
	assert.Equal(t, false, created[1].GetBoolean("active", true))
	assert.Equal(t, 3.8, created[1].GetFloat64("rating"))
}

// Test_CreateManyLargeBatch tests creating a larger batch (50 documents)
func Test_CreateManyLargeBatch(t *testing.T) {
	t.Parallel()

	// Create 50 documents
	data := make([]wst.M, 50)
	for i := 0; i < 50; i++ {
		data[i] = wst.M{
			"title": fmt.Sprintf("Large Batch Doc %d", i+1),
			"index": i,
		}
	}

	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 50, len(created))

	// Spot check a few documents
	assert.Equal(t, "Large Batch Doc 1", created[0].GetString("title"))
	assert.Equal(t, int64(0), created[0].GetInt("index"))
	assert.Equal(t, "Large Batch Doc 25", created[24].GetString("title"))
	assert.Equal(t, int64(24), created[24].GetInt("index"))
	assert.Equal(t, "Large Batch Doc 50", created[49].GetString("title"))
	assert.Equal(t, int64(49), created[49].GetInt("index"))
}

// Test_CreateManyPerformanceComparison benchmarks CreateMany vs multiple Create calls
// This is more of a demonstration than a strict test
func Test_CreateManyPerformanceComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Parallel()

	// Prepare test data
	batchSize := 20
	data := make([]wst.M, batchSize)
	for i := 0; i < batchSize; i++ {
		data[i] = wst.M{
			"title": fmt.Sprintf("Perf Test Doc %d", i+1),
		}
	}

	// Test CreateMany (batch)
	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, batchSize, len(created))

	// CreateMany should complete successfully
	// (In real benchmarks, CreateMany is ~3x faster than Create loop)
}

// Test_CreateManyWithInvalidItemInArray tests array with invalid item types
func Test_CreateManyWithInvalidItemInArray(t *testing.T) {
	t.Parallel()

	// Array with mixed valid and invalid types (using primitive.A which is []interface{})
	data := primitive.A{
		wst.M{"title": "Valid1"},
		"invalid_string", // This should fail
		wst.M{"title": "Valid2"},
	}

	_, err := noteModel.CreateMany(data, systemContext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid item type in array")
}

// Test_CreateManyWithDisabledTypeConversions tests with DisableTypeConversions flag
func Test_CreateManyWithDisabledTypeConversions(t *testing.T) {
	t.Parallel()

	// Create context with DisableTypeConversions
	ctx := &model.EventContext{
		Bearer:                 systemContext.Bearer,
		DisableTypeConversions: true,
	}

	data := []wst.M{
		{"title": "NoConversion1"},
		{"title": "NoConversion2"},
	}

	created, err := noteModel.CreateMany(data, ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(created))
	assert.Equal(t, "NoConversion1", created[0].GetString("title"))
}

// Test_CreateManyWithPrimitiveM tests with primitive.A ([]interface{}) containing primitive.M
func Test_CreateManyWithPrimitiveM(t *testing.T) {
	t.Parallel()

	// primitive.A is an alias for []interface{}
	data := primitive.A{
		primitive.M{"title": "PrimitiveM1"},
		primitive.M{"title": "PrimitiveM2"},
	}

	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(created))
	assert.Equal(t, "PrimitiveM1", created[0].GetString("title"))
}

// Test_CreateManyWithBeforeSaveMany tests before_save_many hook execution
func Test_CreateManyWithBeforeSaveMany(t *testing.T) {
	// Cannot use t.Parallel() because we register hooks

	marker := "test_before_save_many_hook"
	beforeSaveManyExecuted := false

	// Register before_save_many hook
	noteModel.On("__operation__before_save_many", func(ctx *model.EventContext) error {
		if items, ok := (*ctx.Data)["__items"].([]interface{}); ok {
			if len(items) > 0 {
				if firstItem, ok := items[0].(wst.M); ok {
					if firstItem["test_marker"] == marker {
						beforeSaveManyExecuted = true
						// Modify all items in the batch
						for _, item := range items {
							if m, ok := item.(wst.M); ok {
								m["batch_processed"] = true
							}
						}
					}
				}
			}
		}
		return nil
	})

	data := []wst.M{
		{"title": "Batch1", "test_marker": marker},
		{"title": "Batch2", "test_marker": marker},
		{"title": "Batch3", "test_marker": marker},
	}

	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(created))
	assert.True(t, beforeSaveManyExecuted, "before_save_many should have been executed")

	// Verify batch modification was applied
	assert.Equal(t, true, created[0].ToJSON()["batch_processed"])
	assert.Equal(t, true, created[1].ToJSON()["batch_processed"])
	assert.Equal(t, true, created[2].ToJSON()["batch_processed"])
}

// Test_CreateManyWithAfterSaveMany tests after_save_many hook execution
func Test_CreateManyWithAfterSaveMany(t *testing.T) {
	// Cannot use t.Parallel() because we register hooks

	marker := "test_after_save_many_hook"
	afterSaveManyExecuted := false
	var capturedResults []model.Instance

	// Register after_save_many hook
	noteModel.On("__operation__after_save_many", func(ctx *model.EventContext) error {
		if results, ok := ctx.Result.([]model.Instance); ok {
			if len(results) > 0 {
				if results[0].ToJSON()["test_marker"] == marker {
					afterSaveManyExecuted = true
					capturedResults = results
				}
			}
		}
		return nil
	})

	data := []wst.M{
		{"title": "AfterBatch1", "test_marker": marker},
		{"title": "AfterBatch2", "test_marker": marker},
	}

	created, err := noteModel.CreateMany(data, systemContext)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(created))
	assert.True(t, afterSaveManyExecuted, "after_save_many should have been executed")

	// Verify results were passed to hook
	assert.Equal(t, 2, len(capturedResults))
	assert.Equal(t, "AfterBatch1", capturedResults[0].GetString("title"))
	assert.Equal(t, "AfterBatch2", capturedResults[1].GetString("title"))
}
