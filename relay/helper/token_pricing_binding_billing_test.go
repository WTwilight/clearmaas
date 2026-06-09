package helper

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: TestMain is defined in setup_test.go — all test files share the same setup.

// ============================================================================
// Billing chain priority: Token binding > Enterprise sheet > Platform sheet > Group ratio
// Data dimensions:
//   - Token has binding → uses token binding sheet for this model
//   - Token has no binding, user in enterprise → uses enterprise sheet
//   - Token has no binding, user not in enterprise → uses platform sheet
//   - Enterprise sheet model not matched → falls back to platform or group ratio
//   - Platform sheet model not matched → falls back to group ratio
//   - Token binding overrides enterprise sheet (even if enterprise has different price)
// ============================================================================

func TestBillingChain_TokenBindingOverridesEnterpriseSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	// Setup: platform sheet has gpt-4o at 0.9
	platformE := seedBillingEnterprise(t, -1, "平台", model.EnterpriseStatusEnabled)
	platformSheet := seedBillingSheet(t, platformE.Id, "平台报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, platformSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.9)

	// Setup: enterprise sheet has gpt-4o at 0.7
	e := seedBillingEnterprise(t, 0, "CorpA", model.EnterpriseStatusEnabled)
	entSheet := seedBillingSheet(t, e.Id, "企业报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, entSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.7)

	// Setup: user bound to enterprise
	user := seedBillingUser(t, 1, "chain-user")
	seedBillingBinding(t, e.Id, user.Id)

	// Setup: token with binding that sets gpt-4o to 0.5
	token := seedBillingToken(t, user.Id, "chain-token")
	seedTokenBinding(t, user.Id, token.Id, entSheet.Id, "gpt-4o")

	// Setup context: user in enterprise with group ratio = 1.5
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:           user.Id,
		TokenId:          token.Id,
		UsingGroup:       "default",
		OriginModelName:  "gpt-4o",
	}

	// Billing chain: token binding should win (0.5)
	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	// Token binding (0.5) overrides enterprise sheet (0.7)
	assert.Equal(t, 0.5, result.GroupRatio)
	assert.Equal(t, "token_pricing_binding", result.RatioSource)
}

func TestBillingChain_TokenBinding_ModelNotInBinding_FallsBackToEnterprise(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	platformE := seedBillingEnterprise(t, -1, "平台", model.EnterpriseStatusEnabled)
	platformSheet := seedBillingSheet(t, platformE.Id, "平台报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, platformSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.9)

	e := seedBillingEnterprise(t, 0, "CorpB", model.EnterpriseStatusEnabled)
	entSheet := seedBillingSheet(t, e.Id, "企业报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, entSheet.Id, "openai", []string{"gpt-4o", "gpt-4o-mini"}, model.DiscountTypeRatio, 0.7)

	user := seedBillingUser(t, 1, "fallback-user")
	seedBillingBinding(t, e.Id, user.Id)

	token := seedBillingToken(t, user.Id, "fallback-token")
	// Bind gpt-4o only; gpt-4o-mini is not bound
	seedTokenBinding(t, user.Id, token.Id, entSheet.Id, "gpt-4o")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o-mini", // not in token binding
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	// No token binding for gpt-4o-mini → enterprise sheet applies (0.7)
	assert.Equal(t, 0.7, result.GroupRatio)
	assert.Equal(t, "enterprise_pricing_sheet", result.RatioSource)
}

func TestBillingChain_NoTokenBinding_UserInEnterprise(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	platformE := seedBillingEnterprise(t, -1, "平台", model.EnterpriseStatusEnabled)
	platformSheet := seedBillingSheet(t, platformE.Id, "平台报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, platformSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.9)

	e := seedBillingEnterprise(t, 0, "CorpC", model.EnterpriseStatusEnabled)
	entSheet := seedBillingSheet(t, e.Id, "企业报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, entSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.6)

	user := seedBillingUser(t, 1, "ent-only-user")
	seedBillingBinding(t, e.Id, user.Id)

	token := seedBillingToken(t, user.Id, "ent-only-token")
	// No token bindings created

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o",
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	// Enterprise sheet applies (0.6)
	assert.Equal(t, 0.6, result.GroupRatio)
	assert.Equal(t, "enterprise_pricing_sheet", result.RatioSource)
}

func TestBillingChain_NoTokenBinding_UserNotInEnterprise(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	// Platform sheet
	platformE := seedBillingEnterprise(t, -1, "平台", model.EnterpriseStatusEnabled)
	platformSheet := seedBillingSheet(t, platformE.Id, "平台报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, platformSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.8)

	// User not bound to any enterprise
	user := seedBillingUser(t, 1, "no-ent-user")
	token := seedBillingToken(t, user.Id, "no-ent-token")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o",
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	// Platform sheet applies (0.8)
	assert.Equal(t, 0.8, result.GroupRatio)
	assert.Equal(t, "platform_pricing_sheet", result.RatioSource)
}

func TestBillingChain_ModelNotInEnterprise_FallsBackToPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	platformE := seedBillingEnterprise(t, -1, "平台", model.EnterpriseStatusEnabled)
	platformSheet := seedBillingSheet(t, platformE.Id, "平台报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, platformSheet.Id, "openai", []string{"gpt-4o", "gpt-4o-mini"}, model.DiscountTypeRatio, 0.85)

	e := seedBillingEnterprise(t, 0, "CorpD", model.EnterpriseStatusEnabled)
	entSheet := seedBillingSheet(t, e.Id, "企业报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	// Enterprise sheet only has gpt-4o, not gpt-4o-mini
	seedBillingItem(t, entSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.5)

	user := seedBillingUser(t, 1, "partial-match-user")
	seedBillingBinding(t, e.Id, user.Id)

	token := seedBillingToken(t, user.Id, "partial-token")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o-mini", // not in enterprise sheet
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	// Falls back to platform sheet (0.85)
	assert.Equal(t, 0.85, result.GroupRatio)
	assert.Equal(t, "platform_pricing_sheet", result.RatioSource)
}

func TestBillingChain_ModelNotInAnySheet_FallsBackToGroupRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	platformE := seedBillingEnterprise(t, -1, "平台", model.EnterpriseStatusEnabled)
	platformSheet := seedBillingSheet(t, platformE.Id, "平台报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, platformSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.8)

	e := seedBillingEnterprise(t, 0, "CorpE", model.EnterpriseStatusEnabled)
	entSheet := seedBillingSheet(t, e.Id, "企业报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, entSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.6)

	user := seedBillingUser(t, 1, "unknown-model-user")
	seedBillingBinding(t, e.Id, user.Id)

	token := seedBillingToken(t, user.Id, "unknown-token")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "completely-unknown-model-xyz",
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	// No sheet covers this model → falls back to group ratio (1.0)
	assert.Equal(t, 1.0, result.GroupRatio)
	assert.Equal(t, "group_ratio", result.RatioSource)
}

func TestBillingChain_TokenBindingWithPlatformSheet_OverridesPlatform(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	// Platform sheet
	platformE := seedBillingEnterprise(t, -1, "平台", model.EnterpriseStatusEnabled)
	platformSheet := seedBillingSheet(t, platformE.Id, "平台报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, platformSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.8)

	// User not in any enterprise
	user := seedBillingUser(t, 1, "platform-bind-user")
	token := seedBillingToken(t, user.Id, "platform-bind-token")

	// Token binding uses platform sheet
	seedTokenBinding(t, user.Id, token.Id, platformSheet.Id, "gpt-4o")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o",
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	// Token binding uses platform sheet (0.8), and token binding overrides
	assert.Equal(t, 0.8, result.GroupRatio)
	assert.Equal(t, "token_pricing_binding", result.RatioSource)
}

func TestBillingChain_EnterpriseSheetOverridesPlatformSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	// Platform sheet has gpt-4o at 0.9
	platformE := seedBillingEnterprise(t, -1, "平台", model.EnterpriseStatusEnabled)
	platformSheet := seedBillingSheet(t, platformE.Id, "平台报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, platformSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.9)

	// Enterprise sheet has gpt-4o at 0.5
	e := seedBillingEnterprise(t, 0, "CorpF", model.EnterpriseStatusEnabled)
	entSheet := seedBillingSheet(t, e.Id, "企业报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, entSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.5)

	user := seedBillingUser(t, 1, "ent-wins-user")
	seedBillingBinding(t, e.Id, user.Id)
	token := seedBillingToken(t, user.Id, "ent-wins-token")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o",
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	// Enterprise (0.5) should beat platform (0.9)
	assert.Equal(t, 0.5, result.GroupRatio)
	assert.Equal(t, "enterprise_pricing_sheet", result.RatioSource)
}

// ============================================================================
// RatioSource values — trace field verification
// ============================================================================

func TestBillingChain_RatioSource_TokenBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	platformE := seedBillingEnterprise(t, -1, "平台", model.EnterpriseStatusEnabled)
	platformSheet := seedBillingSheet(t, platformE.Id, "平台报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, platformSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.8)

	user := seedBillingUser(t, 1, "source-user")
	token := seedBillingToken(t, user.Id, "source-token")
	seedTokenBinding(t, user.Id, token.Id, platformSheet.Id, "gpt-4o")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o",
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	assert.Equal(t, "token_pricing_binding", result.RatioSource)
	assert.True(t, result.HasSpecialRatio)
}

func TestBillingChain_RatioSource_EnterpriseSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	e := seedBillingEnterprise(t, 0, "CorpG", model.EnterpriseStatusEnabled)
	entSheet := seedBillingSheet(t, e.Id, "企业报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, entSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.6)

	user := seedBillingUser(t, 1, "ent-source-user")
	seedBillingBinding(t, e.Id, user.Id)
	token := seedBillingToken(t, user.Id, "ent-source-token")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o",
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	assert.Equal(t, "enterprise_pricing_sheet", result.RatioSource)
	assert.True(t, result.HasSpecialRatio)
	assert.Equal(t, entSheet.Id, result.EnterpriseSheetId)
}

func TestBillingChain_RatioSource_PlatformSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	platformE := seedBillingEnterprise(t, -1, "平台", model.EnterpriseStatusEnabled)
	platformSheet := seedBillingSheet(t, platformE.Id, "平台报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, platformSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.9)

	user := seedBillingUser(t, 1, "plat-source-user")
	token := seedBillingToken(t, user.Id, "plat-source-token")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o",
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	assert.Equal(t, "platform_pricing_sheet", result.RatioSource)
	assert.True(t, result.HasSpecialRatio)
}

func TestBillingChain_RatioSource_GroupRatio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	// No sheets at all
	user := seedBillingUser(t, 1, "group-source-user")
	token := seedBillingToken(t, user.Id, "group-source-token")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o",
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	assert.Equal(t, "group_ratio", result.RatioSource)
	assert.Equal(t, 1.0, result.GroupRatio)
	assert.False(t, result.HasSpecialRatio)
}

// ============================================================================
// Complete billing chain: Token binding disabled → enterprise takes over
// ============================================================================

func TestBillingChain_TokenBinding_DisabledEnterprise(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncateBillingChain(t)

	now := time.Now().Unix()

	e := seedBillingEnterprise(t, 0, "DisabledCorp", model.EnterpriseStatusDisabled)
	entSheet := seedBillingSheet(t, e.Id, "企业报价单", model.PricingSheetStatusActive, now-86400, now+86400)
	seedBillingItem(t, entSheet.Id, "openai", []string{"gpt-4o"}, model.DiscountTypeRatio, 0.5)

	user := seedBillingUser(t, 1, "disabled-ent-user")
	seedBillingBinding(t, e.Id, user.Id)
	token := seedBillingToken(t, user.Id, "disabled-ent-token")
	seedTokenBinding(t, user.Id, token.Id, entSheet.Id, "gpt-4o")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("group", "default")
	c.Set("id", user.Id)

	info := &relaycommon.RelayInfo{
		UserId:          user.Id,
		TokenId:         token.Id,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o",
	}

	result, _ := HandleGroupRatio(c, info)
	result = HandleEnterprisePricingSheet(c, info, result)

	// Disabled enterprise → enterprise sheet not applied
	// Token binding should still be used since it references the sheet directly
	assert.Equal(t, 0.5, result.GroupRatio)
	assert.Equal(t, "token_pricing_binding", result.RatioSource)
}

// ============================================================================
// Helper seeders for billing chain tests
// ============================================================================

func truncateBillingChain(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM token_pricing_model_bindings")
		model.DB.Exec("DELETE FROM tokens")
		// Do NOT delete enterprise_pricing_sheets or enterprise_pricing_items:
		// They are the shared baseline seeded by seedBillingTestData and reused across tests.
		// Only billing chain test cases that create new sheets/items should manage their own cleanup.
		// enterprise_user_bindings are also shared baseline — do NOT delete them.
	})
	// Re-create platform enterprise so it persists across truncate cycles.
	ensurePlatformEnterpriseForBilling(t)
	// NOTE: Do NOT call seedBillingTestData here. The baseline users, enterprises,
	// and bindings (users 1,2,3; enterprises 1,2,3) are set up once in TestMain
	// and reused across billing chain tests. Each billing chain test creates its
	// own additional test fixtures (sheets, tokens, bindings) that are cleaned up
	// by the t.Cleanup above.
}

func seedBillingEnterprise(t *testing.T, id int, name string, status int) *model.Enterprise {
	t.Helper()
	now := time.Now().Unix()
	if id == -1 {
		// Platform enterprise is identified by type='platform', not by id=-1.
		ensurePlatformEnterpriseForBilling(t)
		var e model.Enterprise
		err := model.DB.Where("`ent_type` = ?", model.EnterpriseTypePlatform).First(&e).Error
		require.NoError(t, err)
		return &e
	}
	e := &model.Enterprise{Name: name, EntType: model.EnterpriseTypeEnterprise, Status: status}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, model.DB.Create(e).Error)
	return e
}

func seedBillingUser(t *testing.T, id int, username string) *model.User {
	t.Helper()
	u := &model.User{
		Id:       id,
		Username: username,
		Status:   1,
		Quota:    1000,
		AffCode:  username,
	}
	u.CreatedAt = time.Now().Unix()
	require.NoError(t, model.DB.Save(u).Error)
	return u
}

func seedBillingToken(t *testing.T, userId int, key string) *model.Token {
	t.Helper()
	tk := &model.Token{
		UserId:  userId,
		Name:    key + "-token",
		Key:     key,
		Status:  1,
		Group:   "default",
	}
	tk.CreatedTime = time.Now().Unix()
	tk.AccessedTime = time.Now().Unix()
	tk.ExpiredTime = -1
	require.NoError(t, model.DB.Create(tk).Error)
	return tk
}

func seedBillingBinding(t *testing.T, enterpriseId, userId int) *model.EnterpriseUserBinding {
	t.Helper()
	b := &model.EnterpriseUserBinding{EnterpriseId: enterpriseId, UserId: userId}
	b.CreatedAt = time.Now().Unix()
	require.NoError(t, model.DB.Create(b).Error)
	return b
}

func seedBillingSheet(t *testing.T, enterpriseId int, name string, status int, startTime, endTime int64) *model.EnterprisePricingSheet {
	t.Helper()
	s := &model.EnterprisePricingSheet{
		EnterpriseId: enterpriseId,
		Name:        name,
		Status:      status,
		StartTime:   startTime,
		EndTime:     endTime,
	}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, model.DB.Create(s).Error)
	return s
}

func seedBillingItem(t *testing.T, sheetId int, vendorType string, models []string, discountType string, discountValue float64) *model.EnterprisePricingItem {
	t.Helper()
	item := &model.EnterprisePricingItem{
		PricingSheetId: sheetId,
		VendorType:    vendorType,
		Models:        models,
		DiscountType:  discountType,
		DiscountValue: discountValue,
	}
	require.NoError(t, model.DB.Create(item).Error)
	return item
}

func seedTokenBinding(t *testing.T, userId, tokenId, sheetId int, modelName string) *model.TokenPricingModelBinding {
	t.Helper()
	b := &model.TokenPricingModelBinding{
		UserId:        userId,
		TokenId:       tokenId,
		PricingSheetId: sheetId,
		Model:         modelName,
	}
	b.CreatedAt = time.Now().Unix()
	require.NoError(t, b.Create())
	return b
}
