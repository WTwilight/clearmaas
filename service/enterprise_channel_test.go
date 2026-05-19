package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Integration tests for GetEnterpriseChannelForModel
// Tests the full chain: user → enterprise → pricing sheet → channel → cache match
// ---------------------------------------------------------------------------

// setupChannelCacheForTest sets up the in-memory channel cache for testing.
// It resets the cache after the test.
func setupChannelCacheForTest(t *testing.T) {
	origEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true

	origGroup2Model2Channels := model.Group2Model2Channels
	origChannelsIDM := model.ChannelsIDM

	model.Group2Model2Channels = map[string]map[string][]int{}
	model.ChannelsIDM = map[int]*model.Channel{}
	model.InitChannelCache()

	t.Cleanup(func() {
		common.MemoryCacheEnabled = origEnabled
		model.Group2Model2Channels = origGroup2Model2Channels
		model.ChannelsIDM = origChannelsIDM
		model.InitChannelCache()
	})
}

func addChannelToCache(t *testing.T, id int, name string, status int, group string, models []string) {
	ch := &model.Channel{Id: id, Name: name, Status: status}
	model.ChannelsIDM[id] = ch
	for _, m := range models {
		if _, ok := model.Group2Model2Channels[group]; !ok {
			model.Group2Model2Channels[group] = map[string][]int{}
		}
		model.Group2Model2Channels[group][m] = append(model.Group2Model2Channels[group][m], id)
	}
}

// Test: User not bound to any enterprise → (nil, false) → fallback to original routing
func TestGetEnterpriseChannelForModel_Integration_UserNotInEnterprise(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	channel, found := GetEnterpriseChannelForModel(99999, "gpt-4o", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}

// Test: Enterprise is disabled → (nil, false) → fallback
func TestGetEnterpriseChannelForModel_Integration_EnterpriseDisabled(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusDisabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")

	channel, found := GetEnterpriseChannelForModel(user.Id, "gpt-4o", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}

// Test: Enterprise enabled but no pricing sheet → (nil, false) → fallback
func TestGetEnterpriseChannelForModel_Integration_NoPricingSheet(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")

	channel, found := GetEnterpriseChannelForModel(user.Id, "gpt-4o", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}

// Test: Pricing sheet has no bound channels → (nil, false) → fallback
func TestGetEnterpriseChannelForModel_Integration_NoChannelBound(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	now := time.Now().Unix()
	_ = seedPricingSheet(t, e.Id, "ActiveSheet", model.PricingSheetStatusActive,
		now-86400, now+86400)

	channel, found := GetEnterpriseChannelForModel(user.Id, "gpt-4o", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}

// Test: Channel bound but not in cache (no model support) → fallback
func TestGetEnterpriseChannelForModel_Integration_ChannelNotInCache(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "ActiveSheet", model.PricingSheetStatusActive,
		now-86400, now+86400)
	seedSheetChannelBinding(t, sheet.Id, 10)

	// channel 10 is bound but NOT in cache → skipped
	addChannelToCache(t, 20, "other-channel", common.ChannelStatusEnabled, "default", []string{"gpt-4o"})

	channel, found := GetEnterpriseChannelForModel(user.Id, "gpt-4o", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}

// Test: Channel bound, in cache, enabled, supports model → (channel, true) → use it
func TestGetEnterpriseChannelForModel_Integration_ChannelMatches(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "ActiveSheet", model.PricingSheetStatusActive,
		now-86400, now+86400)
	seedSheetChannelBinding(t, sheet.Id, 10)

	addChannelToCache(t, 10, "vip-channel", common.ChannelStatusEnabled, "default",
		[]string{"gpt-4o", "gpt-4o-mini"})

	channel, found := GetEnterpriseChannelForModel(user.Id, "gpt-4o", "default")
	assert.True(t, found)
	assert.NotNil(t, channel)
	assert.Equal(t, 10, channel.Id)
	assert.Equal(t, "vip-channel", channel.Name)
}

// Test: Multiple channels bound, first one matches → (first channel, true)
func TestGetEnterpriseChannelForModel_Integration_FirstChannelMatches(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "ActiveSheet", model.PricingSheetStatusActive,
		now-86400, now+86400)
	seedSheetChannelBinding(t, sheet.Id, 10)
	seedSheetChannelBinding(t, sheet.Id, 20)

	addChannelToCache(t, 10, "channel-10", common.ChannelStatusEnabled, "default",
		[]string{"gpt-4o", "claude-3-5-sonnet"})
	addChannelToCache(t, 20, "channel-20", common.ChannelStatusEnabled, "default",
		[]string{"gemini-2.0-flash"})

	// First bound channel (10) supports gpt-4o
	channel, found := GetEnterpriseChannelForModel(user.Id, "gpt-4o", "default")
	assert.True(t, found)
	assert.NotNil(t, channel)
	assert.Equal(t, 10, channel.Id)

	// First bound channel (10) doesn't support gemini → check 20 → found
	channel, found = GetEnterpriseChannelForModel(user.Id, "gemini-2.0-flash", "default")
	assert.True(t, found)
	assert.NotNil(t, channel)
	assert.Equal(t, 20, channel.Id)
}

// Test: First channel doesn't support model, second one does → (second, true)
func TestGetEnterpriseChannelForModel_Integration_SecondChannelMatches(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "ActiveSheet", model.PricingSheetStatusActive,
		now-86400, now+86400)
	seedSheetChannelBinding(t, sheet.Id, 30)
	seedSheetChannelBinding(t, sheet.Id, 31)

	addChannelToCache(t, 30, "channel-30", common.ChannelStatusEnabled, "default",
		[]string{"gpt-4o-mini"})
	addChannelToCache(t, 31, "channel-31", common.ChannelStatusEnabled, "default",
		[]string{"claude-3-5-sonnet"})

	// 30 doesn't support claude-3-5-sonnet, 31 does
	channel, found := GetEnterpriseChannelForModel(user.Id, "claude-3-5-sonnet", "default")
	assert.True(t, found)
	assert.NotNil(t, channel)
	assert.Equal(t, 31, channel.Id)
}

// Test: All channels bound but none support the requested model → (nil, false) → fallback
func TestGetEnterpriseChannelForModel_Integration_NoChannelSupportsModel(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "ActiveSheet", model.PricingSheetStatusActive,
		now-86400, now+86400)
	seedSheetChannelBinding(t, sheet.Id, 40)
	seedSheetChannelBinding(t, sheet.Id, 41)

	addChannelToCache(t, 40, "channel-40", common.ChannelStatusEnabled, "default",
		[]string{"gpt-4o"})
	addChannelToCache(t, 41, "channel-41", common.ChannelStatusEnabled, "default",
		[]string{"gpt-4o-mini"})

	channel, found := GetEnterpriseChannelForModel(user.Id, "unknown-model", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}

// Test: Bound channel is disabled (status != 1) → skipped → (nil, false) → fallback
func TestGetEnterpriseChannelForModel_Integration_ChannelDisabled(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "ActiveSheet", model.PricingSheetStatusActive,
		now-86400, now+86400)
	seedSheetChannelBinding(t, sheet.Id, 50)

	addChannelToCache(t, 50, "channel-50", 0 /* disabled */, "default",
		[]string{"gpt-4o"})

	channel, found := GetEnterpriseChannelForModel(user.Id, "gpt-4o", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}

// Test: Sheet expired → no active sheet → (nil, false) → fallback
func TestGetEnterpriseChannelForModel_Integration_SheetExpired(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	now := time.Now().Unix()
	_ = seedPricingSheet(t, e.Id, "ExpiredSheet", model.PricingSheetStatusActive,
		now-2*86400, now-86400) // expired yesterday

	channel, found := GetEnterpriseChannelForModel(user.Id, "gpt-4o", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}

// Test: Sheet in future → no active sheet → (nil, false) → fallback
func TestGetEnterpriseChannelForModel_Integration_SheetFuture(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	now := time.Now().Unix()
	_ = seedPricingSheet(t, e.Id, "FutureSheet", model.PricingSheetStatusActive,
		now+86400, now+30*86400) // starts tomorrow

	channel, found := GetEnterpriseChannelForModel(user.Id, "gpt-4o", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}

// Test: Sheet status inactive → not active → (nil, false) → fallback
func TestGetEnterpriseChannelForModel_Integration_SheetInactiveStatus(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	now := time.Now().Unix()
	_ = seedPricingSheet(t, e.Id, "InactiveSheet", model.PricingSheetStatusInactive,
		now-86400, now+86400)

	channel, found := GetEnterpriseChannelForModel(user.Id, "gpt-4o", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}

// Test: Channel bound in different group → not matched → fallback
func TestGetEnterpriseChannelForModel_Integration_DifferentGroup(t *testing.T) {
	setupChannelCacheForTest(t)
	truncateEnterprise(t)

	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "ActiveSheet", model.PricingSheetStatusActive,
		now-86400, now+86400)
	seedSheetChannelBinding(t, sheet.Id, 60)

	// channel 60 is only in "vip" group, not in "default"
	addChannelToCache(t, 60, "channel-60", common.ChannelStatusEnabled, "vip",
		[]string{"gpt-4o"})

	channel, found := GetEnterpriseChannelForModel(user.Id, "gpt-4o", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}
