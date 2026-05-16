package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// IsUserInEnterprise
// ---------------------------------------------------------------------------

func TestIsUserInEnterprise_NotBound(t *testing.T) {
	// User has no enterprise binding
	enterpriseId, found := IsUserInEnterprise(99999)
	assert.False(t, found)
	assert.Zero(t, enterpriseId)
}

func TestIsUserInEnterprise_Bound(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")

	enterpriseId, found := IsUserInEnterprise(user.Id)
	assert.True(t, found)
	assert.Equal(t, e.Id, enterpriseId)
}

func TestIsUserInEnterprise_DisabledEnterprise(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusDisabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")

	// User is bound but enterprise is disabled → still found (binding exists)
	enterpriseId, found := IsUserInEnterprise(user.Id)
	assert.True(t, found)
	assert.Equal(t, e.Id, enterpriseId)
}

// ---------------------------------------------------------------------------
// GetUserActivePricingSheet
// ---------------------------------------------------------------------------

func TestGetUserActivePricingSheet_NoBinding(t *testing.T) {
	truncateEnterprise(t)
	sheet, err := GetUserActivePricingSheet(99999)
	require.NoError(t, err)
	require.Nil(t, sheet)
}

func TestGetUserActivePricingSheet_NoActiveSheet(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")
	// User bound to enterprise but enterprise has no pricing sheet
	_ = user

	sheet, err := GetUserActivePricingSheet(user.Id)
	require.NoError(t, err)
	require.Nil(t, sheet)
}

func TestGetUserActivePricingSheet_ActiveSheet(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")

	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "2024报价单", model.PricingSheetStatusActive,
		now-86400, now+86400)

	s, err := GetUserActivePricingSheet(user.Id)
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, sheet.Id, s.Id)
	assert.Equal(t, "2024报价单", s.Name)
}

func TestGetUserActivePricingSheet_ExpiredSheet(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")

	now := time.Now().Unix()
	// Sheet expired yesterday
	seedPricingSheet(t, e.Id, "已过期", model.PricingSheetStatusActive,
		now-2*86400, now-86400)

	sheet, err := GetUserActivePricingSheet(user.Id)
	require.NoError(t, err)
	require.Nil(t, sheet, "expired sheet should not be returned")
}

func TestGetUserActivePricingSheet_FutureSheet(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")

	now := time.Now().Unix()
	// Sheet becomes active tomorrow
	seedPricingSheet(t, e.Id, "未来生效", model.PricingSheetStatusActive,
		now+86400, now+30*86400)

	sheet, err := GetUserActivePricingSheet(user.Id)
	require.NoError(t, err)
	require.Nil(t, sheet, "future sheet should not be returned")
}

func TestGetUserActivePricingSheet_InactiveStatus(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")

	now := time.Now().Unix()
	// Sheet is active in time but status=inactive
	seedPricingSheet(t, e.Id, "已禁用状态", model.PricingSheetStatusInactive,
		now-86400, now+86400)

	sheet, err := GetUserActivePricingSheet(user.Id)
	require.NoError(t, err)
	require.Nil(t, sheet, "inactive-status sheet should not be returned")
}

func TestGetUserActivePricingSheet_DisabledEnterprise(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusDisabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")

	now := time.Now().Unix()
	seedPricingSheet(t, e.Id, "报价单", model.PricingSheetStatusActive,
		now-86400, now+86400)

	// Disabled enterprise → no active sheet
	sheet, err := GetUserActivePricingSheet(user.Id)
	require.NoError(t, err)
	require.Nil(t, sheet, "disabled enterprise should not return sheets")
}

func TestGetUserActivePricingSheet_MultipleSheets_ReturnsNewest(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	user := seedUserForEnterprise(t, e.Id, 1, "alice")

	now := time.Now().Unix()

	// Older active sheet
	_ = seedPricingSheet(t, e.Id, "旧报价单", model.PricingSheetStatusActive,
		now-30*86400, now-1*86400)
	// Newer active sheet
	newSheet := seedPricingSheet(t, e.Id, "新报价单", model.PricingSheetStatusActive,
		now-1*86400, now+30*86400)

	s, err := GetUserActivePricingSheet(user.Id)
	require.NoError(t, err)
	require.NotNil(t, s)
	// Returns the one that is currently active and within time range
	// (the newer one has EndTime = now+30d, old has EndTime = now-1d)
	// So the "new" sheet is still active, old sheet expired
	assert.Equal(t, newSheet.Id, s.Id)
}

// ---------------------------------------------------------------------------
// GetPricingSheetItems
// ---------------------------------------------------------------------------

func TestGetPricingSheetItems_Empty(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "空报价单", model.PricingSheetStatusActive,
		now-86400, now+86400)

	items, err := GetPricingSheetItems(sheet.Id)
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestGetPricingSheetItems_WithItems(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "报价单", model.PricingSheetStatusActive,
		now-86400, now+86400)

	models := []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"}
	for _, m := range models {
		seedPricingItem(t, sheet.Id, m, model.DiscountTypeRatio, 0.7)
	}

	items, err := GetPricingSheetItems(sheet.Id)
	require.NoError(t, err)
	assert.Len(t, items, 3)
}

// ---------------------------------------------------------------------------
// GetModelDiscount
// ---------------------------------------------------------------------------

func TestGetModelDiscount_Found(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "报价单", model.PricingSheetStatusActive,
		now-86400, now+86400)
	seedPricingItem(t, sheet.Id, "gpt-4o", model.DiscountTypeRatio, 0.6)
	seedPricingItem(t, sheet.Id, "gpt-4o-mini", model.DiscountTypeRatio, 0.5)

	ratio, found := GetModelDiscount(sheet.Id, "gpt-4o")
	assert.True(t, found)
	assert.Equal(t, 0.6, ratio)
}

func TestGetModelDiscount_NotFound(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "报价单", model.PricingSheetStatusActive,
		now-86400, now+86400)
	seedPricingItem(t, sheet.Id, "gpt-4o", model.DiscountTypeRatio, 0.6)

	ratio, found := GetModelDiscount(sheet.Id, "gpt-4o-mini") // not configured
	assert.False(t, found)
	assert.Zero(t, ratio)
}

func TestGetModelDiscount_FixedPriceDiscountType(t *testing.T) {
	truncateEnterprise(t)
	e := seedEnterprise(t, "Corp", model.EnterpriseStatusEnabled)
	now := time.Now().Unix()
	sheet := seedPricingSheet(t, e.Id, "报价单", model.PricingSheetStatusActive,
		now-86400, now+86400)
	// fixed_price discount type: discount value is used as fixed ratio
	seedPricingItem(t, sheet.Id, "gpt-4o", model.DiscountTypeFixedPrice, 0.004)

	ratio, found := GetModelDiscount(sheet.Id, "gpt-4o")
	assert.True(t, found)
	assert.Equal(t, 0.004, ratio) // fixed price is used as ratio directly
}

// ---------------------------------------------------------------------------
// Helper seeders
// ---------------------------------------------------------------------------

func seedEnterprise(t *testing.T, name string, status int) *model.Enterprise {
	t.Helper()
	now := time.Now().Unix()
	e := &model.Enterprise{Name: name, Status: status}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, model.DB.Create(e).Error)
	return e
}

func seedUserForEnterprise(t *testing.T, enterpriseId, userId int, username string) *model.User {
	t.Helper()
	now := time.Now().Unix()
	user := &model.User{
		Id:     userId,
		Username: username,
		Status:   1,
		Quota:    1000,
	}
	user.CreatedAt = now
	require.NoError(t, model.DB.Create(user).Error)

	binding := &model.EnterpriseUserBinding{
		EnterpriseId: enterpriseId,
		UserId:      userId,
	}
	binding.CreatedAt = now
	require.NoError(t, model.DB.Create(binding).Error)
	return user
}

func seedPricingSheet(t *testing.T, enterpriseId int, name string, status int, startTime, endTime int64) *model.EnterprisePricingSheet {
	t.Helper()
	now := time.Now().Unix()
	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterpriseId,
		Name:        name,
		Status:      status,
		StartTime:   startTime,
		EndTime:     endTime,
	}
	sheet.CreatedAt = now
	sheet.UpdatedAt = now
	require.NoError(t, model.DB.Create(sheet).Error)
	return sheet
}

func seedPricingItem(t *testing.T, sheetId int, modelName string, discountType string, discountValue float64) *model.EnterprisePricingItem {
	t.Helper()
	item := &model.EnterprisePricingItem{
		PricingSheetId: sheetId,
		Model:         modelName,
		DiscountType:  discountType,
		DiscountValue: discountValue,
	}
	require.NoError(t, model.DB.Create(item).Error)
	return item
}

func truncateEnterprise(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM enterprise_pricing_items")
		model.DB.Exec("DELETE FROM enterprise_pricing_sheets")
		model.DB.Exec("DELETE FROM enterprise_user_bindings")
		model.DB.Exec("DELETE FROM enterprises")
		model.DB.Exec("DELETE FROM users")
	})
}
