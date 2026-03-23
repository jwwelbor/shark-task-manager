package repository

import (
	"github.com/jwwelbor/shark-task-manager/internal/repository/repoutil"
)

// orderedItem represents any entity with an ID and execution order.
// This is a type alias for repoutil.OrderedItem, allowing repository files
// to use the shorter name while delegating logic to repoutil.ResequenceOrders.
type orderedItem = repoutil.OrderedItem
