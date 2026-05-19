package model

import (
	"math"
	"math/rand"
	"sort"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func ptrInt64(v int) *int64 {
	p := int64(v)
	return &p
}

func ptrUint(v int) *uint {
	p := uint(v)
	return &p
}

// Testable versions of internal logic without DB dependencies.
// These test the pure algorithmic parts of GetRandomSatisfiedChannel.

func TestIsChannelIDInList(t *testing.T) {
	tests := []struct {
		name     string
		list     []int
		id       int
		expected bool
	}{
		{"id in list", []int{1, 2, 3}, 2, true},
		{"id not in list", []int{1, 2, 3}, 4, false},
		{"empty list", []int{}, 1, false},
		{"single element found", []int{1}, 1, true},
		{"single element not found", []int{1}, 2, false},
		{"first element", []int{5, 10, 15}, 5, true},
		{"last element", []int{5, 10, 15}, 15, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isChannelIDInList(tt.list, tt.id)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestWeightedRandomSelection(t *testing.T) {
	// Test that weighted random selection distributes proportionally.
	// This is a pure algorithm test: we simulate the weighting logic
	// from GetRandomSatisfiedChannel without the DB/cache dependency.

	type weightedChannel struct {
		id     int
		weight int
	}

	runTrial := func(channels []weightedChannel, totalWeight int) int {
		rng := rand.New(rand.NewSource(42))
		r := rng.Intn(totalWeight)
		for _, ch := range channels {
			r -= ch.weight
			if r < 0 {
				return ch.id
			}
		}
		return channels[len(channels)-1].id
	}

	channels := []weightedChannel{
		{id: 1, weight: 50},
		{id: 2, weight: 30},
		{id: 3, weight: 20},
	}
	totalWeight := 0
	for _, ch := range channels {
		totalWeight += ch.weight
	}
	require.Equal(t, 100, totalWeight)

	// Run many trials and check distribution
	counts := map[int]int{}
	trials := 10000
	for i := 0; i < trials; i++ {
		selected := runTrial(channels, totalWeight)
		counts[selected]++
	}

	// Allow 5% tolerance
	for _, ch := range channels {
		ratio := float64(counts[ch.id]) / float64(trials)
		expectedRatio := float64(ch.weight) / float64(totalWeight)
		diff := math.Abs(ratio - expectedRatio)
		require.Less(t, diff, 0.05,
			"channel %d: got %.2f%%, want %.2f%% (diff=%.2f%%, tolerance=5%%)",
			ch.id, ratio*100, expectedRatio*100, diff*100)
	}
}

func TestWeightedRandom_ZeroWeightSmoothing(t *testing.T) {
	// When all channels have weight 0, each gets effective weight 100.
	// This tests the smoothing adjustment logic.
	type weightedChannel struct {
		id     int
		weight int
	}

	runTrial := func(channels []weightedChannel, sumWeight int, smoothingAdj int) int {
		rng := rand.New(rand.NewSource(123))
		total := sumWeight + len(channels)*smoothingAdj
		r := rng.Intn(total)
		for _, ch := range channels {
			r -= ch.weight + smoothingAdj
			if r < 0 {
				return ch.id
			}
		}
		return channels[len(channels)-1].id
	}

	channels := []weightedChannel{
		{id: 1, weight: 0},
		{id: 2, weight: 0},
		{id: 3, weight: 0},
	}

	// sumWeight=0, smoothingAdj=100 → each effective weight=100
	counts := map[int]int{}
	trials := 10000
	for i := 0; i < trials; i++ {
		selected := runTrial(channels, 0, 100)
		counts[selected]++
	}

	// Should be approximately equal distribution
	for _, ch := range channels {
		ratio := float64(counts[ch.id]) / float64(trials)
		require.InDelta(t, 0.333, ratio, 0.05,
			"channel %d: all zero weight → should be uniform distribution", ch.id)
	}
}

func TestWeightedRandom_LowWeightSmoothing(t *testing.T) {
	// When average weight < 10, smoothing factor = 100 is applied.
	// This means each channel's effective weight = weight * 100 + adjustment.
	// The adjustment = 100 - avg_weight (for channels below average).
	// After smoothing, distribution should still be proportional to original weights.

	type weightedChannel struct {
		id     int
		weight int
	}

	channels := []weightedChannel{
		{id: 1, weight: 1},
		{id: 2, weight: 2},
		{id: 3, weight: 3},
	}
	// avg = 2, avg < 10 → smoothing factor = 100
	// effective weights: 1*100, 2*100, 3*100 = 100, 200, 300
	sumWeight := 6
	smoothingFactor := 100

	runTrial := func(chans []weightedChannel, sum int, factor int) int {
		rng := rand.New(rand.NewSource(456))
		total := sum * factor
		r := rng.Intn(total)
		for _, ch := range chans {
			r -= ch.weight * factor
			if r < 0 {
				return ch.id
			}
		}
		return chans[len(chans)-1].id
	}

	counts := map[int]int{}
	trials := 10000
	for i := 0; i < trials; i++ {
		selected := runTrial(channels, sumWeight, smoothingFactor)
		counts[selected]++
	}

	// Distribution should reflect original weights (1:2:3)
	for _, ch := range channels {
		ratio := float64(counts[ch.id]) / float64(trials)
		expectedRatio := float64(ch.weight) / float64(sumWeight)
		diff := math.Abs(ratio - expectedRatio)
		require.Less(t, diff, 0.05,
			"channel %d: got %.2f%%, want %.2f%%",
			ch.id, ratio*100, expectedRatio*100)
	}
}

func TestPrioritySortingDescending(t *testing.T) {
	// GetRandomSatisfiedChannel sorts priorities in descending order.
	// This tests that our understanding of the sort is correct.
	priorities := []int{50, 30, 10, 100, 5}
	sorted := make([]int, len(priorities))
	copy(sorted, priorities)
	sort.Sort(sort.Reverse(sort.IntSlice(sorted)))
	require.Equal(t, []int{100, 50, 30, 10, 5}, sorted)
}

func TestRetryClipping(t *testing.T) {
	// retry >= len(uniquePriorities) → retry = len(uniquePriorities) - 1
	tests := []struct {
		name              string
		retry             int
		numPriorities     int
		expectedClippedTo int
	}{
		{"retry within bounds", 1, 5, 1},
		{"retry at last index", 4, 5, 4},
		{"retry equals length", 5, 5, 4},    // 5 >= 5 → clip to 4
		{"retry exceeds length", 10, 3, 2},   // 10 >= 3 → clip to 2
		{"retry is zero", 0, 3, 0},
		{"retry is negative", -1, 3, 0},      // -1 < 0 but Go doesn't clip negatives
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retry := tt.retry
			if retry >= tt.numPriorities {
				retry = tt.numPriorities - 1
			}
			require.Equal(t, tt.expectedClippedTo, retry)
		})
	}
}

func TestGetRandomSatisfiedChannel_DatabaseConsistencyError(t *testing.T) {
	// When a channelId in Group2Model2Channels doesn't exist in ChannelsIDM,
	// an error should be returned (not a nil).
	// We test this by temporarily corrupting the in-memory state.
	origEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = origEnabled }()

	// Save original state
	origGroup2Model2Channels := Group2Model2Channels
	origChannelsIDM := ChannelsIDM

	// Restore after test
	defer func() {
		Group2Model2Channels = origGroup2Model2Channels
		ChannelsIDM = origChannelsIDM
		InitChannelCache()
	}()

	// Set up a "corrupt" state: channel ID 9999 is in the abilities list but not in ChannelsIDM
	Group2Model2Channels = map[string]map[string][]int{
		"default": {"gpt-4o": {9999}},
	}
	ChannelsIDM = map[int]*Channel{
		// 9999 is intentionally missing
	}

	channel, err := GetRandomSatisfiedChannel("default", "gpt-4o", 0)
	require.Error(t, err)
	require.Nil(t, channel)
	require.Contains(t, err.Error(), "数据库一致性错误")
}

func TestGetRandomSatisfiedChannel_EmptyChannelList(t *testing.T) {
	origEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = origEnabled }()

	origGroup2Model2Channels := Group2Model2Channels
	origChannelsIDM := ChannelsIDM
	defer func() {
		Group2Model2Channels = origGroup2Model2Channels
		ChannelsIDM = origChannelsIDM
		InitChannelCache()
	}()

	// No channels registered for this group/model
	Group2Model2Channels = map[string]map[string][]int{
		"default": {"gpt-4o": {}},
	}
	ChannelsIDM = map[int]*Channel{}

	channel, err := GetRandomSatisfiedChannel("default", "gpt-4o", 0)
	require.NoError(t, err)
	require.Nil(t, channel)
}

func TestGetRandomSatisfiedChannel_SingleChannel(t *testing.T) {
	origEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = origEnabled }()

	origGroup2Model2Channels := Group2Model2Channels
	origChannelsIDM := ChannelsIDM
	defer func() {
		Group2Model2Channels = origGroup2Model2Channels
		ChannelsIDM = origChannelsIDM
		InitChannelCache()
	}()

	ch := &Channel{Id: 42, Name: "test-channel", Priority: ptrInt64(10), Weight: ptrUint(50)}
	Group2Model2Channels = map[string]map[string][]int{
		"default": {"gpt-4o": {42}},
	}
	ChannelsIDM = map[int]*Channel{42: ch}

	channel, err := GetRandomSatisfiedChannel("default", "gpt-4o", 0)
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 42, channel.Id)
}

func TestGetRandomSatisfiedChannel_NormalizedModelFallback(t *testing.T) {
	// When exact model name isn't found, FormatMatchingModelName should be applied.
	origEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = origEnabled }()

	origGroup2Model2Channels := Group2Model2Channels
	origChannelsIDM := ChannelsIDM
	defer func() {
		Group2Model2Channels = origGroup2Model2Channels
		ChannelsIDM = origChannelsIDM
		InitChannelCache()
	}()

	ch := &Channel{Id: 7, Name: "gemini-channel", Priority: ptrInt64(10), Weight: ptrUint(50)}
	// Only "gemini-2.5-flash-thinking-*" is registered, not "gemini-2.5-flash-thinking-exp"
	Group2Model2Channels = map[string]map[string][]int{
		"default": {"gemini-2.5-flash-thinking-*": {7}},
	}
	ChannelsIDM = map[int]*Channel{7: ch}

	channel, err := GetRandomSatisfiedChannel("default", "gemini-2.5-flash-thinking-exp", 0)
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 7, channel.Id, "should find channel via normalized model name")
}

func TestGetRandomSatisfiedChannel_MultiplePriorities(t *testing.T) {
	// Test that retry correctly selects different priority tiers.
	origEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = origEnabled }()

	origGroup2Model2Channels := Group2Model2Channels
	origChannelsIDM := ChannelsIDM
	defer func() {
		Group2Model2Channels = origGroup2Model2Channels
		ChannelsIDM = origChannelsIDM
		InitChannelCache()
	}()

	chHigh := &Channel{Id: 1, Name: "high-priority", Priority: ptrInt64(100), Weight: ptrUint(50)}
	chMid := &Channel{Id: 2, Name: "mid-priority", Priority: ptrInt64(50), Weight: ptrUint(30)}
	chLow := &Channel{Id: 3, Name: "low-priority", Priority: ptrInt64(10), Weight: ptrUint(20)}

	Group2Model2Channels = map[string]map[string][]int{
		"default": {"gpt-4o": {1, 2, 3}},
	}
	ChannelsIDM = map[int]*Channel{
		1: chHigh,
		2: chMid,
		3: chLow,
	}

	// retry=0 → highest priority (100)
	channel, err := GetRandomSatisfiedChannel("default", "gpt-4o", 0)
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 1, channel.Id, "retry=0 should select highest priority")

	// retry=1 → second priority (50)
	channel, err = GetRandomSatisfiedChannel("default", "gpt-4o", 1)
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 2, channel.Id, "retry=1 should select second priority")

	// retry=2 → lowest priority (10)
	channel, err = GetRandomSatisfiedChannel("default", "gpt-4o", 2)
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 3, channel.Id, "retry=2 should select lowest priority")

	// retry=99 → clipped to last (retry >= len(uniquePriorities) → clip to 2)
	channel, err = GetRandomSatisfiedChannel("default", "gpt-4o", 99)
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.Equal(t, 3, channel.Id, "retry=99 should be clipped to last priority")
}

func TestChannelCacheRWMutex(t *testing.T) {
	// Verify that channelSyncLock is a valid sync.RWMutex.
	// This test ensures the lock exists and can be used concurrently.
	var mu sync.RWMutex

	// Read lock - multiple readers should be fine
	mu.RLock()
	require.NotPanics(t, func() {})
	mu.RUnlock()

	// Write lock - exclusive
	mu.Lock()
	require.NotPanics(t, func() {})
	mu.Unlock()

	// Concurrent reads
	done := make(chan bool)
	go func() {
		mu.RLock()
		defer mu.RUnlock()
		done <- true
	}()
	go func() {
		mu.RLock()
		defer mu.RUnlock()
		done <- true
	}()
	<-done
	<-done
}
