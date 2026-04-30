package repoutil

import (
	"sort"
)

// OrderedItem represents any entity with an ID and execution order.
// Entity-specific repositories convert their domain objects to this type
// when calling ResequenceOrders.
type OrderedItem struct {
	ID             int64
	ExecutionOrder *int
}

// ResequenceOrders recalculates execution orders when one item's order changes.
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
func ResequenceOrders(items []OrderedItem, changedID int64, newOrder *int) []OrderedItem {
	if newOrder == nil {
		return items
	}

	// Find the item being changed and its current order
	var changedItem *OrderedItem
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

	// changedItem is excluded here and reinserted into reorderedItems below.
	var orderedItems []OrderedItem
	var unorderedItems []OrderedItem

	for i := range items {
		if items[i].ID == changedID {
			continue
		}
		if items[i].ExecutionOrder != nil {
			orderedItems = append(orderedItems, items[i])
		} else {
			unorderedItems = append(unorderedItems, items[i])
		}
	}

	sort.Slice(orderedItems, func(i, j int) bool {
		return *orderedItems[i].ExecutionOrder < *orderedItems[j].ExecutionOrder
	})

	// newOrder is 1-based, so position index is newOrder - 1
	insertIndex := *newOrder - 1
	if insertIndex < 0 {
		insertIndex = 0
	}
	if insertIndex > len(orderedItems) {
		insertIndex = len(orderedItems)
	}

	var reorderedItems []OrderedItem
	reorderedItems = append(reorderedItems, orderedItems[:insertIndex]...)
	changedItem.ExecutionOrder = newOrder
	reorderedItems = append(reorderedItems, *changedItem)
	reorderedItems = append(reorderedItems, orderedItems[insertIndex:]...)

	// Reassign sequential orders (1, 2, 3, ...)
	for i := range reorderedItems {
		order := i + 1
		reorderedItems[i].ExecutionOrder = &order
	}

	// Combine ordered and unordered items
	result := append(reorderedItems, unorderedItems...)

	return result
}
