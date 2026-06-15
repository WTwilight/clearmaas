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

func TestGetGroupedSelectableModelsForUser_GroupsByModelSquareParent(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "grouped-user")
	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"gpt-4o", "gpt-4o-alias"}, model.DiscountTypeRatio, 0.5)

	vendor := seedServiceVendor(t, "OpenAI", "openai")
	seedServiceModelMeta(t, "gpt-4o", "GPT-4o", "chat,vision", vendor.Id)
	seedServiceChannel(t, 9001, "openai-route", "gpt-4o,gpt-4o-alias", `{"gpt-4o-alias":"gpt-4o"}`, "fast")
	seedServiceAbility(t, "default", "gpt-4o", 9001)
	seedServiceAbility(t, "default", "gpt-4o-alias", 9001)

	data, err := GetGroupedSelectableModelsForUser(user.Id)
	require.NoError(t, err)
	require.Len(t, data.Models, 1)
	assert.Equal(t, 1, data.Total)

	group := data.Models[0]
	assert.Equal(t, "gpt-4o", group.Name)
	assert.Equal(t, "GPT 4O", group.DisplayName)
	assert.Equal(t, "GPT-4o", group.Icon)
	assert.Equal(t, "OpenAI", group.Vendor)
	assert.Equal(t, "openai", group.VendorIcon)
	assert.Contains(t, group.Tags, "chat")
	assert.Greater(t, group.Context, int64(0))
	assert.Greater(t, group.MaxOutput, int64(0))

	require.Len(t, group.Versions, 2)
	versionsByModel := make(map[string]dtoAvailableVersionForTest, len(group.Versions))
	for _, version := range group.Versions {
		versionsByModel[version.Model] = dtoAvailableVersionForTest{
			upstreamKey:    version.UpstreamKey,
			channelID:      version.ChannelID,
			channelName:    version.ChannelName,
			channelTags:    version.ChannelTags,
			pricingSheetID: version.PricingSheetID,
			sheetName:      version.SheetName,
			source:         version.Source,
			vendorType:     version.VendorType,
			discountRatio:  version.DiscountRatio,
			originalInput:  version.Original.Input,
			discountInput:  version.Discounted.Input,
		}
	}

	alias := versionsByModel["gpt-4o-alias"]
	assert.Equal(t, "gpt-4o", alias.upstreamKey)
	assert.Equal(t, 9001, alias.channelID)
	assert.Equal(t, "openai-route", alias.channelName)
	assert.Contains(t, alias.channelTags, "fast")
	assert.Equal(t, platformSheet.Id, alias.pricingSheetID)
	assert.Equal(t, "平台默认", alias.sheetName)
	assert.Equal(t, "platform", alias.source)
	assert.Equal(t, "openai", alias.vendorType)
	assert.Equal(t, 0.5, alias.discountRatio)
	assert.Greater(t, alias.originalInput, 0.0)
	assert.Equal(t, alias.originalInput*0.5, alias.discountInput)
}

func TestGetGroupedSelectableModelsForUser_KeepsSelectableModelsWithoutModelSquareMeta(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "fallback-grouped-user")
	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"legacy-only-model"}, model.DiscountTypeRatio, 0.6)

	data, err := GetGroupedSelectableModelsForUser(user.Id)
	require.NoError(t, err)
	require.Len(t, data.Models, 1)
	assert.Equal(t, 1, data.Total)

	group := data.Models[0]
	assert.Equal(t, "legacy-only-model", group.Name)
	assert.Equal(t, "legacy-only-model", group.DisplayName)
	assert.Equal(t, "openai", group.Vendor)
	assert.Equal(t, "openai", group.VendorName)
	assert.Equal(t, 0.6, group.BestDiscountRatio)

	require.Len(t, group.Versions, 1)
	version := group.Versions[0]
	assert.Equal(t, "legacy-only-model", version.Model)
	assert.Equal(t, platformSheet.Id, version.PricingSheetID)
	assert.Equal(t, platformSheet.Id, version.SheetID)
	assert.Equal(t, "平台默认", version.SheetName)
	assert.Equal(t, "platform", version.Source)
	assert.Equal(t, "openai", version.VendorType)
	assert.Equal(t, 0.6, version.DiscountRatio)
	assert.Greater(t, version.InputOriginalPrice, 0.0)
	assert.Equal(t, version.InputOriginalPrice*0.6, version.InputDiscountedPrice)
}

func TestGetGroupedSelectableModelsForUser_FallbackGroupsByChannelMapping(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "mapped-fallback-user")
	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"gpt-4.1-c1"}, model.DiscountTypeRatio, 0.62)

	vendor := seedServiceVendor(t, "OpenAI", "OpenAI.Color")
	seedServiceModelMeta(t, "gpt-4.1", "OpenAI.Color", "chat", vendor.Id)
	seedServiceChannel(t, 9002, "openai-c-route", "gpt-4.1-c1", `{"gpt-4.1-c1":"gpt-4.1"}`, "官方标准版")

	data, err := GetGroupedSelectableModelsForUser(user.Id)
	require.NoError(t, err)
	require.Len(t, data.Models, 1)

	group := data.Models[0]
	assert.Equal(t, "gpt-4.1", group.Name)
	assert.Equal(t, "GPT 4.1", group.DisplayName)
	assert.Equal(t, "OpenAI.Color", group.Icon)
	assert.Equal(t, "OpenAI", group.VendorName)
	assert.Contains(t, group.Tags, "chat")

	require.Len(t, group.Versions, 1)
	version := group.Versions[0]
	assert.Equal(t, "gpt-4.1-c1", version.Model)
	assert.Equal(t, "gpt-4.1", version.UpstreamKey)
	assert.Equal(t, 9002, version.ChannelID)
	assert.Equal(t, "openai-c-route", version.ChannelName)
	assert.Contains(t, version.ChannelTags, "官方标准版")
	assert.Equal(t, 0.62, version.DiscountRatio)
}

func TestGetGroupedSelectableModelsForUser_FallbackGroupsByChannelSuffix(t *testing.T) {
	truncateServiceTestData(t)

	user := seedServiceUser(t, 1, "suffix-fallback-user")
	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"gpt-5.5-pro-b1"}, model.DiscountTypeRatio, 0.8)

	vendor := seedServiceVendor(t, "OpenAI", "OpenAI.Color")
	seedServiceModelMeta(t, "gpt-5.5-pro", "OpenAI.Color", "chat", vendor.Id)
	seedServiceChannel(t, 9003, "openai-b-route", "gpt-5.5-pro-b1", `{}`, "大额官方稳定版")

	data, err := GetGroupedSelectableModelsForUser(user.Id)
	require.NoError(t, err)
	require.Len(t, data.Models, 1)

	group := data.Models[0]
	assert.Equal(t, "gpt-5.5-pro", group.Name)
	assert.Equal(t, "GPT 5.5 PRO", group.DisplayName)
	assert.Equal(t, "OpenAI", group.VendorName)

	require.Len(t, group.Versions, 1)
	version := group.Versions[0]
	assert.Equal(t, "gpt-5.5-pro-b1", version.Model)
	assert.Equal(t, "gpt-5.5-pro", version.UpstreamKey)
	assert.Equal(t, 9003, version.ChannelID)
	assert.Contains(t, version.ChannelTags, "大额官方稳定版")
	assert.Equal(t, 0.8, version.DiscountRatio)
}

func TestGetModelSquareData_GroupsChannelSuffixAsVersion(t *testing.T) {
	truncateServiceTestData(t)

	vendor := seedServiceVendor(t, "OpenAI", "OpenAI.Color")
	seedServiceModelMeta(t, "gpt-5.5-pro", "OpenAI.Color", "chat", vendor.Id)

	platformE := seedServicePlatformEnterprise(t)
	platformSheet := seedServicePricingSheet(t, platformE.Id, "平台默认", model.PricingSheetStatusActive, now()-86400, now()+86400)
	seedServicePricingItem(t, platformSheet.Id, []string{"gpt-5.5-pro-b1"}, model.DiscountTypeRatio, 0.8)
	seedServiceChannel(t, 9004, "openai-b-route", "gpt-5.5-pro-b1", `{}`, "大额官方稳定版")
	seedServiceAbility(t, "default", "gpt-5.5-pro-b1", 9004)

	data, err := GetModelSquareData(ModelSquareQuery{})
	require.NoError(t, err)
	require.Len(t, data.Models, 1)

	item := data.Models[0]
	assert.Equal(t, "gpt-5.5-pro", item.Name)
	assert.Equal(t, "GPT 5.5 PRO", item.DisplayName)

	require.Len(t, item.Versions, 1)
	assert.Equal(t, "gpt-5.5-pro-b1", item.Versions[0].ModelName)
	assert.Equal(t, "gpt-5.5-pro", item.Versions[0].UpstreamKey)
	assert.Contains(t, item.Versions[0].ChannelTags, "大额官方稳定版")
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

type dtoAvailableVersionForTest struct {
	upstreamKey    string
	channelID      int
	channelName    string
	channelTags    []string
	pricingSheetID int
	sheetName      string
	source         string
	vendorType     string
	discountRatio  float64
	originalInput  float64
	discountInput  float64
}

func truncateServiceTestData(t *testing.T) {
	t.Helper()
	model.InvalidatePricingCache()
	// Truncate BEFORE the test runs so each test starts with a clean slate.
	// t.Cleanup is still registered so the DB is cleaned after all tests finish.
	model.DB.Exec("DELETE FROM abilities")
	model.DB.Exec("DELETE FROM channels")
	model.DB.Exec("DELETE FROM models")
	model.DB.Exec("DELETE FROM vendors")
	model.DB.Exec("DELETE FROM token_pricing_model_bindings")
	model.DB.Exec("DELETE FROM tokens")
	model.DB.Exec("DELETE FROM enterprise_user_bindings")
	model.DB.Exec("DELETE FROM enterprise_pricing_sheet_channels")
	model.DB.Exec("DELETE FROM enterprise_pricing_items")
	model.DB.Exec("DELETE FROM enterprise_pricing_sheets")
	model.DB.Where("ent_type <> ?", model.EnterpriseTypePlatform).Delete(&model.Enterprise{})
	model.DB.Exec("DELETE FROM users")
	// Ensure platform enterprise exists for all tests.
	ensurePlatformEnterprise(t)
	t.Cleanup(func() {
		model.InvalidatePricingCache()
		model.DB.Exec("DELETE FROM abilities")
		model.DB.Exec("DELETE FROM channels")
		model.DB.Exec("DELETE FROM models")
		model.DB.Exec("DELETE FROM vendors")
		model.DB.Exec("DELETE FROM token_pricing_model_bindings")
		model.DB.Exec("DELETE FROM tokens")
		model.DB.Exec("DELETE FROM enterprise_user_bindings")
		model.DB.Exec("DELETE FROM enterprise_pricing_sheet_channels")
		model.DB.Exec("DELETE FROM enterprise_pricing_items")
		model.DB.Exec("DELETE FROM enterprise_pricing_sheets")
		model.DB.Where("ent_type <> ?", model.EnterpriseTypePlatform).Delete(&model.Enterprise{})
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
	platform := &model.Enterprise{EntType: model.EnterpriseTypePlatform}
	err := model.DB.Where("ent_type = ?", model.EnterpriseTypePlatform).
		Assign(model.Enterprise{
			Name:      "平台",
			Status:    model.EnterpriseStatusEnabled,
			UpdatedAt: now(),
		}).
		FirstOrCreate(platform, model.Enterprise{
			Name:      "平台",
			EntType:   model.EnterpriseTypePlatform,
			Status:    model.EnterpriseStatusEnabled,
			CreatedAt: now(),
			UpdatedAt: now(),
		}).Error
	require.NoError(t, err)
}

func seedServicePlatformEnterprise(t *testing.T) *model.Enterprise {
	t.Helper()
	ensurePlatformEnterprise(t)
	var e model.Enterprise
	err := model.DB.Where("ent_type = ?", model.EnterpriseTypePlatform).First(&e).Error
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
		Name:         name,
		Status:       status,
		StartTime:    startTime,
		EndTime:      endTime,
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

func seedServiceVendor(t *testing.T, name string, icon string) *model.Vendor {
	t.Helper()
	vendor := &model.Vendor{
		Name:   name,
		Icon:   icon,
		Status: 1,
	}
	vendor.CreatedTime = now()
	vendor.UpdatedTime = now()
	require.NoError(t, model.DB.Create(vendor).Error)
	return vendor
}

func seedServiceModelMeta(t *testing.T, modelName string, icon string, tags string, vendorId int) *model.Model {
	t.Helper()
	meta := &model.Model{
		ModelName:   modelName,
		Icon:        icon,
		Tags:        tags,
		VendorID:    vendorId,
		Status:      1,
		NameRule:    model.NameRuleExact,
		CreatedTime: now(),
		UpdatedTime: now(),
	}
	require.NoError(t, model.DB.Create(meta).Error)
	return meta
}

func seedServiceChannel(t *testing.T, id int, name string, models string, mapping string, tag string) *model.Channel {
	t.Helper()
	priority := int64(10)
	channel := &model.Channel{
		Id:           id,
		Type:         1,
		Key:          "sk-test",
		Status:       1,
		Name:         name,
		Models:       models,
		ModelMapping: &mapping,
		Priority:     &priority,
		Tag:          &tag,
	}
	channel.CreatedTime = now()
	require.NoError(t, model.DB.Create(channel).Error)
	return channel
}

func seedServiceAbility(t *testing.T, group string, modelName string, channelId int) *model.Ability {
	t.Helper()
	priority := int64(10)
	ability := &model.Ability{
		Group:     group,
		Model:     modelName,
		ChannelId: channelId,
		Enabled:   true,
		Priority:  &priority,
		Weight:    1,
	}
	require.NoError(t, model.DB.Create(ability).Error)
	return ability
}
