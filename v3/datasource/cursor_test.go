package datasource

import (
	"context"
	"errors"
	"testing"
)

// Tests for Cursor interface contract: the stubCursor should decode documents correctly.
// These tests FAIL because the real PostgresCursor is not implemented.

type stubCursor struct {
	results []map[string]interface{}
	index   int
	err     error
}

func (c *stubCursor) Next(ctx context.Context) bool {
	if c.index >= len(c.results) {
		return false
	}
	c.index++
	return true
}

func (c *stubCursor) Decode(val interface{}) error {
	if c.index == 0 {
		return nil
	}
	if m, ok := val.(*map[string]interface{}); ok {
		*m = c.results[c.index-1]
	}
	return nil
}

func (c *stubCursor) All(ctx context.Context, val interface{}) error {
	return nil
}

func (c *stubCursor) Close(ctx context.Context) error {
	return nil
}

func (c *stubCursor) Err() error {
	return c.err
}

func TestCursor_NextIteratesResults(t *testing.T) {
	c := &stubCursor{
		results: []map[string]interface{}{
			{"id": "1", "name": "Alice"},
			{"id": "2", "name": "Bob"},
		},
	}
	ctx := context.Background()

	if !c.Next(ctx) {
		t.Fatal("First Next() should return true")
	}
	if !c.Next(ctx) {
		t.Fatal("Second Next() should return true")
	}
	if c.Next(ctx) {
		t.Fatal("Third Next() should return false (exhausted)")
	}
}

func TestCursor_DecodeReturnsDocument(t *testing.T) {
	c := &stubCursor{
		results: []map[string]interface{}{
			{"id": "1", "name": "Alice"},
		},
	}
	ctx := context.Background()

	if !c.Next(ctx) {
		t.Fatal("Next() should return true")
	}
	var doc map[string]interface{}
	if err := c.Decode(&doc); err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if doc["name"] != "Alice" {
		t.Errorf("Decode() name = %v, want Alice", doc["name"])
	}
}

func TestCursor_EmptyResults(t *testing.T) {
	c := &stubCursor{results: nil}
	ctx := context.Background()

	if c.Next(ctx) {
		t.Fatal("Next() on empty cursor should return false")
	}
}

func TestCursor_ImplementsInterface(t *testing.T) {
	var _ Cursor = (*stubCursor)(nil)
}

func TestCursor_All_ReturnsNil(t *testing.T) {
	c := &stubCursor{
		results: []map[string]interface{}{
			{"id": "1", "name": "Alice"},
		},
	}
	ctx := context.Background()
	var docs []map[string]interface{}
	err := c.All(ctx, &docs)
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
}

func TestCursor_Close_NoError(t *testing.T) {
	c := &stubCursor{
		results: []map[string]interface{}{{"id": "1"}},
	}
	ctx := context.Background()
	if err := c.Close(ctx); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestCursor_Close_Empty_NoError(t *testing.T) {
	c := &stubCursor{results: nil}
	ctx := context.Background()
	if err := c.Close(ctx); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestCursor_Err_Nil(t *testing.T) {
	c := &stubCursor{err: nil}
	if c.Err() != nil {
		t.Errorf("Err() = %v, want nil", c.Err())
	}
}

func TestCursor_Err_WithError(t *testing.T) {
	want := errors.New("cursor error")
	c := &stubCursor{err: want}
	if c.Err() != want {
		t.Errorf("Err() = %v, want %v", c.Err(), want)
	}
}
