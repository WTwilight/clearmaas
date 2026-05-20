package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Constants (moved to model files)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Supplier model tests
// ---------------------------------------------------------------------------

func TestSupplier_CRUD(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	// Create
	s := &Supplier{
		Name:      "测试供应商",
		Status:    SupplierStatusEnabled,
		Remark:    "这是一个测试供应商",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(s).Error)
	require.NotZero(t, s.Id)

	// Read
	found, err := GetSupplierById(s.Id)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "测试供应商", found.Name)
	assert.Equal(t, SupplierStatusEnabled, found.Status)
	assert.Equal(t, "这是一个测试供应商", found.Remark)

	// Update
	found.Status = SupplierStatusDisabled
	found.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Save(found).Error)

	found2, err := GetSupplierById(s.Id)
	require.NoError(t, err)
	assert.Equal(t, SupplierStatusDisabled, found2.Status)

	// Delete
	require.NoError(t, DeleteSupplier(s.Id))
	found3, err := GetSupplierById(s.Id)
	require.NoError(t, err)
	require.Nil(t, found3)
}

func TestSupplier_GetAll(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()
	for i := 1; i <= 3; i++ {
		s := &Supplier{
			Name:      "供应商",
			Status:    SupplierStatusEnabled,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, DB.Create(s).Error)
	}

	all, err := GetAllSuppliers()
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestSupplier_GetById(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()
	s := &Supplier{
		Name:      "已知供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(s).Error)

	// Query by ID
	found, err := GetSupplierById(s.Id)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "已知供应商", found.Name)
}

func TestSupplier_GetById_NotFound(t *testing.T) {
	truncateSupplier(t)

	found, err := GetSupplierById(99999)
	require.NoError(t, err)
	require.Nil(t, found)
}

func TestSupplier_GetById_KnownNotFound(t *testing.T) {
	truncateSupplier(t)

	// Create and then delete to ensure ID exists but record doesn't
	now := time.Now().Unix()
	s := &Supplier{
		Name:      "待删除供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(s).Error)
	deletedId := s.Id

	require.NoError(t, DB.Delete(s).Error)

	found, err := GetSupplierById(deletedId)
	require.NoError(t, err)
	require.Nil(t, found)
}

func TestSupplier_IsEnabled(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	// Enabled supplier
	enabled := &Supplier{
		Name:      "已启用供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(enabled).Error)

	// Disabled supplier
	disabled := &Supplier{
		Name:      "已禁用供应商",
		Status:    SupplierStatusDisabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(disabled).Error)

	// Enabled supplier should return true
	assert.True(t, IsSupplierEnabled(enabled.Id))

	// Disabled supplier should return false
	assert.False(t, IsSupplierEnabled(disabled.Id))

	// Non-existent supplier should return false
	assert.False(t, IsSupplierEnabled(99999))
}

func TestSupplier_GetPaginated(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()
	for i := 1; i <= 5; i++ {
		s := &Supplier{
			Name:      "供应商" + string(rune('0'+i)),
			Status:    SupplierStatusEnabled,
			CreatedAt: now,
			UpdatedAt: now,
		}
		require.NoError(t, DB.Create(s).Error)
	}

	// Get page 1, page size 2
	suppliers, total, err := GetSuppliers(1, 2)
	require.NoError(t, err)
	assert.Len(t, suppliers, 2)
	assert.Equal(t, int64(5), total)

	// Get page 3, page size 2
	suppliers2, total2, err := GetSuppliers(3, 2)
	require.NoError(t, err)
	assert.Len(t, suppliers2, 1)
	assert.Equal(t, int64(5), total2)
}

// ---------------------------------------------------------------------------
// SupplierPricingSheet model tests
// ---------------------------------------------------------------------------

func TestSupplierPricingSheet_CRUD(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	// Create parent supplier first
	supplier := &Supplier{
		Name:      "测试供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	weekAgo := now - 7*86400
	weekLater := now + 7*86400

	// Create active pricing sheet
	sheet := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0, // universal
		Name:       "2024年报价单",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  weekAgo,
		EndTime:    weekLater,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(sheet).Error)
	require.NotZero(t, sheet.Id)

	// Read
	found, err := GetSupplierPricingSheetById(sheet.Id)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "2024年报价单", found.Name)
	assert.Equal(t, supplier.Id, found.SupplierId)
	assert.Equal(t, 0, found.ChannelId)

	// Update
	found.Name = "2025年报价单"
	found.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Save(found).Error)

	found2, err := GetSupplierPricingSheetById(sheet.Id)
	require.NoError(t, err)
	assert.Equal(t, "2025年报价单", found2.Name)

	// Delete
	require.NoError(t, DeleteSupplierPricingSheet(sheet.Id))
	found3, err := GetSupplierPricingSheetById(sheet.Id)
	require.NoError(t, err)
	require.Nil(t, found3)
}

func TestSupplierPricingSheet_GetBySupplierId(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	for i := 0; i < 3; i++ {
		sheet := &SupplierPricingSheet{
			SupplierId: supplier.Id,
			ChannelId:  0,
			Name:       "报价单",
			Status:     SupplierPricingSheetStatusActive,
			StartTime:  now - 86400,
			EndTime:    now + 86400,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		require.NoError(t, DB.Create(sheet).Error)
	}

	sheets, err := GetPricingSheetsBySupplierId(supplier.Id)
	require.NoError(t, err)
	assert.Len(t, sheets, 3)
}

func TestSupplierPricingSheet_GetByChannelId(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	// Create sheets with different channel IDs
	channelIds := []int{0, 1, 2, 1}
	for _, ch := range channelIds {
		sheet := &SupplierPricingSheet{
			SupplierId: supplier.Id,
			ChannelId:  ch,
			Name:       "报价单",
			Status:     SupplierPricingSheetStatusActive,
			StartTime:  now - 86400,
			EndTime:    now + 86400,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		require.NoError(t, DB.Create(sheet).Error)
	}

	sheets, err := GetPricingSheetsByChannelId(1)
	require.NoError(t, err)
	assert.Len(t, sheets, 2)
}

func TestSupplierPricingSheet_GetById_NotFound(t *testing.T) {
	truncateSupplier(t)

	found, err := GetSupplierPricingSheetById(99999)
	require.NoError(t, err)
	require.Nil(t, found)
}

func TestSupplierPricingSheet_GetActive_ExpiredSheet(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	// Expired sheet
	expired := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "已过期",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 30*86400,
		EndTime:    now - 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(expired).Error)

	sheets, err := GetActivePricingSheetsBySupplierId(supplier.Id)
	require.NoError(t, err)
	assert.Len(t, sheets, 0)
}

func TestSupplierPricingSheet_GetActive_FutureSheet(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	// Future sheet
	future := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "未来生效",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now + 86400,
		EndTime:    now + 30*86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(future).Error)

	sheets, err := GetActivePricingSheetsBySupplierId(supplier.Id)
	require.NoError(t, err)
	assert.Len(t, sheets, 0)
}

func TestSupplierPricingSheet_GetActive_InactiveStatus(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	// Inactive sheet (within time range but status=0)
	inactive := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "已禁用",
		Status:     SupplierPricingSheetStatusInactive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(inactive).Error)

	sheets, err := GetActivePricingSheetsBySupplierId(supplier.Id)
	require.NoError(t, err)
	assert.Len(t, sheets, 0)
}

func TestSupplierPricingSheet_GetActive_WithinTimeRange(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	// Active sheet
	active := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "当前有效",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(active).Error)

	sheets, err := GetActivePricingSheetsBySupplierId(supplier.Id)
	require.NoError(t, err)
	require.Len(t, sheets, 1)
	assert.Equal(t, "当前有效", sheets[0].Name)
}

func TestSupplierPricingSheet_GetActive_MultipleSheets(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	// Multiple active sheets - should return the latest one (by id desc order)
	for i := 0; i < 3; i++ {
		sheet := &SupplierPricingSheet{
			SupplierId: supplier.Id,
			ChannelId:  0,
			Name:       "报价单" + string(rune('A'+i)),
			Status:     SupplierPricingSheetStatusActive,
			StartTime:  now - 86400,
			EndTime:    now + 86400,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		require.NoError(t, DB.Create(sheet).Error)
	}

	sheets, err := GetActivePricingSheetsBySupplierId(supplier.Id)
	require.NoError(t, err)
	assert.Len(t, sheets, 1)
	assert.Equal(t, "报价单C", sheets[0].Name) // Last created should be first due to ORDER BY id desc
}

func TestSupplierPricingSheet_GetActive_ChannelIdPriority(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	// Universal sheet (channel_id = 0)
	universal := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "通用报价",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(universal).Error)

	// Channel-specific sheet (channel_id = 100)
	channelSheet := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  100,
		Name:       "渠道100专用",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(channelSheet).Error)

	// Get active sheet for channel 100 - should return channel-specific sheet
	sheet, err := GetFirstActivePricingSheetByChannelId(supplier.Id, 100)
	require.NoError(t, err)
	require.NotNil(t, sheet)
	assert.Equal(t, "渠道100专用", sheet.Name)
	assert.Equal(t, 100, sheet.ChannelId)
}

func TestSupplierPricingSheet_GetActive_ChannelIdZero(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	// ChannelId=0 means universal sheet
	universal := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "通用报价单",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(universal).Error)

	sheets, err := GetPricingSheetsByChannelId(0)
	require.NoError(t, err)
	require.Len(t, sheets, 1)
	assert.Equal(t, "通用报价单", sheets[0].Name)
	assert.Equal(t, 0, sheets[0].ChannelId)
}

func TestSupplierPricingSheet_GetActive_DisabledSupplier(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	// Disabled supplier
	supplier := &Supplier{
		Name:      "已禁用供应商",
		Status:    SupplierStatusDisabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	// Active sheet for disabled supplier
	sheet := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "报价单",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(sheet).Error)

	// Should not return sheet for disabled supplier
	sheets, err := GetActivePricingSheetsBySupplierId(supplier.Id)
	require.NoError(t, err)
	assert.Len(t, sheets, 0)
}

func TestSupplierPricingSheet_GetAll_WithSupplierName(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "测试供应商A",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	sheet := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "报价单",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(sheet).Error)

	sheets, total, err := GetAllSupplierPricingSheets(1, 10, "", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))

	// Find our sheet
	var found *SupplierPricingSheetWithSupplier
	for _, s := range sheets {
		if s.Id == sheet.Id {
			found = s
			break
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "测试供应商A", found.SupplierName)
}

// ---------------------------------------------------------------------------
// SupplierPricingItem model tests
// ---------------------------------------------------------------------------

func TestSupplierPricingItem_CRUD(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	sheet := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:   0,
		Name:        "报价单",
		Status:      SupplierPricingSheetStatusActive,
		StartTime:   now - 86400,
		EndTime:     now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(sheet).Error)

	// Create item
	item := &SupplierPricingItem{
		PricingSheetId: sheet.Id,
		Models:         []string{"gpt-4o"},
		DiscountType:   DiscountTypeRatio,
		DiscountValue:  0.7,
		Remark:         "7折",
	}
	require.NoError(t, DB.Create(item).Error)
	require.NotZero(t, item.Id)

	// Read
	found, err := GetSupplierPricingItemById(item.Id)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "gpt-4o", found.Models[0])
	assert.Equal(t, DiscountTypeRatio, found.DiscountType)
	assert.Equal(t, 0.7, found.DiscountValue)

	// Update
	found.DiscountValue = 0.5
	require.NoError(t, DB.Save(found).Error)

	found2, err := GetSupplierPricingItemById(item.Id)
	require.NoError(t, err)
	assert.Equal(t, 0.5, found2.DiscountValue)

	// Delete
	require.NoError(t, DeleteSupplierPricingItem(item.Id))
	found3, err := GetSupplierPricingItemById(item.Id)
	require.NoError(t, err)
	require.Nil(t, found3)
}

func TestSupplierPricingItem_GetBySheetId(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	sheet := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "报价单",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(sheet).Error)

	models := []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"}
	for _, m := range models {
		item := &SupplierPricingItem{
			PricingSheetId: sheet.Id,
		Models:          []string{m},
		DiscountType:    DiscountTypeRatio,
		DiscountValue:   0.8,
	}
		require.NoError(t, DB.Create(item).Error)
	}

	items, err := GetSupplierPricingItemsBySheetId(sheet.Id)
	require.NoError(t, err)
	assert.Len(t, items, 3)
}

func TestSupplierPricingItem_GetBySheetIdAndModel(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	sheet := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "报价单",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(sheet).Error)

	item := &SupplierPricingItem{
		PricingSheetId: sheet.Id,
		Model:          "gpt-4o",
		DiscountType:   DiscountTypeRatio,
		DiscountValue:  0.6,
	}
	require.NoError(t, DB.Create(item).Error)

	// Exact match
	found, err := GetSupplierPricingItemBySheetIdAndModel(sheet.Id, "gpt-4o")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, 0.6, found.DiscountValue)

	// Not found
	found2, err := GetSupplierPricingItemBySheetIdAndModel(sheet.Id, "gpt-4o-mini")
	require.NoError(t, err)
	require.Nil(t, found2)
}

func TestSupplierPricingItem_GetById_NotFound(t *testing.T) {
	truncateSupplier(t)

	found, err := GetSupplierPricingItemById(99999)
	require.NoError(t, err)
	require.Nil(t, found)
}

func TestSupplierPricingItem_UniqueModelPerSheet(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	sheet := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "报价单",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(sheet).Error)

	item1 := &SupplierPricingItem{
		PricingSheetId: sheet.Id,
		Models:           "gpt-4o",
		DiscountType:     DiscountTypeRatio,
		DiscountValue:    0.7,
	}
	require.NoError(t, DB.Create(item1).Error)

	// Duplicate model in same sheet should fail
	item2 := &SupplierPricingItem{
		PricingSheetId: sheet.Id,
		Models:           "gpt-4o", // same model
		DiscountType:     DiscountTypeRatio,
		DiscountValue:    0.5,
	}
	assert.Error(t, DB.Create(item2).Error)
}

func TestSupplierPricingItem_DifferentSheetsSameModel(t *testing.T) {
	truncateSupplier(t)

	now := time.Now().Unix()

	supplier := &Supplier{
		Name:      "供应商",
		Status:    SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, DB.Create(supplier).Error)

	// Two different sheets
	sheet1 := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "报价单1",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(sheet1).Error)

	sheet2 := &SupplierPricingSheet{
		SupplierId: supplier.Id,
		ChannelId:  0,
		Name:       "报价单2",
		Status:     SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, DB.Create(sheet2).Error)

	// Same model in different sheets should be allowed
	item1 := &SupplierPricingItem{
		PricingSheetId: sheet1.Id,
		Models:           "gpt-4o",
		DiscountType:     DiscountTypeRatio,
		DiscountValue:    0.7,
	}
	require.NoError(t, DB.Create(item1).Error)

	item2 := &SupplierPricingItem{
		PricingSheetId: sheet2.Id,
		Model:          "gpt-4o",
		DiscountType:   DiscountTypeRatio,
		DiscountValue:  0.6,
	}
	require.NoError(t, DB.Create(item2).Error)

	// Both should exist
	items1, err := GetSupplierPricingItemsBySheetId(sheet1.Id)
	require.NoError(t, err)
	assert.Len(t, items1, 1)

	items2, err := GetSupplierPricingItemsBySheetId(sheet2.Id)
	require.NoError(t, err)
	assert.Len(t, items2, 1)
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

func truncateSupplier(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		DB.Exec("DELETE FROM supplier_pricing_items")
		DB.Exec("DELETE FROM supplier_pricing_sheets")
		DB.Exec("DELETE FROM suppliers")
	})
}

// seedSupplier creates a supplier for testing
func seedSupplier(name string, status int) (*Supplier, error) {
	now := time.Now().Unix()
	s := &Supplier{
		Name:      name,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := DB.Create(s).Error; err != nil {
		return nil, err
	}
	return s, nil
}

// seedPricingSheet creates a pricing sheet for testing
func seedPricingSheet(supplierId int, channelId int, name string, status int, startTime, endTime int64) (*SupplierPricingSheet, error) {
	now := time.Now().Unix()
	sheet := &SupplierPricingSheet{
		SupplierId: supplierId,
		ChannelId:  channelId,
		Name:       name,
		Status:     status,
		StartTime:  startTime,
		EndTime:    endTime,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := DB.Create(sheet).Error; err != nil {
		return nil, err
	}
	return sheet, nil
}

// seedPricingItem creates a pricing item for testing
func seedPricingItem(sheetId int, model string, discountType string, discountValue float64) (*SupplierPricingItem, error) {
	item := &SupplierPricingItem{
		PricingSheetId: sheetId,
		Models:           model,
		DiscountType:     discountType,
		DiscountValue:    discountValue,
	}
	if err := DB.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}
