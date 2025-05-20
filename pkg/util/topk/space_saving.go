// Package topk implements the Space-Saving algorithm for streaming top-k
package topk

import (
	"container/heap"
	"sort"
	"sync"
)

// Item represents a counter in the Space-Saving algorithm
type Item struct {
	ID    string  // Identifier for this item
	Count float64 // Estimated count for this item
	Error float64 // Maximum error in the count estimate
	index int     // Index in the heap, used by heap.Interface
}

// minHeap implements a min-heap of Items based on Count
type minHeap []*Item

// SpaceSaving implements the Space-Saving algorithm for streaming top-k
type SpaceSaving struct {
	k          int              // Maximum number of items to track
	items      map[string]*Item // Map of item ID to item
	heap       minHeap          // Min-heap of items
	totalCount float64          // Total count of all items seen
	lock       sync.RWMutex     // For thread safety
}

// NewSpaceSaving creates a new Space-Saving instance with the specified k
func NewSpaceSaving(k int) *SpaceSaving {
	ss := &SpaceSaving{
		k:          k,
		items:      make(map[string]*Item),
		heap:       make(minHeap, 0, k),
		totalCount: 0,
	}
	heap.Init(&ss.heap)
	return ss
}

// Add adds a count for the specified item
func (ss *SpaceSaving) Add(id string, count float64) {
	if count <= 0 {
		return // Ignore non-positive counts
	}

	ss.lock.Lock()
	defer ss.lock.Unlock()

	ss.totalCount += count

	// If item exists, update its count
	if item, exists := ss.items[id]; exists {
		item.Count += count
		heap.Fix(&ss.heap, item.index)
		return
	}

	// If we haven't reached capacity, add the item
	if len(ss.items) < ss.k {
		item := &Item{
			ID:    id,
			Count: count,
			Error: 0,
		}
		heap.Push(&ss.heap, item)
		ss.items[id] = item
		return
	}

	// Otherwise, replace the minimum item
	minItem := ss.heap[0]
	error := minItem.Count

	// Remove the minimum item
	delete(ss.items, minItem.ID)

	// Replace with the new item
	minItem.ID = id
	minItem.Count = minItem.Count + count
	minItem.Error = error

	// Update the map and fix the heap
	ss.items[id] = minItem
	heap.Fix(&ss.heap, 0)
}

// GetTopK returns the top-k items sorted by count
func (ss *SpaceSaving) GetTopK() []*Item {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	result := make([]*Item, len(ss.items))
	i := 0
	for _, item := range ss.items {
		result[i] = &Item{
			ID:    item.ID,
			Count: item.Count,
			Error: item.Error,
		}
		i++
	}

	// Sort by count in descending order
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

// SetK updates the k value and adjusts the data structure accordingly
func (ss *SpaceSaving) SetK(newK int) {
	if newK <= 0 {
		return // Invalid k
	}

	ss.lock.Lock()
	defer ss.lock.Unlock()

	// If reducing k, keep only the top newK items
	if newK < ss.k && len(ss.items) > newK {
		// Get current top items
		topItems := make([]*Item, len(ss.items))
		i := 0
		for _, item := range ss.items {
			topItems[i] = item
			i++
		}

		// Sort by count in descending order
		sort.Slice(topItems, func(i, j int) bool {
			return topItems[i].Count > topItems[j].Count
		})

		// Clear current structures
		ss.items = make(map[string]*Item)
		ss.heap = make(minHeap, 0, newK)
		heap.Init(&ss.heap)

		// Add back only the top newK items
		for i := 0; i < newK && i < len(topItems); i++ {
			item := topItems[i]
			newItem := &Item{
				ID:    item.ID,
				Count: item.Count,
				Error: item.Error,
			}
			heap.Push(&ss.heap, newItem)
			ss.items[item.ID] = newItem
		}
	}

	ss.k = newK
}

// GetCoverage returns the fraction of the total count covered by the top-k items
func (ss *SpaceSaving) GetCoverage() float64 {
	ss.lock.RLock()
	defer ss.lock.RUnlock()

	if ss.totalCount == 0 {
		return 1.0 // By convention, if nothing's been added, coverage is 100%
	}

	topKCount := 0.0
	for _, item := range ss.items {
		topKCount += item.Count
	}

	return topKCount / ss.totalCount
}

// minHeap implementation (heap.Interface)
func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].Count < h[j].Count }
func (h minHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *minHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*Item)
	item.index = n
	*h = append(*h, item)
}

func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*h = old[0 : n-1]
	return item
}
