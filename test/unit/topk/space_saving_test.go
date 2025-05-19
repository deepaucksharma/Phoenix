package topk

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/deepaucksharma/Phoenix/pkg/util/topk"
)

func TestSpaceSavingBasic(t *testing.T) {
	// Create a new Space-Saving instance with k=3
	ss := topk.NewSpaceSaving(3)
	require.NotNil(t, ss)

	// Add some items
	ss.Add("item1", 10)
	ss.Add("item2", 5)
	ss.Add("item3", 3)

	// Get the top-k items
	items := ss.GetTopK()
	require.Len(t, items, 3)

	// Verify the order
	assert.Equal(t, "item1", items[0].ID)
	assert.Equal(t, "item2", items[1].ID)
	assert.Equal(t, "item3", items[2].ID)

	// Verify the counts
	assert.Equal(t, 10.0, items[0].Count)
	assert.Equal(t, 5.0, items[1].Count)
	assert.Equal(t, 3.0, items[2].Count)

	// Add another count to item3
	ss.Add("item3", 8)

	// Get the top-k items again
	items = ss.GetTopK()

	// Verify the new order
	assert.Equal(t, "item1", items[0].ID)
	assert.Equal(t, "item3", items[1].ID) // item3 should now be second
	assert.Equal(t, "item2", items[2].ID)

	// Verify the counts
	assert.Equal(t, 10.0, items[0].Count)
	assert.Equal(t, 11.0, items[1].Count) // 3 + 8 = 11
	assert.Equal(t, 5.0, items[2].Count)
}

func TestSpaceSavingReplacement(t *testing.T) {
	// Create a new Space-Saving instance with k=3
	ss := topk.NewSpaceSaving(3)

	// Fill it up
	ss.Add("item1", 10)
	ss.Add("item2", 5)
	ss.Add("item3", 3)

	// Add a new item that should replace the minimum
	ss.Add("item4", 4)

	// Get the top-k items
	items := ss.GetTopK()
	require.Len(t, items, 3)

	// Verify item3 was replaced
	found := false
	for _, item := range items {
		if item.ID == "item3" {
			found = true
			break
		}
	}
	assert.False(t, found, "item3 should have been replaced")

	// Verify item4 is now in the top-k
	found = false
	for _, item := range items {
		if item.ID == "item4" {
			found = true
			assert.GreaterOrEqual(t, item.Count, 4.0)
			assert.GreaterOrEqual(t, item.Error, 3.0) // Error should be at least the count of the replaced item
			break
		}
	}
	assert.True(t, found, "item4 should be in the top-k")
}

func TestSpaceSavingSetK(t *testing.T) {
	// Create a new Space-Saving instance with k=5
	ss := topk.NewSpaceSaving(5)

	// Add some items
	for i := 1; i <= 5; i++ {
		ss.Add(fmt.Sprintf("item%d", i), float64(i))
	}

	// Verify we have 5 items
	items := ss.GetTopK()
	assert.Len(t, items, 5)

	// Reduce k to 3
	ss.SetK(3)

	// Verify we now have only the top 3 items
	items = ss.GetTopK()
	assert.Len(t, items, 3)
	assert.Equal(t, "item5", items[0].ID)
	assert.Equal(t, "item4", items[1].ID)
	assert.Equal(t, "item3", items[2].ID)

	// Increase k to 4
	ss.SetK(4)

	// Add a new item
	ss.Add("item6", 6)

	// Verify we now have 4 items with the new one at the top
	items = ss.GetTopK()
	assert.Len(t, items, 4)
	assert.Equal(t, "item6", items[0].ID)
}

func TestSpaceSavingCoverage(t *testing.T) {
	// Create a new Space-Saving instance with k=3
	ss := topk.NewSpaceSaving(3)

	// Add items with known counts
	ss.Add("item1", 50)
	ss.Add("item2", 30)
	ss.Add("item3", 10)
	ss.Add("item4", 5)  // This should replace item3
	ss.Add("item5", 3)  // This should replace item4
	ss.Add("item6", 2)  // This should replace item5

	// Total count should be 50 + 30 + 10 + 5 + 3 + 2 = 100
	// Top 3 items should be item1(50), item2(30), and item6(>=2)
	// Coverage should be around (50 + 30 + x) / 100, where x >= 2

	coverage := ss.GetCoverage()
	assert.GreaterOrEqual(t, coverage, 0.8, "Coverage should be at least 80%")
	assert.LessOrEqual(t, coverage, 1.0, "Coverage should be at most 100%")
}

func TestSpaceSavingThreadSafety(t *testing.T) {
	// Create a new Space-Saving instance with k=10
	ss := topk.NewSpaceSaving(10)

	// Create some random data
	rand.Seed(time.Now().UnixNano())
	const numItems = 100
	const numGoroutines = 10
	const opsPerGoroutine = 1000

	// Add items concurrently
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				item := fmt.Sprintf("item%d", rand.Intn(numItems))
				ss.Add(item, 1.0)
			}
		}()
	}

	wg.Wait()

	// Verify we have items
	items := ss.GetTopK()
	assert.NotEmpty(t, items)
	assert.LessOrEqual(t, len(items), 10)

	// Coverage should be valid
	coverage := ss.GetCoverage()
	assert.GreaterOrEqual(t, coverage, 0.0)
	assert.LessOrEqual(t, coverage, 1.0)
}

func TestSpaceSavingSkewedDistribution(t *testing.T) {
	// Create a new Space-Saving instance with k=5
	ss := topk.NewSpaceSaving(5)

	// Add items with a skewed zipf distribution
	zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().UnixNano())), 1.5, 1.0, 100)
	const numOps = 10000
	
	counts := make(map[string]int)
	for i := 0; i < numOps; i++ {
		item := fmt.Sprintf("item%d", zipf.Uint64())
		ss.Add(item, 1.0)
		counts[item]++
	}

	// Get the actual top 5 items
	type itemCount struct {
		item  string
		count int
	}
	
	var actualTop []itemCount
	for item, count := range counts {
		actualTop = append(actualTop, itemCount{item, count})
	}
	
	// Sort by count in descending order
	sort.Slice(actualTop, func(i, j int) bool {
		return actualTop[i].count > actualTop[j].count
	})
	
	// Get the top-k items from our algorithm
	items := ss.GetTopK()
	
	// Compare the top 5 items
	for i := 0; i < 5 && i < len(actualTop) && i < len(items); i++ {
		assert.Equal(t, actualTop[i].item, items[i].ID, 
			"Item at position %d should be %s, got %s", i, actualTop[i].item, items[i].ID)
	}
	
	// Calculate the actual coverage
	var topkSum int
	var totalSum int
	for i, item := range actualTop {
		totalSum += item.count
		if i < 5 {
			topkSum += item.count
		}
	}
	
	actualCoverage := float64(topkSum) / float64(totalSum)
	estimatedCoverage := ss.GetCoverage()
	
	// Allow for some error in the coverage estimation
	assert.InDelta(t, actualCoverage, estimatedCoverage, 0.1, 
		"Coverage estimation should be close to actual coverage")
}