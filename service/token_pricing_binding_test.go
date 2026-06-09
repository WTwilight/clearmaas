package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// GetSelectableModelsForUser — models available for token binding
// Data dimensions:
//   - User has no enterprise binding → returns platform sheet models
//   - User has enterprise binding with active sheet → returns enterprise sheet models
//   - User has enterprise binding but no active sheet → returns platform sheet models
//   - User has enterprise binding with empty sheet → returns platform sheet models
//   - Enterprise disabled → falls back to platform sheet
//   - Enterprise sheet expired → falls back to platform sheet
//   - Enterprise sheet in the future → falls back to platform sheet
//   - No platform sheet at all → returns empty
//   - Platform enterprise (type='platform') → returns platform sheet models
// ============================================================================

func TestGetSelectableModelsForUser_NoEnterpriseBinding(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "no-enterprise-user")

	// Create platform enterprise and sheet
	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"gpt-4o", "gpt-4o-mini"}, model.DiscountTypeRatio, 0.8)

	models := GetSelectableModelsForUser(user.Id)
	require.Len(t, models, 2)

	names := modelNamesFromResult(models)
	assert.Contains(t, names, "gpt-4o")
	assert.Contains(t, names, "gpt-4o-mini")
}

func TestGetSelectableModelsForUser_WithEnterpriseBinding_HasActiveSheet(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "enterprise-user")
	e := seedServiceEnterprise(t, "EnterpriseCorp", model.EnterpriseStatusEnabled)
	seedServiceUserBinding(t, e.Id, user.Id)

	sheet := seedServicePricingSheet(t, e.Id, "企业报价单", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, sheet.Id, []string{"gpt-4o", "gpt-4o-mini"}, model.DiscountTypeRatio, 0.7)

	models := GetSelectableModelsForUser(user.Id)
	require.Len(t, models, 2)

	names := modelNamesFromResult(models)
	assert.Contains(t, names, "gpt-4o")
	assert.Contains(t, names, "gpt-4o-mini")

	for _, m := range models {
		assert.Equal(t, "enterprise", m.Source, "source should be enterprise")
		assert.Equal(t, "企业报价单", m.SheetName)
		assert.Equal(t, 0.7, m.DiscountRatio)
	}
}

func TestGetSelectableModelsForUser_WithEnterpriseBinding_NoActiveSheet(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "no-sheet-user")
	e := seedServiceEnterprise(t, "NoSheetCorp", model.EnterpriseStatusEnabled)
	seedServiceUserBinding(t, e.Id, user.Id)

	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"gpt-4o", "gpt-4o-mini"}, model.DiscountTypeRatio, 0.9)

	models := GetSelectableModelsForUser(user.Id)
	require.Len(t, models, 2)

	names := modelNamesFromResult(models)
	assert.Contains(t, names, "gpt-4o")
	assert.Contains(t, names, "gpt-4o-mini")

	for _, m := range models {
		assert.Equal(t, "platform", m.Source)
		assert.Equal(t, "平台默认", m.SheetName)
	}
}

func TestGetSelectableModelsForUser_WithEnterpriseBinding_EmptySheet(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "empty-sheet-user")
	e := seedServiceEnterprise(t, "EmptySheetCorp", model.EnterpriseStatusEnabled)
	seedServiceUserBinding(t, e.Id, user.Id)
	seedServicePricingSheet(t, e.Id, "空报价单", model.PricingSheetStatusActive, now()-86400, now()+86400)

	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.9)

	models := GetSelectableModelsForUser(user.Id)
	require.Len(t, models, 1)
	assert.Equal(t, "gpt-4o", models[0].Model)
}

func TestGetSelectableModelsForUser_EnterpriseDisabled(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "disabled-ent-user")
	e := seedServiceEnterprise(t, "DisabledCorp", model.EnterpriseStatusDisabled)
	seedServiceUserBinding(t, e.Id, user.Id)

	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"gpt-4o", "claude-3-5-sonnet"}, model.DiscountTypeRatio, 0.9)

	models := GetSelectableModelsForUser(user.Id)
	require.Len(t, models, 2)

	for _, m := range models {
		assert.Equal(t, "platform", m.Source)
	}
}

func TestGetSelectableModelsForUser_EnterpriseSheetExpired(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "expired-sheet-user")
	e := seedServiceEnterprise(t, "ExpiredCorp", model.EnterpriseStatusEnabled)
	seedServiceUserBinding(t, e.Id, user.Id)
	// Expired sheet (ended yesterday)
	seedServicePricingSheet(t, e.Id, "已过期", model.PricingSheetStatusActive, now()-2*86400, now()-86400)

	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.9)

	models := GetSelectableModelsForUser(user.Id)
	require.Len(t, models, 1)
	assert.Equal(t, "gpt-4o", models[0].Model)
	assert.Equal(t, "platform", models[0].Source)
}

func TestGetSelectableModelsForUser_EnterpriseSheetFuture(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "future-sheet-user")
	e := seedServiceEnterprise(t, "FutureCorp", model.EnterpriseStatusEnabled)
	seedServiceUserBinding(t, e.Id, user.Id)
	// Sheet starts tomorrow
	seedServicePricingSheet(t, e.Id, "未来生效", model.PricingSheetStatusActive, now()+86400, now()+30*86400)

	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.9)

	models := GetSelectableModelsForUser(user.Id)
	require.Len(t, models, 1)
	assert.Equal(t, "platform", models[0].Source)
}

func TestGetSelectableModelsForUser_NoPlatformSheet(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "no-platform-user")

	models := GetSelectableModelsForUser(user.Id)
	require.Len(t, models, 0)
}

func TestGetSelectableModelsForUser_DuplicateModelAcrossItems(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "dup-vendor-user")
	e := seedServiceEnterprise(t, "DupVendorCorp", model.EnterpriseStatusEnabled)
	seedServiceUserBinding(t, e.Id, user.Id)

	sheet := seedServicePricingSheet(t, e.Id, "多供应商报价单", model.PricingSheetStatusActive, now()-86400, now()+86400)
	// Same model name in different items
	seedServicePricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.7)
	seedServicePricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.8)

	models := GetSelectableModelsForUser(user.Id)
	// Map deduplicates — only one entry for gpt-4o
	names := modelNamesFromResult(models)
	assert.Contains(t, names, "gpt-4o")
	require.Len(t, models, 1)
}

// ============================================================================
// Priority: Enterprise sheet beats platform sheet
// ============================================================================

func TestGetSelectableModelsForUser_EnterpriseSheetOverridesPlatform(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "priority-user")

	// Platform has gpt-4o at 0.9
	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.9)

	// Enterprise has gpt-4o at 0.6
	e := seedServiceEnterprise(t, "PriorityCorp", model.EnterpriseStatusEnabled)
	seedServiceUserBinding(t, e.Id, user.Id)
	entSheet := seedServicePricingSheet(t, e.Id, "企业报价单", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, entSheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.6)

	models := GetSelectableModelsForUser(user.Id)
	require.Len(t, models, 1)
	// Enterprise should win
	assert.Equal(t, "enterprise", models[0].Source)
	assert.Equal(t, 0.6, models[0].DiscountRatio)
	assert.Equal(t, "企业报价单", models[0].SheetName)
}

// ============================================================================
// Helper functions
// ============================================================================

func modelNamesFromResult(models []SelectableModelInfo) []string {
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.Model
	}
	return names
}

func truncateServiceTestData(t *testing.T) {
	t.Helper()
	// Truncate BEFORE the test runs so each test starts with a clean slate.
	// t.Cleanup is still registered so the DB is cleaned after all tests finish.
	model.DB.Exec("DELETE FROM token_pricing_model_bindings")
	model.DB.Exec("DELETE FROM tokens")
	model.DB.Exec("DELETE FROM enterprise_user_bindings")
	model.DB.Exec("DELETE FROM enterprise_pricing_sheet_channels")
	model.DB.Exec("DELETE FROM enterprise_pricing_items")
	model.DB.Exec("DELETE FROM enterprise_pricing_sheets")
	model.DB.Exec("DELETE FROM enterprises WHERE `ent_type` != ?", model.EnterpriseTypePlatform)
	model.DB.Exec("DELETE FROM users")
	// Ensure platform enterprise exists for all tests.
	ensurePlatformEnterprise(t)
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM token_pricing_model_bindings")
		model.DB.Exec("DELETE FROM tokens")
		model.DB.Exec("DELETE FROM enterprise_user_bindings")
		model.DB.Exec("DELETE FROM enterprise_pricing_sheet_channels")
		model.DB.Exec("DELETE FROM enterprise_pricing_items")
		model.DB.Exec("DELETE FROM enterprise_pricing_sheets")
		model.DB.Exec("DELETE FROM enterprises WHERE `ent_type` != ?", model.EnterpriseTypePlatform)
		model.DB.Exec("DELETE FROM users")
	})
}

func now() int64 {
	return time.Now().Unix()
}

func seedServiceUser(t *testing.T, userId int, username string) *model.User {
	t.Helper()
	u := &model.User{
		Id:       userId,
		Username: username,
		Status:   1,
		Quota:    1000,
		AffCode:  username,
	}
	u.CreatedAt = now()
	require.NoError(t, model.DB.Create(u).Error)
	return u
}

func seedServiceEnterprise(t *testing.T, name string, status int) *model.Enterprise {
	t.Helper()
	e := &model.Enterprise{Name: name, EntType: model.EnterpriseTypeEnterprise, Status: status}
	e.CreatedAt = now()
	e.UpdatedAt = now()
	require.NoError(t, model.DB.Create(e).Error)
	return e
}

func ensurePlatformEnterprise(t *testing.T) {
	t.Helper()
	// Platform enterprise is identified by type='platform', not by id=-1.
	// Use REPLACE to upsert: insert if not exists, or update if exists.
	model.DB.Exec(
		"INSERT OR REPLACE INTO enterprises (name, type, status, created_at, updated_at) VALUES ('平台', ?, 1, ?, ?)",
		model.EnterpriseTypePlatform, now(), now(),
	)
}

func seedServicePlatformEnterprise(t *testing.T) *model.Enterprise {
	t.Helper()
	ensurePlatformEnterprise(t)
	var e model.Enterprise
	err := model.DB.Where("type = ?", model.EnterpriseTypePlatform).First(&e).Error
	require.NoError(t, err)
	return &e
}

func seedServiceUserBinding(t *testing.T, enterpriseId, userId int) *model.EnterpriseUserBinding {
	t.Helper()
	b := &model.EnterpriseUserBinding{EnterpriseId: enterpriseId, UserId: userId}
	b.CreatedAt = now()
	require.NoError(t, model.DB.Create(b).Error)
	return b
}

func seedServicePricingSheet(t *testing.T, enterpriseId int, name string, status int, startTime, endTime int64) *model.EnterprisePricingSheet {
	t.Helper()
	s := &model.EnterprisePricingSheet{
		EnterpriseId: enterpriseId,
		Name:        name,
		Status:      status,
		StartTime:   startTime,
		EndTime:     endTime,
	}
	s.CreatedAt = now()
	s.UpdatedAt = now()
	require.NoError(t, model.DB.Create(s).Error)
	return s
}

func seedServicePricingItem(t *testing.T, sheetId int, models []string, discountType string, discountValue float64) *model.EnterprisePricingItem {
	t.Helper()
	item := &model.EnterprisePricingItem{
		PricingSheetId: sheetId,
		VendorType:     "openai",
		Models:         models,
		DiscountType:   discountType,
		DiscountValue:  discountValue,
	}
	require.NoError(t, model.DB.Create(item).Error)
	return item
}
