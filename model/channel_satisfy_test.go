package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// IsChannelEnabledForGroupModel: guard clauses and cache path
// ---------------------------------------------------------------------------

func TestIsChannelEnabledForGroupModel_EmptyInput(t *testing.T) {
	// Guard clauses: group="" || model="" || channelID<=0 → false
	require.False(t, IsChannelEnabledForGroupModel("", "gpt-4o", 1))
	require.False(t, IsChannelEnabledForGroupModel("default", "", 1))
	require.False(t, IsChannelEnabledForGroupModel("default", "gpt-4o", 0))
	require.False(t, IsChannelEnabledForGroupModel("default", "gpt-4o", -1))
}

func TestIsChannelEnabledForGroupModel_NilMap(t *testing.T) {
	origEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = origEnabled }()

	origGroup2Model2Channels := group2model2channels
	origChannelsIDM := channelsIDM
	defer func() {
		group2model2channels = origGroup2Model2Channels
		channelsIDM = origChannelsIDM
		InitChannelCache()
	}()

	// When group2model2channels is nil, should return false without panic
	group2model2channels = nil
	result := IsChannelEnabledForGroupModel("default", "gpt-4o", 1)
	require.False(t, result)
}

func TestIsChannelEnabledForGroupModel_DirectMatch(t *testing.T) {
	origEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = origEnabled }()

	origGroup2Model2Channels := group2model2channels
	origChannelsIDM := channelsIDM
	defer func() {
		group2model2channels = origGroup2Model2Channels
		channelsIDM = origChannelsIDM
		InitChannelCache()
	}()

	ch := &Channel{Id: 10, Name: "test-channel"}
	group2model2channels = map[string]map[string][]int{
		"default": {"gpt-4o": {10}},
	}
	channelsIDM = map[int]*Channel{10: ch}

	// Exact model match
	require.True(t, IsChannelEnabledForGroupModel("default", "gpt-4o", 10))
	// Model not in this group
	require.False(t, IsChannelEnabledForGroupModel("default", "gpt-4.1", 10))
	// Channel ID not registered for this model
	require.False(t, IsChannelEnabledForGroupModel("default", "gpt-4o", 99))
}

func TestIsChannelEnabledForGroupModel_NormalizedFallback(t *testing.T) {
	origEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = origEnabled }()

	origGroup2Model2Channels := group2model2channels
	origChannelsIDM := channelsIDM
	defer func() {
		group2model2channels = origGroup2Model2Channels
		channelsIDM = origChannelsIDM
		InitChannelCache()
	}()

	ch := &Channel{Id: 5, Name: "gemini-channel"}
	// Only normalized key exists
	group2model2channels = map[string]map[string][]int{
		"default": {"gemini-2.5-flash-thinking-*": {5}},
	}
	channelsIDM = map[int]*Channel{5: ch}

	// Exact match fails, normalized fallback succeeds
	result := IsChannelEnabledForGroupModel("default", "gemini-2.5-flash-thinking-exp", 5)
	require.True(t, result, "should match via normalized model name")

	// Normalized model same as original → no second lookup
	result = IsChannelEnabledForGroupModel("default", "gpt-4o", 5)
	require.False(t, result, "no match even with normalization")
}

func TestIsChannelEnabledForGroupModel_NormalizedSameAsOriginal(t *testing.T) {
	// When FormatMatchingModelName returns the same name, no second lookup is needed.
	// This tests that the "normalized != modelName" check prevents infinite loops.
	name := "gpt-4o"
	normalized := ratio_setting.FormatMatchingModelName(name)
	require.Equal(t, name, normalized, "gpt-4o should not be normalized")
}

func TestIsChannelEnabledForAnyGroupModel(t *testing.T) {
	origEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	defer func() { common.MemoryCacheEnabled = origEnabled }()

	origGroup2Model2Channels := group2model2channels
	origChannelsIDM := channelsIDM
	defer func() {
		group2model2channels = origGroup2Model2Channels
		channelsIDM = origChannelsIDM
		InitChannelCache()
	}()

	ch := &Channel{Id: 7, Name: "vip-channel"}
	group2model2channels = map[string]map[string][]int{
		"vip": {"gpt-4o": {7}},
	}
	channelsIDM = map[int]*Channel{7: ch}

	// Channel in first group
	groups := []string{"vip", "default"}
	require.True(t, IsChannelEnabledForAnyGroupModel(groups, "gpt-4o", 7))

	// Channel in second group
	groups = []string{"default", "vip"}
	require.True(t, IsChannelEnabledForAnyGroupModel(groups, "gpt-4o", 7))

	// Channel not in any group
	groups = []string{"default", "vip"}
	require.False(t, IsChannelEnabledForAnyGroupModel(groups, "gpt-4o", 99))

	// Empty groups
	require.False(t, IsChannelEnabledForAnyGroupModel([]string{}, "gpt-4o", 7))
}

func TestIsChannelEnabledForAnyGroupModel_EmptySlice(t *testing.T) {
	require.False(t, IsChannelEnabledForAnyGroupModel([]string{}, "gpt-4o", 1))
}
