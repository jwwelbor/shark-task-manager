package repository

import (
	"sort"
)

// orderedItem represents any entity with an ID and execution order.
// This is used internally by resequenceOrders to work with both tasks and features.
type orderedItem struct {
	ID             int64
	ExecutionOrder *int
}

// resequenceOrders recalculates execution orders when one item's order changes.
// It ensures all items maintain sequential ordering by shifting items as needed.
//
// Parameters:
//   - items: All items in the collection (with their current orders)
//   - changedID: The ID of the item whose order is being changed
//   - newOrder: The new order for the changed item
//
// Returns:
//   - Updated items with new execution orders
//
// Example:
//
//	items: a-1, b-2, c-3, d-4
//	changedID: 4 (d), newOrder: 2
//	result: a-1, d-2, b-3, c-4
func resequenceOrders(items []orderedItem, changedID int64, newOrder *int) []orderedItem {
	if newOrder == nil {
		return items
	}

	// Find the item being changed and its current order
	var changedItem *orderedItem
	var oldOrder *int
	for i := range items {
		if items[i].ID == changedID {
			changedItem = &items[i]
			oldOrder = items[i].ExecutionOrder
			break
		}
	}

	if changedItem == nil {
		return items
	}

	// If order hasn't changed, no resequencing needed
	if oldOrder != nil && *oldOrder == *newOrder {
		return items
	}

	// Separate items with orders from those without
	var orderedItems []orderedItem
	var unorderedItems []orderedItem

	for i := range items {
		if items[i].ExecutionOrder != nil {
			orderedItems = append(orderedItems, items[i])
		} else {
			unorderedItems = append(unorderedItems, items[i])
		}
	}

	// Sort by current execution order
	sort.Slice(orderedItems, func(i, j int) bool {
		return *orderedItems[i].ExecutionOrder < *orderedItems[j].ExecutionOrder
	})

	// Remove the changed item from its current position
	var remainingItems []orderedItem
	for _, item := range orderedItems {
		if item.ID != changedID {
			remainingItems = append(remainingItems, item)
		}
	}

	// Insert the changed item at the new position
	// newOrder is 1-based, so position index is newOrder - 1
	insertIndex := *newOrder - 1
	if insertIndex < 0 {
		insertIndex = 0
	}
	if insertIndex > len(remainingItems) {
		insertIndex = len(remainingItems)
	}

	// Build the new ordered list
	var reorderedItems []orderedItem
	reorderedItems = append(reorderedItems, remainingItems[:insertIndex]...)
	changedItem.ExecutionOrder = newOrder
	reorderedItems = append(reorderedItems, *changedItem)
	reorderedItems = append(reorderedItems, remainingItems[insertIndex:]...)

	// Reassign sequential orders (1, 2, 3, ...)
	for i := range reorderedItems {
		order := i + 1
		reorderedItems[i].ExecutionOrder = &order
	}

	// Combine ordered and unordered items
	result := append(reorderedItems, unorderedItems...)

	return result
}
