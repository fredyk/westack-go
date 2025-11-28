package tests

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fredyk/westack-go/v2/model"
	"github.com/stretchr/testify/assert"
	wst "github.com/fredyk/westack-go/v2/common"
)

// TestQueueOperationIsolationInParallelCreates tests that queued operations
// are isolated between parallel Create operations. This simulates the production
// scenario where multiple goroutines create instances concurrently and each
// queues operations that should only execute for their own instance.
func TestQueueOperationIsolationInParallelCreates(t *testing.T) {
	// Use existing noteModel from the test suite
	if noteModel == nil {
		t.Skip("noteModel not initialized")
		return
	}

	// Track which operations were queued and executed
	var queuedOps sync.Map     // executionId -> title
	var executedOps sync.Map   // executionId -> title
	var executionCount int32
	var mismatchCount int32
	var hookId = fmt.Sprintf("test-hook-%d", time.Now().UnixNano())

	// Setup temporary hook to queue operations
	noteModel.Observe("before save", func(ctx *model.EventContext) error {
		if ctx.IsNewInstance && ctx.Data.GetString("__testHookId") == hookId {
			title := ctx.Data.GetString("title")
			executionId := model.FindBaseContext(ctx).ExecutionId
			
			// Store that this operation was queued
			queuedOps.Store(executionId, title)

			// Queue an operation for "after save"
			// This simulates the production code pattern where we capture variables
			// and need them to match when the queued operation executes
			ctx.QueueOperation("after save", func(nextCtx *model.EventContext) error {
				instanceTitle := nextCtx.Instance.GetString("title")
				
				// Store execution info
				executedOps.Store(executionId, instanceTitle)
				atomic.AddInt32(&executionCount, 1)
				
				// Verify the instance matches the captured variable
				// With ExecutionId isolation, this should NEVER mismatch
				if instanceTitle != title {
					atomic.AddInt32(&mismatchCount, 1)
					return fmt.Errorf("MISMATCH: expected title='%s' but got title='%s' (ExecutionId: %s)", 
						title, instanceTitle, executionId)
				}
				
				return nil
			})
		}
		return nil
	})

	systemContext := &model.EventContext{}

	// Create multiple notes in parallel (simulates concurrent Lambda updates)
	concurrentOps := 50
	var wg sync.WaitGroup
	wg.Add(concurrentOps)

	errors := make(chan error, concurrentOps)

	for i := 0; i < concurrentOps; i++ {
		go func(idx int) {
			defer wg.Done()

			title := fmt.Sprintf("test-note-%d-%d", idx, time.Now().UnixNano())
			_, err := noteModel.Create(wst.M{
				"title":        title,
				"body":         fmt.Sprintf("body-%d", idx),
				"__testHookId": hookId,
			}, systemContext)

			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors during creation
	for err := range errors {
		assert.NoError(t, err)
	}

	// Wait for all queued operations to execute
	time.Sleep(500 * time.Millisecond)

	// Verify that all queued operations were executed
	execCount := atomic.LoadInt32(&executionCount)
	assert.Equal(t, int32(concurrentOps), execCount, 
		"All queued operations should have been executed")

	// THE KEY ASSERTION: No mismatches should occur with ExecutionId isolation
	mismatchCnt := atomic.LoadInt32(&mismatchCount)
	assert.Equal(t, int32(0), mismatchCnt, 
		"With ExecutionId isolation, NO mismatches should occur between captured variables and executed context")

	// Verify that each execution matched its queued operation
	queuedOps.Range(func(key, value interface{}) bool {
		executionId := key.(string)
		queuedTitle := value.(string)
		
		executedTitle, ok := executedOps.Load(executionId)
		assert.True(t, ok, 
			"Queued operation should have been executed for ExecutionId: %s", executionId)
		assert.Equal(t, queuedTitle, executedTitle, 
			"Executed operation should match queued operation for ExecutionId: %s", executionId)
		
		return true
	})
}

// TestQueueOperationExecutionIdUniqueness tests that each operation gets a unique ExecutionId
// even when operations run in parallel
func TestQueueOperationExecutionIdUniqueness(t *testing.T) {
	if noteModel == nil {
		t.Skip("noteModel not initialized")
		return
	}

	// Collect all ExecutionIds seen
	var executionIds sync.Map
	var duplicateFound int32
	var hookId = fmt.Sprintf("test-exec-id-%d", time.Now().UnixNano())

	noteModel.Observe("before save", func(ctx *model.EventContext) error {
		if ctx.IsNewInstance && ctx.Data.GetString("__testHookId") == hookId {
			executionId := model.FindBaseContext(ctx).ExecutionId
			
			// Try to store the execution ID - if it was already there, we have a duplicate
			_, loaded := executionIds.LoadOrStore(executionId, true)
			if loaded {
				atomic.StoreInt32(&duplicateFound, 1)
				return fmt.Errorf("duplicate ExecutionId detected: %s", executionId)
			}
		}
		return nil
	})

	systemContext := &model.EventContext{}

	// Create many instances in parallel to stress test uniqueness
	concurrentOps := 100
	var wg sync.WaitGroup
	wg.Add(concurrentOps)

	for i := 0; i < concurrentOps; i++ {
		go func(idx int) {
			defer wg.Done()
			_, _ = noteModel.Create(wst.M{
				"title":        fmt.Sprintf("exec-id-test-%d-%d", idx, time.Now().UnixNano()),
				"body":         "test",
				"__testHookId": hookId,
			}, systemContext)
		}(i)
	}

	wg.Wait()

	// Verify no duplicates were found
	hasDuplicate := atomic.LoadInt32(&duplicateFound)
	assert.Equal(t, int32(0), hasDuplicate, 
		"No duplicate ExecutionIds should be found - atomic.AddInt64 must guarantee uniqueness")

	// Count unique IDs collected
	count := 0
	executionIds.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	
	// We should have exactly concurrentOps unique ExecutionIds
	assert.Equal(t, concurrentOps, count, 
		"Should have collected unique ExecutionIds for all operations")
}

// TestQueueOperationWithSharedContext tests the critical scenario where
// multiple parallel goroutines share the same global context but each
// must have isolated queued operations (the user's production scenario)
func TestQueueOperationWithSharedContext(t *testing.T) {
	if noteModel == nil {
		t.Skip("noteModel not initialized")
		return
	}

	var hookId = fmt.Sprintf("test-shared-ctx-%d", time.Now().UnixNano())
	var queuedOps sync.Map     // executionId -> title
	var executedOps sync.Map   // executionId -> title
	var mismatchCount int32

	noteModel.Observe("before save", func(ctx *model.EventContext) error {
		if !ctx.IsNewInstance && ctx.Data.GetString("__testHookId") == hookId {
			title := ctx.Data.GetString("title")
			executionId := model.FindBaseContext(ctx).ExecutionId
			
			// Capture the title (simulates capturing dbId in production)
			capturedTitle := title
			queuedOps.Store(executionId, capturedTitle)

			ctx.QueueOperation("after save", func(nextCtx *model.EventContext) error {
				instanceTitle := nextCtx.Instance.GetString("title")
				executedOps.Store(executionId, instanceTitle)
				
				// This is the critical check: even with shared context,
				// the instance should match the captured variable
				if instanceTitle != capturedTitle {
					atomic.AddInt32(&mismatchCount, 1)
					return fmt.Errorf("MISMATCH with shared context: expected '%s' but got '%s'", 
						capturedTitle, instanceTitle)
				}
				return nil
			})
		}
		return nil
	})

	systemContext := &model.EventContext{}

	// First create notes that we'll update
	concurrentOps := 30
	notes := make([]model.Instance, concurrentOps)
	for i := 0; i < concurrentOps; i++ {
		note, _ := noteModel.Create(wst.M{
			"title": fmt.Sprintf("shared-ctx-note-%d", i),
			"body":  "initial",
		}, systemContext)
		notes[i] = note
	}

	// Now update all notes in parallel, ALL using the SAME shared context
	// This simulates the user's scenario where a global context is shared
	sharedGlobalContext := systemContext

	var wg sync.WaitGroup
	wg.Add(concurrentOps)

	for i := 0; i < concurrentOps; i++ {
		go func(idx int) {
			defer wg.Done()

			newTitle := fmt.Sprintf("updated-title-%d-%d", idx, time.Now().UnixNano())
			_, _ = noteModel.UpdateById(notes[idx].GetID(), wst.M{
				"title":        newTitle,
				"__testHookId": hookId,
			}, sharedGlobalContext) // ← ALL goroutines use SAME context!
		}(i)
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	// THE CRITICAL ASSERTION: Even with shared context, NO mismatches
	mismatchCnt := atomic.LoadInt32(&mismatchCount)
	assert.Equal(t, int32(0), mismatchCnt, 
		"Even with shared global context, ExecutionId (with goroutine ID) should prevent mismatches")

	// Verify each queued operation matched its execution
	queuedOps.Range(func(key, value interface{}) bool {
		executionId := key.(string)
		queuedTitle := value.(string)
		
		executedTitle, ok := executedOps.Load(executionId)
		assert.True(t, ok, "Operation should have executed for ExecutionId: %s", executionId)
		assert.Equal(t, queuedTitle, executedTitle, 
			"Executed title should match queued title for ExecutionId: %s", executionId)
		
		return true
	})
}

// TestQueueOperationThreadSafety tests that concurrent access to pendingOperations
// doesn't cause panics (tests the mutex protection)
func TestQueueOperationThreadSafety(t *testing.T) {
	if noteModel == nil {
		t.Skip("noteModel not initialized")
		return
	}

	var hookId = fmt.Sprintf("test-thread-safety-%d", time.Now().UnixNano())
	var queuedCount int32
	var executedCount int32

	// Setup hooks that queue and execute operations
	noteModel.Observe("before save", func(ctx *model.EventContext) error {
		if ctx.IsNewInstance && ctx.Data.GetString("__testHookId") == hookId {
			// Queue multiple operations to stress test the mutex
			for j := 0; j < 5; j++ {
				jCopy := j
				ctx.QueueOperation("after save", func(nextCtx *model.EventContext) error {
					atomic.AddInt32(&executedCount, 1)
					// Simulate some work
					time.Sleep(time.Millisecond)
					_ = jCopy // use captured variable
					return nil
				})
				atomic.AddInt32(&queuedCount, 1)
			}
		}
		return nil
	})

	systemContext := &model.EventContext{}

	// Create many instances concurrently
	concurrentOps := 20
	var wg sync.WaitGroup
	wg.Add(concurrentOps)

	for i := 0; i < concurrentOps; i++ {
		go func(idx int) {
			defer wg.Done()
			_, _ = noteModel.Create(wst.M{
				"title":        fmt.Sprintf("thread-safety-test-%d-%d", idx, time.Now().UnixNano()),
				"body":         "test",
				"__testHookId": hookId,
			}, systemContext)
		}(i)
	}

	wg.Wait()

	// Wait for all queued operations
	time.Sleep(1 * time.Second)

	// If we get here without a panic, the mutex protection is working
	// Verify all operations were queued and executed
	qCount := atomic.LoadInt32(&queuedCount)
	eCount := atomic.LoadInt32(&executedCount)
	
	assert.Equal(t, qCount, eCount, 
		"All queued operations should have been executed without race conditions")
	assert.Equal(t, int32(concurrentOps*5), qCount, 
		"Should have queued 5 operations per instance")
}
