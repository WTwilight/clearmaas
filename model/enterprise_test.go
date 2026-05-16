package model

import (
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Enterprise model
// ---------------------------------------------------------------------------

func TestEnterprise_CRUD(t *testing.T) {
	truncateEnterprise(t)

	// Create
	e := &Enterprise{
		Name:   "测试公司",
		Status: EnterpriseStatusEnabled,
		Remark: "这是一个测试企业",
	}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)
	require.NotZero(t, e.Id)

	// Read
	found, err := GetEnterpriseById(e.Id)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "测试公司", found.Name)
	assert.Equal(t, EnterpriseStatusEnabled, found.Status)
	assert.Equal(t, "这是一个测试企业", found.Remark)

	// Update
	found.Status = EnterpriseStatusDisabled
	found.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Save(found).Error)

	found2, err := GetEnterpriseById(e.Id)
	require.NoError(t, err)
	assert.Equal(t, EnterpriseStatusDisabled, found2.Status)

	// Delete
	require.NoError(t, DeleteEnterprise(e.Id))
	// After deletion, GetEnterpriseById returns nil, nil (not an error)
	// This matches the codebase convention for "not found" → (nil, nil)
	found3, err := GetEnterpriseById(e.Id)
	require.NoError(t, err)
	require.Nil(t, found3)
}

func TestEnterprise_GetAll(t *testing.T) {
	truncateEnterprise(t)

	// Create multiple enterprises
	for i := 1; i <= 3; i++ {
		e := &Enterprise{Name: "公司", Status: EnterpriseStatusEnabled}
		e.CreatedAt = time.Now().Unix()
		e.UpdatedAt = time.Now().Unix()
		require.NoError(t, DB.Create(e).Error)
	}

	all, err := GetAllEnterprises()
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestEnterprise_GetById_NotFound(t *testing.T) {
	truncateEnterprise(t)
	found, err := GetEnterpriseById(99999)
	require.NoError(t, err)
	require.Nil(t, found)
}

// ---------------------------------------------------------------------------
// EnterprisePricingSheet model
// ---------------------------------------------------------------------------

func TestEnterprisePricingSheet_CRUD(t *testing.T) {
	truncateEnterprise(t)

	// Create parent enterprise first
	e := &Enterprise{Name: "Parent Corp", Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	now := time.Now().Unix()
	weekAgo := now - 7*86400
	weekLater := now + 7*86400

	// Create active pricing sheet
	sheet := &EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:         "2024年报价单",
		Status:       PricingSheetStatusActive,
		StartTime:    weekAgo,
		EndTime:      weekLater,
	}
	sheet.CreatedAt = now
	sheet.UpdatedAt = now
	require.NoError(t, DB.Create(sheet).Error)
	require.NotZero(t, sheet.Id)

	// Read
	found, err := GetPricingSheetById(sheet.Id)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "2024年报价单", found.Name)
	assert.Equal(t, e.Id, found.EnterpriseId)

	// Update
	found.Name = "2025年报价单"
	found.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Save(found).Error)

	found2, err := GetPricingSheetById(sheet.Id)
	require.NoError(t, err)
	assert.Equal(t, "2025年报价单", found2.Name)

	// Delete
	require.NoError(t, DeletePricingSheet(sheet.Id))
	found3, err := GetPricingSheetById(sheet.Id)
	require.NoError(t, err)
	require.Nil(t, found3)
}

func TestEnterprisePricingSheet_GetByEnterpriseId(t *testing.T) {
	truncateEnterprise(t)

	e := &Enterprise{Name: "Corp", Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	now := time.Now().Unix()
	for i := 0; i < 3; i++ {
		sheet := &EnterprisePricingSheet{
			EnterpriseId: e.Id,
			Name:         "报价单",
			Status:       PricingSheetStatusActive,
			StartTime:    now - 86400,
			EndTime:      now + 86400,
		}
		sheet.CreatedAt = now
		sheet.UpdatedAt = now
		require.NoError(t, DB.Create(sheet).Error)
	}

	sheets, err := GetPricingSheetsByEnterpriseId(e.Id)
	require.NoError(t, err)
	assert.Len(t, sheets, 3)
}

func TestEnterprisePricingSheet_GetById_NotFound(t *testing.T) {
	truncateEnterprise(t)
	found, err := GetPricingSheetById(99999)
	require.NoError(t, err)
	require.Nil(t, found)
}

// ---------------------------------------------------------------------------
// EnterprisePricingItem model
// ---------------------------------------------------------------------------

func TestEnterprisePricingItem_CRUD(t *testing.T) {
	truncateEnterprise(t)

	e := &Enterprise{Name: "Corp", Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	sheet := &EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:         "报价单",
		Status:       PricingSheetStatusActive,
		StartTime:    time.Now().Unix() - 86400,
		EndTime:      time.Now().Unix() + 86400,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(sheet).Error)

	// Create item
	item := &EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:          "gpt-4o",
		DiscountType:   DiscountTypeRatio,
		DiscountValue:  0.7,
		Remark:         "7折",
	}
	require.NoError(t, DB.Create(item).Error)
	require.NotZero(t, item.Id)

	// Read
	found, err := GetPricingItemById(item.Id)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "gpt-4o", found.Model)
	assert.Equal(t, DiscountTypeRatio, found.DiscountType)
	assert.Equal(t, 0.7, found.DiscountValue)

	// Update
	found.DiscountValue = 0.5
	require.NoError(t, DB.Save(found).Error)

	found2, err := GetPricingItemById(item.Id)
	require.NoError(t, err)
	assert.Equal(t, 0.5, found2.DiscountValue)

	// Delete
	require.NoError(t, DeletePricingItem(item.Id))
	found3, err := GetPricingItemById(item.Id)
	require.NoError(t, err)
	require.Nil(t, found3)
}

func TestEnterprisePricingItem_GetBySheetId(t *testing.T) {
	truncateEnterprise(t)

	e := &Enterprise{Name: "Corp", Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	sheet := &EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:         "报价单",
		Status:       PricingSheetStatusActive,
		StartTime:    time.Now().Unix() - 86400,
		EndTime:      time.Now().Unix() + 86400,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(sheet).Error)

	models := []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"}
	for _, m := range models {
		item := &EnterprisePricingItem{
			PricingSheetId: sheet.Id,
			Model:          m,
			DiscountType:   DiscountTypeRatio,
			DiscountValue:  0.8,
		}
		require.NoError(t, DB.Create(item).Error)
	}

	items, err := GetPricingItemsBySheetId(sheet.Id)
	require.NoError(t, err)
	assert.Len(t, items, 3)
}

func TestEnterprisePricingItem_GetBySheetIdAndModel(t *testing.T) {
	truncateEnterprise(t)

	e := &Enterprise{Name: "Corp", Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	sheet := &EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:         "报价单",
		Status:       PricingSheetStatusActive,
		StartTime:    time.Now().Unix() - 86400,
		EndTime:      time.Now().Unix() + 86400,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(sheet).Error)

	item := &EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:          "gpt-4o",
		DiscountType:   DiscountTypeRatio,
		DiscountValue:  0.6,
	}
	require.NoError(t, DB.Create(item).Error)

	// Exact match
	found, err := GetPricingItemBySheetIdAndModel(sheet.Id, "gpt-4o")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, 0.6, found.DiscountValue)

	// Not found
	found2, err := GetPricingItemBySheetIdAndModel(sheet.Id, "gpt-4o-mini")
	require.NoError(t, err)
	require.Nil(t, found2)
}

func TestEnterprisePricingItem_GetById_NotFound(t *testing.T) {
	truncateEnterprise(t)
	found, err := GetPricingItemById(99999)
	require.NoError(t, err)
	require.Nil(t, found)
}

// ---------------------------------------------------------------------------
// EnterpriseUserBinding model
// ---------------------------------------------------------------------------

func TestEnterpriseUserBinding_CRUD(t *testing.T) {
	truncateEnterprise(t)

	// Create parent enterprise
	e := &Enterprise{Name: "Corp", Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	// Create user
	user := &User{Id: 1, Username: "alice-bind", Status: common.UserStatusEnabled, Quota: 1000, AffCode: "alice-bind"}
	user.CreatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(user).Error)

	// Create binding
	binding := &EnterpriseUserBinding{
		EnterpriseId: e.Id,
		UserId:       user.Id,
	}
	binding.CreatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(binding).Error)
	require.NotZero(t, binding.Id)

	// Read
	found, err := GetUserBinding(user.Id)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, e.Id, found.EnterpriseId)
	assert.Equal(t, user.Id, found.UserId)

	// Delete
	require.NoError(t, DeleteUserBinding(user.Id))
	found2, err := GetUserBinding(user.Id)
	require.NoError(t, err)
	require.Nil(t, found2)
}

func TestEnterpriseUserBinding_GetByEnterpriseId(t *testing.T) {
	truncateEnterprise(t)

	e := &Enterprise{Name: "Corp", Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	for i := 1; i <= 3; i++ {
		user := &User{Id: i, Username: "user" + strconv.Itoa(i), Status: common.UserStatusEnabled, Quota: 1000, AffCode: "code" + strconv.Itoa(i)}
		user.CreatedAt = time.Now().Unix()
		require.NoError(t, DB.Create(user).Error)

		binding := &EnterpriseUserBinding{
			EnterpriseId: e.Id,
			UserId:       user.Id,
		}
		binding.CreatedAt = time.Now().Unix()
		require.NoError(t, DB.Create(binding).Error)
	}

	bindings, err := GetBindingsByEnterpriseId(e.Id)
	require.NoError(t, err)
	assert.Len(t, bindings, 3)
}

func TestEnterpriseUserBinding_GetByEnterpriseId_NotFound(t *testing.T) {
	truncateEnterprise(t)
	bindings, err := GetBindingsByEnterpriseId(99999)
	require.NoError(t, err)
	assert.Len(t, bindings, 0)
}

func TestEnterpriseUserBinding_GetByUserId_NotFound(t *testing.T) {
	truncateEnterprise(t)
	binding, err := GetUserBinding(99999)
	require.NoError(t, err)
	require.Nil(t, binding)
}

func TestEnterpriseUserBinding_UniqueConstraint(t *testing.T) {
	truncateEnterprise(t)

	e := &Enterprise{Name: "Corp", Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	user := &User{Id: 1, Username: "alice-unique", Status: common.UserStatusEnabled, Quota: 1000, AffCode: "alice02"}
	user.CreatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(user).Error)

	binding := &EnterpriseUserBinding{
		EnterpriseId: e.Id,
		UserId:       user.Id,
	}
	binding.CreatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(binding).Error)

	// Duplicate binding should fail
	dup := &EnterpriseUserBinding{
		EnterpriseId: e.Id,
		UserId:       user.Id,
	}
	dup.CreatedAt = time.Now().Unix()
	assert.Error(t, DB.Create(dup).Error)
}

// ---------------------------------------------------------------------------
// Cross-model query: IsUserInEnterprise
// ---------------------------------------------------------------------------

func TestIsUserInEnterprise(t *testing.T) {
	truncateEnterprise(t)

	e := &Enterprise{Name: "Corp", Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	user := &User{Id: 1, Username: "alice-isuser", Status: common.UserStatusEnabled, Quota: 1000, AffCode: "alice03"}
	user.CreatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(user).Error)

	// User not in any enterprise
	id, found := IsUserInEnterprise(user.Id)
	assert.False(t, found)
	assert.Zero(t, id)

	// Bind user to enterprise
	binding := &EnterpriseUserBinding{
		EnterpriseId: e.Id,
		UserId:       user.Id,
	}
	binding.CreatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(binding).Error)

	id, found = IsUserInEnterprise(user.Id)
	assert.True(t, found)
	assert.Equal(t, e.Id, id)
}

// ---------------------------------------------------------------------------
// Pricing sheet validity: active within time range
// ---------------------------------------------------------------------------

func TestGetActivePricingSheet(t *testing.T) {
	truncateEnterprise(t)

	e := &Enterprise{Name: "Corp", Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	now := time.Now().Unix()

	// Expired sheet
	expired := &EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:         "已过期",
		Status:       PricingSheetStatusActive,
		StartTime:    now - 30*86400,
		EndTime:      now - 86400,
	}
	expired.CreatedAt = now
	expired.UpdatedAt = now
	require.NoError(t, DB.Create(expired).Error)

	// Future sheet
	future := &EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:         "未来生效",
		Status:       PricingSheetStatusActive,
		StartTime:    now + 86400,
		EndTime:      now + 30*86400,
	}
	future.CreatedAt = now
	future.UpdatedAt = now
	require.NoError(t, DB.Create(future).Error)

	// Active sheet
	active := &EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:         "当前有效",
		Status:       PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
	}
	active.CreatedAt = now
	active.UpdatedAt = now
	require.NoError(t, DB.Create(active).Error)

	// Inactive status
	inactive := &EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:         "已禁用",
		Status:       PricingSheetStatusInactive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
	}
	inactive.CreatedAt = now
	inactive.UpdatedAt = now
	require.NoError(t, DB.Create(inactive).Error)

	// GetActivePricingSheetByEnterpriseId should return only the active-in-time sheet
	sheets, err := GetActivePricingSheetsByEnterpriseId(e.Id)
	require.NoError(t, err)
	require.Len(t, sheets, 1)
	assert.Equal(t, "当前有效", sheets[0].Name)
}

func TestGetActivePricingSheet_DisabledEnterprise(t *testing.T) {
	truncateEnterprise(t)

	// Disabled enterprise
	e := &Enterprise{Name: "Corp", Status: EnterpriseStatusDisabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	now := time.Now().Unix()
	sheet := &EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:         "报价单",
		Status:       PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
	}
	sheet.CreatedAt = now
	sheet.UpdatedAt = now
	require.NoError(t, DB.Create(sheet).Error)

	// Should not return sheet for disabled enterprise
	sheets, err := GetActivePricingSheetsByEnterpriseId(e.Id)
	require.NoError(t, err)
	assert.Len(t, sheets, 0)
}

func TestGetActivePricingSheet_NoSheet(t *testing.T) {
	truncateEnterprise(t)
	sheets, err := GetActivePricingSheetsByEnterpriseId(99999)
	require.NoError(t, err)
	assert.Len(t, sheets, 0)
}

// ---------------------------------------------------------------------------
// Pricing item uniqueness: same model in same sheet
// ---------------------------------------------------------------------------

func TestPricingItem_UniqueModelPerSheet(t *testing.T) {
	truncateEnterprise(t)

	e := &Enterprise{Name: "Corp", Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)

	sheet := &EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:         "报价单",
		Status:       PricingSheetStatusActive,
		StartTime:    time.Now().Unix() - 86400,
		EndTime:      time.Now().Unix() + 86400,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(sheet).Error)

	item1 := &EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:          "gpt-4o",
		DiscountType:   DiscountTypeRatio,
		DiscountValue:  0.7,
	}
	require.NoError(t, DB.Create(item1).Error)

	// Duplicate model in same sheet should fail
	item2 := &EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:          "gpt-4o", // same model
		DiscountType:   DiscountTypeRatio,
		DiscountValue:  0.5,
	}
	assert.Error(t, DB.Create(item2).Error)
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func truncateEnterprise(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		DB.Exec("DELETE FROM enterprise_pricing_items")
		DB.Exec("DELETE FROM enterprise_pricing_sheets")
		DB.Exec("DELETE FROM enterprise_user_bindings")
		DB.Exec("DELETE FROM enterprises")
		DB.Exec("DELETE FROM users")
	})
}
