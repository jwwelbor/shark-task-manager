package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResequenceOrders_MoveItemDown tests moving an item down in the order sequence
// Example: a-1, b-2, c-3, d-4 → d changes to 2 → a-1, d-2, b-3, c-4
func TestResequenceOrders_MoveItemDown(t *testing.T) {
	// Given: Four items in order 1, 2, 3, 4
	order1 := 1
	order2 := 2
	order3 := 3
	order4 := 4

	items := []orderedItem{
		{ID: 1, ExecutionOrder: &order1}, // a-1
		{ID: 2, ExecutionOrder: &order2}, // b-2
		{ID: 3, ExecutionOrder: &order3}, // c-3
		{ID: 4, ExecutionOrder: &order4}, // d-4
	}

	// When: Item 4 (d) changes from order 4 to order 2
	newOrder := 2
	updatedItems := resequenceOrders(items, 4, &newOrder)

	// Then: Orders should be resequenced as a-1, d-2, b-3, c-4
	assert.Len(t, updatedItems, 4)

	// Find each item and verify its new order
	for _, item := range updatedItems {
		switch item.ID {
		case 1: // a
			assert.NotNil(t, item.ExecutionOrder)
			assert.Equal(t, 1, *item.ExecutionOrder, "Item a should remain at order 1")
		case 4: // d (moved)
			assert.NotNil(t, item.ExecutionOrder)
			assert.Equal(t, 2, *item.ExecutionOrder, "Item d should move to order 2")
		case 2: // b (shifted down)
			assert.NotNil(t, item.ExecutionOrder)
			assert.Equal(t, 3, *item.ExecutionOrder, "Item b should shift to order 3")
		case 3: // c (shifted down)
			assert.NotNil(t, item.ExecutionOrder)
			assert.Equal(t, 4, *item.ExecutionOrder, "Item c should shift to order 4")
		}
	}
}

// TestResequenceOrders_MoveItemUp tests moving an item up in the order sequence
// Example: a-1, b-2, c-3, d-4 → b changes to 4 → a-1, c-2, d-3, b-4
func TestResequenceOrders_MoveItemUp(t *testing.T) {
	// Given: Four items in order 1, 2, 3, 4
	order1 := 1
	order2 := 2
	order3 := 3
	order4 := 4

	items := []orderedItem{
		{ID: 1, ExecutionOrder: &order1}, // a-1
		{ID: 2, ExecutionOrder: &order2}, // b-2
		{ID: 3, ExecutionOrder: &order3}, // c-3
		{ID: 4, ExecutionOrder: &order4}, // d-4
	}

	// When: Item 2 (b) changes from order 2 to order 4
	newOrder := 4
	updatedItems := resequenceOrders(items, 2, &newOrder)

	// Then: Orders should be resequenced as a-1, c-2, d-3, b-4
	assert.Len(t, updatedItems, 4)

	// Find each item and verify its new order
	for _, item := range updatedItems {
		switch item.ID {
		case 1: // a
			assert.NotNil(t, item.ExecutionOrder)
			assert.Equal(t, 1, *item.ExecutionOrder, "Item a should remain at order 1")
		case 3: // c (shifted up)
			assert.NotNil(t, item.ExecutionOrder)
			assert.Equal(t, 2, *item.ExecutionOrder, "Item c should shift to order 2")
		case 4: // d (shifted up)
			assert.NotNil(t, item.ExecutionOrder)
			assert.Equal(t, 3, *item.ExecutionOrder, "Item d should shift to order 3")
		case 2: // b (moved)
			assert.NotNil(t, item.ExecutionOrder)
			assert.Equal(t, 4, *item.ExecutionOrder, "Item b should move to order 4")
		}
	}
}

// TestResequenceOrders_NoChange tests that when order doesn't change, items remain the same
func TestResequenceOrders_NoChange(t *testing.T) {
	// Given: Four items in order 1, 2, 3, 4
	order1 := 1
	order2 := 2
	order3 := 3
	order4 := 4

	items := []orderedItem{
		{ID: 1, ExecutionOrder: &order1},
		{ID: 2, ExecutionOrder: &order2},
		{ID: 3, ExecutionOrder: &order3},
		{ID: 4, ExecutionOrder: &order4},
	}

	// When: Item 2 keeps its order 2
	newOrder := 2
	updatedItems := resequenceOrders(items, 2, &newOrder)

	// Then: No items should change
	assert.Len(t, updatedItems, 4)

	for _, item := range updatedItems {
		switch item.ID {
		case 1:
			assert.Equal(t, 1, *item.ExecutionOrder)
		case 2:
			assert.Equal(t, 2, *item.ExecutionOrder)
		case 3:
			assert.Equal(t, 3, *item.ExecutionOrder)
		case 4:
			assert.Equal(t, 4, *item.ExecutionOrder)
		}
	}
}

// TestResequenceOrders_WithNilOrders tests handling of items with nil execution orders
func TestResequenceOrders_WithNilOrders(t *testing.T) {
	// Given: Some items have nil orders (with gaps in ordering)
	order1 := 1
	order3 := 3

	items := []orderedItem{
		{ID: 1, ExecutionOrder: &order1},
		{ID: 2, ExecutionOrder: nil}, // No order set
		{ID: 3, ExecutionOrder: &order3},
	}

	// When: Item 1 changes order from 1 to 2 (moves to second position)
	newOrder := 2
	updatedItems := resequenceOrders(items, 1, &newOrder)

	// Then: Only items with orders should be resequenced to sequential numbers (1, 2)
	// Result: item 3 becomes order 1, item 1 becomes order 2, item 2 stays nil
	assert.Len(t, updatedItems, 3)

	for _, item := range updatedItems {
		switch item.ID {
		case 1: // moved to position 2
			assert.NotNil(t, item.ExecutionOrder)
			assert.Equal(t, 2, *item.ExecutionOrder)
		case 2: // nil order unchanged
			assert.Nil(t, item.ExecutionOrder)
		case 3: // shifted to position 1 (becomes first in sequence)
			assert.NotNil(t, item.ExecutionOrder)
			assert.Equal(t, 1, *item.ExecutionOrder)
		}
	}
}
