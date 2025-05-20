package topk

import (
	"sort"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpaceSavingBasics tests basic functionality of Space-Saving
func TestSpaceSavingBasics(t *testing.T) {
	ss := NewSpaceSaving(3)
	require.NotNil(t, ss, "Space-Saving should be created")
	
	// Add some items
	ss.Add("a", 1.0)
	ss.Add("b", 2.0)
	ss.Add("c", 3.0)
	
	// Get the top items
	items := ss.GetTopK()
	require.Len(t, items, 3, "Should have 3 items")
	
	// Items should be sorted by count in descending order
	assert.Equal(t, "c", items[0].ID, "First item should be c")
	assert.Equal(t, "b", items[1].ID, "Second item should be b")
	assert.Equal(t, "a", items[2].ID, "Third item should be a")
	
	// Counts should be preserved
	assert.Equal(t, 3.0, items[0].Count, "Count for c should be 3.0")
	assert.Equal(t, 2.0, items[1].Count, "Count for b should be 2.0")
	assert.Equal(t, 1.0, items[2].Count, "Count for a should be 1.0")
	
	// Error estimates should be 0 for items added when capacity not reached
	assert.Equal(t, 0.0, items[0].Error, "Error for c should be 0.0")
	assert.Equal(t, 0.0, items[1].Error, "Error for b should be 0.0")
	assert.Equal(t, 0.0, items[2].Error, "Error for a should be 0.0")
}

// TestSpaceSavingReplaceLeastFrequent tests item replacement
func TestSpaceSavingReplaceLeastFrequent(t *testing.T) {
	ss := NewSpaceSaving(3)
	
	// Fill capacity
	ss.Add("a", 3.0)
	ss.Add("b", 2.0)
	ss.Add("c", 1.0)
	
	// Add a new item, should replace c
	ss.Add("d", 5.0)
	
	// Get top items
	items := ss.GetTopK()
	require.Len(t, items, 3, "Should have 3 items")
	
	// Check if c was replaced by d
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	assert.Contains(t, ids, "a", "Items should contain a")
	assert.Contains(t, ids, "b", "Items should contain b")
	assert.Contains(t, ids, "d", "Items should contain d")
	assert.NotContains(t, ids, "c", "Items should not contain c")
	
	// Check the error bound for d
	// Find d in the items
	var dItem *Item
	for _, item := range items {
		if item.ID == "d" {
			dItem = item
			break
		}
	}
	require.NotNil(t, dItem, "Item d should exist")
	
	// Error bound should be the count of the replaced item (c had count 1.0)
	assert.Equal(t, 1.0, dItem.Error, "Error for d should be 1.0")
	
	// Count should be sum of new item count and replaced item count (5.0 + 1.0)
	assert.Equal(t, 6.0, dItem.Count, "Count for d should be 6.0")
}

// TestSpaceSavingUpdateExisting tests updating existing items
func TestSpaceSavingUpdateExisting(t *testing.T) {
	ss := NewSpaceSaving(3)
	
	// Add initial items
	ss.Add("a", 3.0)
	ss.Add("b", 2.0)
	ss.Add("c", 1.0)
	
	// Update an existing item
	ss.Add("b", 4.0)
	
	// Get top items
	items := ss.GetTopK()
	require.Len(t, items, 3, "Should have 3 items")
	
	// Find b in the items
	var bItem *Item
	for _, item := range items {
		if item.ID == "b" {
			bItem = item
			break
		}
	}
	require.NotNil(t, bItem, "Item b should exist")
	
	// Count should be updated (2.0 + 4.0)
	assert.Equal(t, 6.0, bItem.Count, "Count for b should be 6.0")
	
	// Error should still be 0 for items added before capacity reached
	assert.Equal(t, 0.0, bItem.Error, "Error for b should be 0.0")
}

// TestSpaceSavingCoverage tests coverage calculation
func TestSpaceSavingCoverage(t *testing.T) {
	ss := NewSpaceSaving(2)
	
	// Empty state should have 100% coverage by convention
	assert.Equal(t, 1.0, ss.GetCoverage(), "Empty state should have 100% coverage")
	
	// Add some items
	ss.Add("a", 4.0)
	ss.Add("b", 2.0)
	
	// Coverage should be 100% when not exceeding capacity
	assert.Equal(t, 1.0, ss.GetCoverage(), "Should have 100% coverage")
	
	// Now exceed capacity
	ss.Add("c", 3.0) // This replaces b
	
	// Calculate manual coverage for verification
	// a: count 4.0, error 0.0
	// c: count 5.0 (3.0 + 2.0), error 2.0
	// Total observed: 4.0 + 3.0 = 7.0
	// Total stored: 4.0 + 5.0 = 9.0
	// Adjusted coverage: (9.0 - 2.0) / 7.0 = 1.0
	// The coverage should be capped at 1.0
	assert.Equal(t, 1.0, ss.GetCoverage(), "Should have 100% coverage")
	
	// Add more data to create potential overestimation
	ss.Add("d", 10.0) // Replace the smallest item
	ss.Add("e", 20.0) // Replace the smallest item again
	
	// Now the coverage should adjust for potential overestimation
	coverage := ss.GetCoverage()
	assert.LessOrEqual(t, coverage, 1.0, "Coverage should not exceed 100%")
	assert.Greater(t, coverage, 0.0, "Coverage should be greater than 0%")
}

// TestSpaceSavingSetK tests dynamic resizing
func TestSpaceSavingSetK(t *testing.T) {
	ss := NewSpaceSaving(5)
	
	// Add some items
	for i := 0; i < 5; i++ {
		ss.Add(string(rune('a'+i)), float64(i+1))
	}
	
	// Verify we have 5 items
	items1 := ss.GetTopK()
	assert.Len(t, items1, 5, "Should have 5 items")
	
	// Decrease k
	ss.SetK(3)
	
	// Should now have only top 3 items
	items2 := ss.GetTopK()
	assert.Len(t, items2, 3, "Should have 3 items after reducing k")
	
	// Top items should be preserved
	ids := make([]string, len(items2))
	for i, item := range items2 {
		ids[i] = item.ID
	}
	assert.Contains(t, ids, "e", "Should retain highest item e")
	assert.Contains(t, ids, "d", "Should retain second-highest item d")
	assert.Contains(t, ids, "c", "Should retain third-highest item c")
	
	// Increase k again
	ss.SetK(4)
	
	// Add a new item
	ss.Add("f", 10.0)
	
	// Should accept the new item now
	items3 := ss.GetTopK()
	assert.Len(t, items3, 4, "Should have 4 items after increasing k")
	
	// New item should be included
	ids = make([]string, len(items3))
	for i, item := range items3 {
		ids[i] = item.ID
	}
	assert.Contains(t, ids, "f", "Should contain new item f")
}

// TestSpaceSavingMultithreading tests concurrent access
func TestSpaceSavingConcurrent(t *testing.T) {
	ss := NewSpaceSaving(10)
	
	// Add items concurrently from multiple goroutines
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				ss.Add(string(rune('a'+id)), 1.0)
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to finish
	for i := 0; i < 5; i++ {
		<-done
	}
	
	// Check that all items were added correctly
	items := ss.GetTopK()
	
	// Sort by ID for deterministic comparison
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	
	// Should have 5 items
	assert.Len(t, items, 5, "Should have 5 unique items")
	
	// Each item should have count ~100
	for i, item := range items {
		assert.Equal(t, string(rune('a'+i)), item.ID, "Item ID should match")
		assert.InDelta(t, 100.0, item.Count, 1.0, "Count should be approximately 100")
	}
}