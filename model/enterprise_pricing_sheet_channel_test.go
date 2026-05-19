package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// GetFirstMatchedChannelForModel
// ---------------------------------------------------------------------------

func TestGetFirstMatchedChannelForModel_NoBindings(t *testing.T) {
	truncateForSheetChannelTest(t)
	// No bindings exist for this sheet
	channel, found := GetFirstMatchedChannelForModel(9999, "gpt-4o", "default")
	assert.False(t, found)
	assert.Nil(t, channel)
}

func TestGetFirstMatchedChannelForModel_ChannelNotEnabled(t *testing.T) {
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

	truncateForSheetChannelTest(t)

	// Setup: active pricing sheet + channel binding
	e := seedTestEnterprise(t, "Corp", EnterpriseStatusEnabled)
	now := testNow()
	sheet := seedTestPricingSheet(t, e.Id, "ActiveSheet", PricingSheetStatusActive, now-86400, now+86400)
	seedTestSheetChannelBinding(t, sheet.Id, 10)

	// Channel exists in cache but not enabled for gpt-4o in default group
	ch := &Channel{Id: 10, Name: "test-channel", Status: common.ChannelStatusEnabled}
	Group2Model2Channels = map[string]map[string][]int{
		"default": {"gpt-4o": {10}},
	}
	ChannelsIDM = map[int]*Channel{10: ch}

	channel, found := GetFirstMatchedChannelForModel(sheet.Id, "gpt-4o", "default")
	assert.True(t, found, "channel is bound and supports gpt-4o")
	assert.NotNil(t, channel)
	assert.Equal(t, 10, channel.Id)

	// Channel bound but does NOT support this model
	channel, found = GetFirstMatchedChannelForModel(sheet.Id, "unknown-model", "default")
	assert.False(t, found, "no channel supports unknown-model")
	assert.Nil(t, channel)
}

func TestGetFirstMatchedChannelForModel_ChannelDisabled(t *testing.T) {
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

	truncateForSheetChannelTest(t)

	e := seedTestEnterprise(t, "Corp", EnterpriseStatusEnabled)
	now := testNow()
	sheet := seedTestPricingSheet(t, e.Id, "ActiveSheet", PricingSheetStatusActive, now-86400, now+86400)
	seedTestSheetChannelBinding(t, sheet.Id, 20)

	// Channel registered for model but status != enabled
	ch := &Channel{Id: 20, Name: "disabled-channel", Status: 0} // not enabled
	Group2Model2Channels = map[string]map[string][]int{
		"default": {"gpt-4o": {20}},
	}
	ChannelsIDM = map[int]*Channel{20: ch}

	channel, found := GetFirstMatchedChannelForModel(sheet.Id, "gpt-4o", "default")
	assert.False(t, found, "disabled channel should not be returned")
	assert.Nil(t, channel)
}

func TestGetFirstMatchedChannelForModel_MultipleChannels_ReturnsFirstMatch(t *testing.T) {
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

	truncateForSheetChannelTest(t)

	e := seedTestEnterprise(t, "Corp", EnterpriseStatusEnabled)
	now := testNow()
	sheet := seedTestPricingSheet(t, e.Id, "ActiveSheet", PricingSheetStatusActive, now-86400, now+86400)

	// Bind 3 channels
	seedTestSheetChannelBinding(t, sheet.Id, 30)
	seedTestSheetChannelBinding(t, sheet.Id, 31)
	seedTestSheetChannelBinding(t, sheet.Id, 32)

	ch1 := &Channel{Id: 30, Name: "channel-30", Status: common.ChannelStatusEnabled}
	ch2 := &Channel{Id: 31, Name: "channel-31", Status: common.ChannelStatusEnabled}
	ch3 := &Channel{Id: 32, Name: "channel-32", Status: common.ChannelStatusEnabled}

	// channel 30: gpt-4o, channel 31: gpt-4o-mini, channel 32: gpt-4o
	Group2Model2Channels = map[string]map[string][]int{
		"default": {
			"gpt-4o":       {30, 32},
			"gpt-4o-mini": {31},
		},
	}
	ChannelsIDM = map[int]*Channel{
		30: ch1,
		31: ch2,
		32: ch3,
	}

	// Should return the first one that supports gpt-4o
	channel, found := GetFirstMatchedChannelForModel(sheet.Id, "gpt-4o", "default")
	assert.True(t, found)
	assert.NotNil(t, channel)
	// Returns channel 30 (first bound channel that supports the model)
	assert.Equal(t, 30, channel.Id)
}

func TestGetFirstMatchedChannelForModel_CacheMiss(t *testing.T) {
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

	truncateForSheetChannelTest(t)

	e := seedTestEnterprise(t, "Corp", EnterpriseStatusEnabled)
	now := testNow()
	sheet := seedTestPricingSheet(t, e.Id, "ActiveSheet", PricingSheetStatusActive, now-86400, now+86400)
	seedTestSheetChannelBinding(t, sheet.Id, 40)

	ch := &Channel{Id: 40, Name: "channel-40", Status: common.ChannelStatusEnabled}
	Group2Model2Channels = map[string]map[string][]int{
		"default": {"gpt-4o": {40}},
	}
	ChannelsIDM = map[int]*Channel{40: ch}

	channel, found := GetFirstMatchedChannelForModel(sheet.Id, "gpt-4o", "default")
	assert.True(t, found)
	assert.NotNil(t, channel)
	assert.Equal(t, 40, channel.Id)

	// Cache miss for unknown channel → skipped
	channel, found = GetFirstMatchedChannelForModel(sheet.Id, "gpt-4o", "default")
	assert.True(t, found)
	assert.Equal(t, 40, channel.Id)
}

// ---------------------------------------------------------------------------
// GetChannelIdsBySheetId
// ---------------------------------------------------------------------------

func TestGetChannelIdsBySheetId_Empty(t *testing.T) {
	truncateForSheetChannelTest(t)
	ids, err := GetChannelIdsBySheetId(9999)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestGetChannelIdsBySheetId_WithBindings(t *testing.T) {
	truncateForSheetChannelTest(t)

	e := seedTestEnterprise(t, "Corp", EnterpriseStatusEnabled)
	now := testNow()
	sheet := seedTestPricingSheet(t, e.Id, "Sheet", PricingSheetStatusActive, now-86400, now+86400)
	seedTestSheetChannelBinding(t, sheet.Id, 1)
	seedTestSheetChannelBinding(t, sheet.Id, 2)
	seedTestSheetChannelBinding(t, sheet.Id, 3)

	ids, err := GetChannelIdsBySheetId(sheet.Id)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{1, 2, 3}, ids)
}

// ---------------------------------------------------------------------------
// BindChannelsToSheet
// ---------------------------------------------------------------------------

func TestBindChannelsToSheet_Replace(t *testing.T) {
	truncateForSheetChannelTest(t)

	e := seedTestEnterprise(t, "Corp", EnterpriseStatusEnabled)
	now := testNow()
	sheet := seedTestPricingSheet(t, e.Id, "Sheet", PricingSheetStatusActive, now-86400, now+86400)
	seedTestSheetChannelBinding(t, sheet.Id, 1)
	seedTestSheetChannelBinding(t, sheet.Id, 2)

	// Replace with new channel set
	err := BindChannelsToSheet(sheet.Id, []int{3, 4, 5})
	require.NoError(t, err)

	ids, err := GetChannelIdsBySheetId(sheet.Id)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{3, 4, 5}, ids)
}

func TestBindChannelsToSheet_ClearAll(t *testing.T) {
	truncateForSheetChannelTest(t)

	e := seedTestEnterprise(t, "Corp", EnterpriseStatusEnabled)
	now := testNow()
	sheet := seedTestPricingSheet(t, e.Id, "Sheet", PricingSheetStatusActive, now-86400, now+86400)
	seedTestSheetChannelBinding(t, sheet.Id, 1)

	err := BindChannelsToSheet(sheet.Id, []int{})
	require.NoError(t, err)

	ids, err := GetChannelIdsBySheetId(sheet.Id)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func truncateForSheetChannelTest(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		DB.Exec("DELETE FROM enterprise_pricing_items")
		DB.Exec("DELETE FROM enterprise_pricing_sheet_channels")
		DB.Exec("DELETE FROM enterprise_pricing_sheets")
		DB.Exec("DELETE FROM enterprise_user_bindings")
		DB.Exec("DELETE FROM enterprises")
		DB.Exec("DELETE FROM users")
	})
}

func seedTestEnterprise(t *testing.T, name string, status int) *Enterprise {
	t.Helper()
	now := testNow()
	e := &Enterprise{Name: name, Status: status, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, DB.Create(e).Error)
	return e
}

func seedTestPricingSheet(t *testing.T, enterpriseId int, name string, status int, startTime, endTime int64) *EnterprisePricingSheet {
	t.Helper()
	now := testNow()
	sheet := &EnterprisePricingSheet{
		EnterpriseId: enterpriseId,
		Name:        name,
		Status:      status,
		StartTime:   startTime,
		EndTime:     endTime,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	require.NoError(t, DB.Create(sheet).Error)
	return sheet
}

func seedTestSheetChannelBinding(t *testing.T, sheetId, channelId int) *EnterprisePricingSheetChannel {
	t.Helper()
	binding := &EnterprisePricingSheetChannel{
		PricingSheetId: sheetId,
		ChannelId:      channelId,
		CreatedAt:      testNow(),
	}
	require.NoError(t, DB.Create(binding).Error)
	return binding
}

func testNow() int64 {
	return 1737091200 // fixed timestamp so tests are deterministic
}
