package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// IsSupplierEnabled
// ---------------------------------------------------------------------------

func TestIsSupplierEnabled_Enabled(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)

	enabled := IsSupplierEnabled(s.Id)
	assert.True(t, enabled)
}

func TestIsSupplierEnabled_Disabled(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusDisabled)

	enabled := IsSupplierEnabled(s.Id)
	assert.False(t, enabled)
}

func TestIsSupplierEnabled_NotFound(t *testing.T) {
	truncateSupplierPricing(t)

	enabled := IsSupplierEnabled(99999)
	assert.False(t, enabled)
}

// ---------------------------------------------------------------------------
// GetChannelActivePricingSheet
// ---------------------------------------------------------------------------

func TestGetChannelActivePricingSheet_ChannelIdMatch(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 100, "渠道100报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingSheetChannel(t, sheet.Id, 100)

	result, err := GetChannelActivePricingSheet(100, s.Id)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, sheet.Id, result.Id)
	assert.Equal(t, "渠道100报价单", result.Name)
}

func TestGetChannelActivePricingSheet_SupplierFallback(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	// Universal sheet (channel_id = 0) with NO channel binding
	// This sheet has no binding at all, so it will be matched by NOT EXISTS fallback
	sheet := seedSupplierPricingSheet(t, s.Id, 0, "通用报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	// Intentionally NOT calling seedSupplierPricingSheetChannel here

	// Query channel 200, no sheet exists for it, should fall back to universal
	result, err := GetChannelActivePricingSheet(200, s.Id)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, sheet.Id, result.Id)
	assert.Equal(t, "通用报价单", result.Name)
}

func TestGetChannelActivePricingSheet_NoSheet(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	// Create sheet for a different supplier
	s2 := seedSupplierForPricing(t, "另一个供应商", model.SupplierStatusEnabled)
	seedSupplierPricingSheet(t, s2.Id, 100, "另一个报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)

	// Query for non-existent sheet
	result, err := GetChannelActivePricingSheet(999, s.Id)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestGetChannelActivePricingSheet_Expired(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	// Expired sheet
	seedSupplierPricingSheet(t, s.Id, 100, "已过期报价单", model.SupplierPricingSheetStatusActive,
		now-30*86400, now-86400)

	result, err := GetChannelActivePricingSheet(100, s.Id)
	require.NoError(t, err)
	require.Nil(t, result, "expired sheet should not be returned")
}

func TestGetChannelActivePricingSheet_Future(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	// Future sheet
	seedSupplierPricingSheet(t, s.Id, 100, "未来报价单", model.SupplierPricingSheetStatusActive,
		now+86400, now+30*86400)

	result, err := GetChannelActivePricingSheet(100, s.Id)
	require.NoError(t, err)
	require.Nil(t, result, "future sheet should not be returned")
}

func TestGetChannelActivePricingSheet_InactiveStatus(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	// Inactive status sheet
	seedSupplierPricingSheet(t, s.Id, 100, "禁用报价单", model.SupplierPricingSheetStatusInactive,
		now-86400, now+86400)

	result, err := GetChannelActivePricingSheet(100, s.Id)
	require.NoError(t, err)
	require.Nil(t, result, "inactive-status sheet should not be returned")
}

// ---------------------------------------------------------------------------
// GetChannelActivePricingSheetByChannelId
// ---------------------------------------------------------------------------

func TestGetChannelActivePricingSheetByChannelId_NoChannel(t *testing.T) {
	truncateSupplierPricing(t)

	// channelId=0 means no specific channel
	result, err := GetChannelActivePricingSheetByChannelId(0)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestGetChannelActivePricingSheetByChannelId_Found(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 100, "渠道100报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingSheetChannel(t, sheet.Id, 100)

	result, err := GetChannelActivePricingSheetByChannelId(100)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, sheet.Id, result.Id)
}

func TestGetChannelActivePricingSheetByChannelId_ChannelIdPriority(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	// Universal sheet (channel_id = 0) — no channel binding
	_ = seedSupplierPricingSheet(t, s.Id, 0, "通用报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	// Channel-specific sheet — HAS channel binding
	channelSheet := seedSupplierPricingSheet(t, s.Id, 100, "渠道100报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingSheetChannel(t, channelSheet.Id, 100)

	// Channel 100 should match the channel-specific sheet, not the universal one
	result, err := GetChannelActivePricingSheetByChannelId(100)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, channelSheet.Id, result.Id)
	assert.Equal(t, 100, result.ChannelId)
}

// ---------------------------------------------------------------------------
// GetSupplierPricingSheetItems
// ---------------------------------------------------------------------------

func TestGetSupplierPricingSheetItems_Empty(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 0, "空报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)

	items, err := GetSupplierPricingSheetItems(sheet.Id)
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestGetSupplierPricingSheetItems_WithItems(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 0, "报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)

	models := []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"}
	seedSupplierPricingItem(t, sheet.Id, models, model.DiscountTypeRatio, 0.7)

	items, err := GetSupplierPricingSheetItems(sheet.Id)
	require.NoError(t, err)
	assert.Len(t, items, 1)
}

// ---------------------------------------------------------------------------
// GetModelCost
// ---------------------------------------------------------------------------

func TestGetModelCost_Found(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 0, "报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.6)
	seedSupplierPricingItem(t, sheet.Id, []string{"gpt-4o-mini"}, model.DiscountTypeRatio, 0.5)

	cost, found := GetModelCost(sheet.Id, "gpt-4o")
	assert.True(t, found)
	assert.Equal(t, 0.6, cost)
}

func TestGetModelCost_NotFound(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 0, "报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.6)

	cost, found := GetModelCost(sheet.Id, "gpt-4o-mini")
	assert.False(t, found)
	assert.Zero(t, cost)
}

func TestGetModelCost_Ratio(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 0, "报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.75)

	cost, found := GetModelCost(sheet.Id, "gpt-4o")
	assert.True(t, found)
	assert.Equal(t, 0.75, cost)
}

func TestGetModelCost_FixedPrice(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 0, "报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeFixedPrice, 0.004)

	cost, found := GetModelCost(sheet.Id, "gpt-4o")
	assert.True(t, found)
	assert.Equal(t, 0.004, cost)
}

func TestGetModelCost_PerCall(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 0, "报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypePerCall, 0.01)

	cost, found := GetModelCost(sheet.Id, "gpt-4o")
	assert.True(t, found)
	assert.Equal(t, 0.01, cost)
}

// ---------------------------------------------------------------------------
// GetModelCostDiscountType
// ---------------------------------------------------------------------------

func TestGetModelCostDiscountType_Found(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 0, "报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.6)

	discountType, found := GetModelCostDiscountType(sheet.Id, "gpt-4o")
	assert.True(t, found)
	assert.Equal(t, model.DiscountTypeRatio, discountType)
}

func TestGetModelCostDiscountType_NotFound(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 0, "报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.6)

	discountType, found := GetModelCostDiscountType(sheet.Id, "gpt-4o-mini")
	assert.False(t, found)
	assert.Empty(t, discountType)
}

// ---------------------------------------------------------------------------
// GetChannelActivePricingSheetForBilling (billing chain)
// ---------------------------------------------------------------------------

func TestGetChannelActivePricingSheetForBilling_Found(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 100, "渠道100报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingSheetChannel(t, sheet.Id, 100)

	sheetId, found := GetChannelActivePricingSheetForBilling(100)
	assert.True(t, found)
	assert.Equal(t, sheet.Id, sheetId)
}

func TestGetChannelActivePricingSheetForBilling_NotFound(t *testing.T) {
	truncateSupplierPricing(t)

	sheetId, found := GetChannelActivePricingSheetForBilling(999)
	assert.False(t, found)
	assert.Zero(t, sheetId)
}

// ---------------------------------------------------------------------------
// GetModelCostForBilling (billing chain)
// ---------------------------------------------------------------------------

func TestGetModelCostForBilling_Found(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 0, "报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)
	seedSupplierPricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.65)

	cost, found := GetModelCostForBilling(sheet.Id, "gpt-4o")
	assert.True(t, found)
	assert.Equal(t, 0.65, cost)
}

func TestGetModelCostForBilling_NotFound(t *testing.T) {
	truncateSupplierPricing(t)
	s := seedSupplierForPricing(t, "测试供应商", model.SupplierStatusEnabled)
	now := time.Now().Unix()

	sheet := seedSupplierPricingSheet(t, s.Id, 0, "报价单", model.SupplierPricingSheetStatusActive,
		now-86400, now+86400)

	cost, found := GetModelCostForBilling(sheet.Id, "unknown-model")
	assert.False(t, found)
	assert.Zero(t, cost)
}

// ---------------------------------------------------------------------------
// Helper seeders
// ---------------------------------------------------------------------------

func seedSupplierForPricing(t *testing.T, name string, status int) *model.Supplier {
	t.Helper()
	now := time.Now().Unix()
	s := &model.Supplier{
		Name:      name,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(s).Error)
	return s
}

func seedSupplierPricingSheet(t *testing.T, supplierId int, channelId int, name string, status int, startTime, endTime int64) *model.SupplierPricingSheet {
	t.Helper()
	now := time.Now().Unix()
	sheet := &model.SupplierPricingSheet{
		SupplierId: supplierId,
		ChannelId:  channelId,
		Name:       name,
		Status:     status,
		StartTime:  startTime,
		EndTime:    endTime,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, model.DB.Create(sheet).Error)
	return sheet
}

func seedSupplierPricingItem(t *testing.T, sheetId int, models []string, discountType string, discountValue float64) *model.SupplierPricingItem {
	t.Helper()
	item := &model.SupplierPricingItem{
		PricingSheetId: sheetId,
		Models:         models,
		DiscountType:   discountType,
		DiscountValue:  discountValue,
	}
	require.NoError(t, model.DB.Create(item).Error)
	return item
}

func seedSupplierPricingSheetChannel(t *testing.T, sheetId int, channelId int) {
	t.Helper()
	binding := &model.SupplierPricingSheetChannel{
		PricingSheetId: sheetId,
		ChannelId:      channelId,
	}
	require.NoError(t, model.DB.Create(binding).Error)
}

func truncateSupplierPricing(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM supplier_pricing_sheet_channels")
		model.DB.Exec("DELETE FROM supplier_pricing_items")
		model.DB.Exec("DELETE FROM supplier_pricing_sheets")
		model.DB.Exec("DELETE FROM suppliers")
		// Also clean up tables created by other test files in the same shared DB
		model.DB.Exec("DELETE FROM enterprise_pricing_sheet_channels")
		model.DB.Exec("DELETE FROM enterprise_pricing_items")
		model.DB.Exec("DELETE FROM enterprise_pricing_sheets")
		model.DB.Exec("DELETE FROM enterprise_user_bindings")
		model.DB.Exec("DELETE FROM enterprises")
	})
}
