package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	helper "github.com/QuantumNous/new-api/relay/helper"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetRatioSetting is defined in enterprise_billing_test.go and shared within the service package.

// ============================================================================
// Test 1: Three-layer billing chain — enterprise pricing + supplier pricing together
//   Enterprise sheet overrides customer ratio, supplier sheet records cost.
//   Log must contain: ratio_source, enterprise_sheet_*, supplier_cost, etc.
// ============================================================================

func TestThreeLayerBilling_EnterpriseAndSupplierPricing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID    = 3001
		testChannelID = 3001
		modelName     = "claude-opus-4-7-max"
	)

	// -------------------------------------------------------------------------
	// 1. Inject model ratio: claude-opus-4-7-max = 1.0
	// -------------------------------------------------------------------------
	err := ratio_setting.UpdateModelRatioByJSONString(`{"claude-opus-4-7-max": 1.0}`)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 2. Set up enterprise + pricing sheet + item (customer ratio = 0.9)
	// -------------------------------------------------------------------------
	now := time.Now().Unix()

	enterprise := &model.Enterprise{
		Name:      "TestCorp-ThreeLayer",
		Status:    model.EnterpriseStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(enterprise).Error)

	enterpriseSheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterprise.Id,
		Name:         "企业报价单",
		Status:       model.PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, model.DB.Create(enterpriseSheet).Error)

	enterpriseItem := &model.EnterprisePricingItem{
		PricingSheetId: enterpriseSheet.Id,
		Models:        []string{modelName},
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.9,
	}
	require.NoError(t, model.DB.Create(enterpriseItem).Error)

	enterpriseSheetChannel := &model.EnterprisePricingSheetChannel{
		PricingSheetId: enterpriseSheet.Id,
		ChannelId:      testChannelID,
		CreatedAt:      now,
	}
	require.NoError(t, model.DB.Create(enterpriseSheetChannel).Error)

	user := &model.User{
		Id:       testUserID,
		Username:  "enterprise_user_three_layer",
		Quota:     100000,
		Status:    common.UserStatusEnabled,
		CreatedAt: now,
	}
	require.NoError(t, model.DB.Create(user).Error)

	enterpriseBinding := &model.EnterpriseUserBinding{
		EnterpriseId: enterprise.Id,
		UserId:       testUserID,
		CreatedAt:    now,
	}
	require.NoError(t, model.DB.Create(enterpriseBinding).Error)

	// -------------------------------------------------------------------------
	// 3. Set up supplier + pricing sheet + item + channel binding (cost = 0.9 ratio)
	// -------------------------------------------------------------------------
	supplier := &model.Supplier{
		Name:      "TestSupplierA",
		Status:    model.SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(supplier).Error)

	supplierSheet := &model.SupplierPricingSheet{
		SupplierId: supplier.Id,
		Name:       "供应商报价单A",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, model.DB.Create(supplierSheet).Error)

	supplierItem := &model.SupplierPricingItem{
		PricingSheetId: supplierSheet.Id,
		VendorType:     "anthropic",
		Models:         []string{modelName},
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.9,
	}
	require.NoError(t, model.DB.Create(supplierItem).Error)

	supplierSheetChannel := &model.SupplierPricingSheetChannel{
		PricingSheetId: supplierSheet.Id,
		ChannelId:      testChannelID,
		CreatedAt:      now,
	}
	require.NoError(t, model.DB.Create(supplierSheetChannel).Error)

	channel := &model.Channel{
		Id:     testChannelID,
		Name:   "test_channel_three_layer",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, model.DB.Create(channel).Error)

	// -------------------------------------------------------------------------
	// 4. Build RelayInfo and run ModelPriceHelper
	// -------------------------------------------------------------------------
	info := &relaycommon.RelayInfo{
		UserId:           testUserID,
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelId: testChannelID},
		UsingGroup:       "default",
		OriginModelName:  modelName,
		TokenId:          0,
		IsStream:         false,
		StartTime:        time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.UserSetting.AcceptUnsetRatioModel = false

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", "default")
	c.Set("token_name", "test_token")

	promptTokens := 1000
	completionTokens := 200

	_, err = helper.ModelPriceHelper(c, info, promptTokens, nil)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 5. Assert: GroupRatio should be overridden by enterprise pricing sheet
	// -------------------------------------------------------------------------
	assert.Equal(t, 0.9, info.PriceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should be overridden by enterprise pricing sheet (0.9)")
	assert.Equal(t, "enterprise_pricing_sheet", info.PriceData.GroupRatioInfo.RatioSource,
		"RatioSource should be enterprise_pricing_sheet")
	assert.Equal(t, enterpriseSheet.Id, info.PriceData.GroupRatioInfo.EnterpriseSheetId,
		"EnterpriseSheetId should be set")
	assert.Equal(t, enterpriseSheet.Name, info.PriceData.GroupRatioInfo.EnterpriseSheetName,
		"EnterpriseSheetName should be set")

	// -------------------------------------------------------------------------
	// 6. Assert: SupplierCost should be set independently (does NOT affect GroupRatio)
	// -------------------------------------------------------------------------
	assert.Equal(t, 0.9, info.PriceData.GroupRatioInfo.SupplierCost,
		"SupplierCost should be 0.9 from supplier pricing sheet")
	assert.Equal(t, model.DiscountTypeRatio, info.PriceData.GroupRatioInfo.SupplierCostType,
		"SupplierCostType should be ratio")
	assert.Equal(t, supplierSheet.Id, info.PriceData.GroupRatioInfo.SupplierSheetId,
		"SupplierSheetId should be set")
	assert.Equal(t, supplierSheet.Name, info.PriceData.GroupRatioInfo.SupplierSheetName,
		"SupplierSheetName should be set")

	// -------------------------------------------------------------------------
	// 7. Simulate usage and run PostTextConsumeQuota
	// -------------------------------------------------------------------------
	usage := &dto.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 0,
		},
	}

	PostTextConsumeQuota(c, info, usage, nil)

	// -------------------------------------------------------------------------
	// 8. Assert: consume log should contain ALL three-layer billing fields
	// -------------------------------------------------------------------------
	log := getLastLog(t)
	require.NotNil(t, log, "consume log should be created")
	assert.Equal(t, model.LogTypeConsume, log.Type)
	assert.Equal(t, modelName, log.ModelName)
	assert.Greater(t, log.Quota, 0)

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err, "Other field should be valid JSON")

	// Enterprise pricing fields
	assert.Equal(t, "enterprise_pricing_sheet", other["ratio_source"],
		"log should record ratio_source = enterprise_pricing_sheet")
	assert.Equal(t, float64(enterpriseSheet.Id), other["enterprise_sheet_id"],
		"log should record enterprise_sheet_id")
	assert.Equal(t, enterpriseSheet.Name, other["enterprise_sheet_name"],
		"log should record enterprise_sheet_name")
	assert.Equal(t, 0.9, other["discount_ratio"],
		"log should record discount_ratio = 0.9 (enterprise pricing)")

	// Supplier pricing fields
	assert.Equal(t, 0.9, other["supplier_cost"],
		"log should record supplier_cost = 0.9")
	assert.Equal(t, "ratio", other["supplier_cost_type"],
		"log should record supplier_cost_type = ratio")
	assert.Equal(t, float64(supplierSheet.Id), other["supplier_sheet_id"],
		"log should record supplier_sheet_id")
	assert.Equal(t, supplierSheet.Name, other["supplier_sheet_name"],
		"log should record supplier_sheet_name")

	// Other important fields
	assert.Greater(t, other["original_price"].(float64), float64(0),
		"original_price should be recorded")
}

// ============================================================================
// Test 2: Supplier pricing only — no enterprise, supplier_cost still written
//   Customer is billed at group ratio. Supplier cost is recorded independently.
// ============================================================================

func TestThreeLayerBilling_SupplierPricingOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID    = 4001
		testChannelID = 4001
		modelName     = "claude-opus-4-7-max"
		groupRatio    = 1.5
	)

	// -------------------------------------------------------------------------
	// 1. Inject model ratio + group ratio
	// -------------------------------------------------------------------------
	err := ratio_setting.UpdateModelRatioByJSONString(`{"claude-opus-4-7-max": 1.0}`)
	require.NoError(t, err)
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1.5, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 2. Set up supplier only (NO enterprise binding)
	// -------------------------------------------------------------------------
	now := time.Now().Unix()

	supplier := &model.Supplier{
		Name:      "TestSupplierOnly",
		Status:    model.SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(supplier).Error)

	supplierSheet := &model.SupplierPricingSheet{
		SupplierId: supplier.Id,
		Name:       "供应商报价单-仅成本",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, model.DB.Create(supplierSheet).Error)

	supplierItem := &model.SupplierPricingItem{
		PricingSheetId: supplierSheet.Id,
		VendorType:     "anthropic",
		Models:         []string{modelName},
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.6,
	}
	require.NoError(t, model.DB.Create(supplierItem).Error)

	supplierSheetChannel := &model.SupplierPricingSheetChannel{
		PricingSheetId: supplierSheet.Id,
		ChannelId:      testChannelID,
		CreatedAt:      now,
	}
	require.NoError(t, model.DB.Create(supplierSheetChannel).Error)

	user := &model.User{
		Id:       testUserID,
		Username:  "normal_user_supplier_only",
		Quota:     100000,
		Status:    common.UserStatusEnabled,
		CreatedAt: now,
	}
	require.NoError(t, model.DB.Create(user).Error)

	channel := &model.Channel{
		Id:     testChannelID,
		Name:   "test_channel_supplier_only",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, model.DB.Create(channel).Error)

	// -------------------------------------------------------------------------
	// 3. Build RelayInfo and run ModelPriceHelper
	// -------------------------------------------------------------------------
	info := &relaycommon.RelayInfo{
		UserId:           testUserID,
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelId: testChannelID},
		UsingGroup:       "default",
		OriginModelName:  modelName,
		TokenId:          0,
		IsStream:         false,
		StartTime:        time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.UserSetting.AcceptUnsetRatioModel = false

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", "default")
	c.Set("token_name", "test_token")

	promptTokens := 1000
	completionTokens := 200

	_, err = helper.ModelPriceHelper(c, info, promptTokens, nil)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 4. Assert: GroupRatio comes from group ratio (not overridden)
	// -------------------------------------------------------------------------
	assert.Equal(t, groupRatio, info.PriceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should come from group ratio (1.5) since no enterprise pricing")
	assert.Equal(t, "group_ratio", info.PriceData.GroupRatioInfo.RatioSource,
		"RatioSource should be group_ratio")
	assert.Equal(t, 0, info.PriceData.GroupRatioInfo.EnterpriseSheetId,
		"EnterpriseSheetId should be 0 (no enterprise)")

	// -------------------------------------------------------------------------
	// 5. Assert: SupplierCost should still be set
	// -------------------------------------------------------------------------
	assert.Equal(t, 0.6, info.PriceData.GroupRatioInfo.SupplierCost,
		"SupplierCost should be 0.6 from supplier pricing sheet")
	assert.Equal(t, "ratio", info.PriceData.GroupRatioInfo.SupplierCostType,
		"SupplierCostType should be ratio")
	assert.Equal(t, supplierSheet.Id, info.PriceData.GroupRatioInfo.SupplierSheetId,
		"SupplierSheetId should be set")

	// -------------------------------------------------------------------------
	// 6. Simulate usage and run PostTextConsumeQuota
	// -------------------------------------------------------------------------
	usage := &dto.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 0,
		},
	}

	PostTextConsumeQuota(c, info, usage, nil)

	// -------------------------------------------------------------------------
	// 7. Assert: consume log
	// -------------------------------------------------------------------------
	log := getLastLog(t)
	require.NotNil(t, log)

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)

	// Group ratio fields
	assert.Equal(t, "group_ratio", other["ratio_source"],
		"ratio_source should be group_ratio (no enterprise)")
	assert.Equal(t, groupRatio, other["discount_ratio"],
		"discount_ratio should be 1.5 (group ratio)")

	// Supplier fields present
	assert.Equal(t, 0.6, other["supplier_cost"],
		"supplier_cost should be recorded even without enterprise pricing")
	assert.Equal(t, "ratio", other["supplier_cost_type"])
	assert.Equal(t, float64(supplierSheet.Id), other["supplier_sheet_id"])
	assert.Equal(t, supplierSheet.Name, other["supplier_sheet_name"])

	// Enterprise fields absent
	_, hasEnterpriseSheetId := other["enterprise_sheet_id"]
	assert.False(t, hasEnterpriseSheetId,
		"enterprise_sheet_id should not be present when no enterprise pricing")
}

// ============================================================================
// Test 3: Three discount types — ratio, fixed_price, per_call
//   Verifies that supplier_cost and supplier_cost_type are written correctly.
// ============================================================================

func testThreeLayerBilling_DiscountType(t *testing.T, discountType string, discountValue float64, expectedCostType string) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID    = 5001
		testChannelID = 5001
		modelName     = "claude-opus-4-7-max"
	)

	err := ratio_setting.UpdateModelRatioByJSONString(`{"claude-opus-4-7-max": 1.0}`)
	require.NoError(t, err)

	now := time.Now().Unix()

	supplier := &model.Supplier{
		Name:      "TestSupplier-" + discountType,
		Status:    model.SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(supplier).Error)

	supplierSheet := &model.SupplierPricingSheet{
		SupplierId: supplier.Id,
		Name:       "供应商报价单-" + discountType,
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, model.DB.Create(supplierSheet).Error)

	supplierItem := &model.SupplierPricingItem{
		PricingSheetId: supplierSheet.Id,
		VendorType:     "anthropic",
		Models:         []string{modelName},
		DiscountType:   discountType,
		DiscountValue:  discountValue,
	}
	require.NoError(t, model.DB.Create(supplierItem).Error)

	supplierSheetChannel := &model.SupplierPricingSheetChannel{
		PricingSheetId: supplierSheet.Id,
		ChannelId:      testChannelID,
		CreatedAt:      now,
	}
	require.NoError(t, model.DB.Create(supplierSheetChannel).Error)

	user := &model.User{
		Id:       testUserID,
		Username:  "user_" + discountType,
		Quota:     100000,
		Status:    common.UserStatusEnabled,
		CreatedAt: now,
	}
	require.NoError(t, model.DB.Create(user).Error)

	channel := &model.Channel{
		Id:     testChannelID,
		Name:   "channel_" + discountType,
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, model.DB.Create(channel).Error)

	info := &relaycommon.RelayInfo{
		UserId:           testUserID,
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelId: testChannelID},
		UsingGroup:       "default",
		OriginModelName:  modelName,
		TokenId:          0,
		IsStream:         false,
		StartTime:        time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.UserSetting.AcceptUnsetRatioModel = false

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", "default")
	c.Set("token_name", "test_token")

	promptTokens := 1000
	completionTokens := 200

	_, err = helper.ModelPriceHelper(c, info, promptTokens, nil)
	require.NoError(t, err)

	// Assert GroupRatioInfo fields
	assert.Equal(t, discountValue, info.PriceData.GroupRatioInfo.SupplierCost,
		"SupplierCost should be %v for discountType=%s", discountValue, discountType)
	assert.Equal(t, expectedCostType, info.PriceData.GroupRatioInfo.SupplierCostType,
		"SupplierCostType should be %s for discountType=%s", expectedCostType, discountType)
	assert.Equal(t, supplierSheet.Id, info.PriceData.GroupRatioInfo.SupplierSheetId)

	usage := &dto.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 0,
		},
	}

	PostTextConsumeQuota(c, info, usage, nil)

	log := getLastLog(t)
	require.NotNil(t, log)

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)

	assert.Equal(t, discountValue, other["supplier_cost"],
		"log: supplier_cost should be %v for discountType=%s", discountValue, discountType)
	assert.Equal(t, expectedCostType, other["supplier_cost_type"],
		"log: supplier_cost_type should be %s for discountType=%s", expectedCostType, discountType)
	assert.Equal(t, float64(supplierSheet.Id), other["supplier_sheet_id"])
	assert.Equal(t, supplierSheet.Name, other["supplier_sheet_name"])
}

func TestThreeLayerBilling_DiscountTypeRatio(t *testing.T) {
	testThreeLayerBilling_DiscountType(t, model.DiscountTypeRatio, 0.75, "ratio")
}

func TestThreeLayerBilling_DiscountTypeFixedPrice(t *testing.T) {
	testThreeLayerBilling_DiscountType(t, model.DiscountTypeFixedPrice, 0.004, "fixed_price")
}

func TestThreeLayerBilling_DiscountTypePerCall(t *testing.T) {
	testThreeLayerBilling_DiscountType(t, model.DiscountTypePerCall, 0.01, "per_call")
}

// ============================================================================
// Test 4: Enterprise sheet model not matched — fallback to group ratio,
//   but supplier_cost should still be recorded (supplier is independent).
// ============================================================================

func TestThreeLayerBilling_EnterpriseFallbackWithSupplierCost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID    = 6001
		testChannelID = 6001
		modelName     = "claude-opus-4-7-max"
		groupRatio    = 1.5
	)

	err := ratio_setting.UpdateModelRatioByJSONString(`{"claude-opus-4-7-max": 1.0}`)
	require.NoError(t, err)
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1.5, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	now := time.Now().Unix()

	// Enterprise sheet with a DIFFERENT model
	enterprise := &model.Enterprise{
		Name:      "TestCorp-DiffModel",
		Status:    model.EnterpriseStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(enterprise).Error)

	enterpriseSheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterprise.Id,
		Name:         "企业报价单-不同模型",
		Status:       model.PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, model.DB.Create(enterpriseSheet).Error)

	enterpriseItem := &model.EnterprisePricingItem{
		PricingSheetId: enterpriseSheet.Id,
		Models:        []string{"gpt-4o"}, // DIFFERENT model
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.3, // This should NOT be used
	}
	require.NoError(t, model.DB.Create(enterpriseItem).Error)

	enterpriseSheetChannel := &model.EnterprisePricingSheetChannel{
		PricingSheetId: enterpriseSheet.Id,
		ChannelId:      testChannelID,
		CreatedAt:      now,
	}
	require.NoError(t, model.DB.Create(enterpriseSheetChannel).Error)

	user := &model.User{
		Id:       testUserID,
		Username:  "user_fallback",
		Quota:     100000,
		Status:    common.UserStatusEnabled,
		CreatedAt: now,
	}
	require.NoError(t, model.DB.Create(user).Error)

	enterpriseBinding := &model.EnterpriseUserBinding{
		EnterpriseId: enterprise.Id,
		UserId:       testUserID,
		CreatedAt:    now,
	}
	require.NoError(t, model.DB.Create(enterpriseBinding).Error)

	// Supplier sheet still matches the model
	supplier := &model.Supplier{
		Name:      "TestSupplierFallback",
		Status:    model.SupplierStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(supplier).Error)

	supplierSheet := &model.SupplierPricingSheet{
		SupplierId: supplier.Id,
		Name:       "供应商报价单-降级测试",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, model.DB.Create(supplierSheet).Error)

	supplierItem := &model.SupplierPricingItem{
		PricingSheetId: supplierSheet.Id,
		VendorType:     "anthropic",
		Models:         []string{modelName}, // Matches
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.6,
	}
	require.NoError(t, model.DB.Create(supplierItem).Error)

	supplierSheetChannel := &model.SupplierPricingSheetChannel{
		PricingSheetId: supplierSheet.Id,
		ChannelId:      testChannelID,
		CreatedAt:      now,
	}
	require.NoError(t, model.DB.Create(supplierSheetChannel).Error)

	channel := &model.Channel{
		Id:     testChannelID,
		Name:   "test_channel_fallback",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, model.DB.Create(channel).Error)

	info := &relaycommon.RelayInfo{
		UserId:           testUserID,
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelId: testChannelID},
		UsingGroup:       "default",
		OriginModelName:  modelName,
		TokenId:          0,
		IsStream:         false,
		StartTime:        time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.UserSetting.AcceptUnsetRatioModel = false

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", "default")
	c.Set("token_name", "test_token")

	promptTokens := 1000
	completionTokens := 200

	_, err = helper.ModelPriceHelper(c, info, promptTokens, nil)
	require.NoError(t, err)

	// GroupRatio should fallback to group ratio (model not in enterprise sheet)
	assert.Equal(t, groupRatio, info.PriceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should fallback to group ratio when model not in enterprise sheet")
	assert.Equal(t, "group_ratio", info.PriceData.GroupRatioInfo.RatioSource,
		"RatioSource should be group_ratio after fallback")

	// SupplierCost should still be set (supplier pricing is independent of enterprise)
	assert.Equal(t, 0.6, info.PriceData.GroupRatioInfo.SupplierCost,
		"SupplierCost should be set even when enterprise sheet model not matched")
	assert.Equal(t, "ratio", info.PriceData.GroupRatioInfo.SupplierCostType)
	assert.Equal(t, supplierSheet.Id, info.PriceData.GroupRatioInfo.SupplierSheetId)

	usage := &dto.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 0,
		},
	}

	PostTextConsumeQuota(c, info, usage, nil)

	log := getLastLog(t)
	require.NotNil(t, log)

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)

	// Group ratio is used (no enterprise pricing for this model)
	assert.Equal(t, "group_ratio", other["ratio_source"])
	assert.Equal(t, groupRatio, other["discount_ratio"])

	// Supplier cost is still recorded
	assert.Equal(t, 0.6, other["supplier_cost"],
		"supplier_cost should be recorded even when enterprise sheet model not matched")
	assert.Equal(t, "ratio", other["supplier_cost_type"])
	assert.Equal(t, float64(supplierSheet.Id), other["supplier_sheet_id"])
}

// ============================================================================
// Test 5: Supplier stats SQL — verify aggregation from written logs
//   This exercises the same SQL path that powers the supplier analysis page.
// ============================================================================

func TestThreeLayerBilling_SupplierStatsAggregation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		modelName = "claude-opus-4-7-max"
	)

	err := ratio_setting.UpdateModelRatioByJSONString(`{"` + modelName + `": 1.0}`)
	require.NoError(t, err)
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1.0, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	now := time.Now().Unix()

	// -------------------------------------------------------------------------
	// Set up: two suppliers, two channels, one enterprise user
	// -------------------------------------------------------------------------
	enterprise := &model.Enterprise{
		Name:      "StatsCorp",
		Status:    model.EnterpriseStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(enterprise).Error)

	enterpriseSheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterprise.Id,
		Name:         "Stats报价单",
		Status:       model.PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, model.DB.Create(enterpriseSheet).Error)

	enterpriseItem := &model.EnterprisePricingItem{
		PricingSheetId: enterpriseSheet.Id,
		Models:        []string{modelName},
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.5, // customer pays 50%
	}
	require.NoError(t, model.DB.Create(enterpriseItem).Error)

	supplierA := &model.Supplier{Name: "SupplierA", Status: model.SupplierStatusEnabled, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, model.DB.Create(supplierA).Error)

	supplierSheetA := &model.SupplierPricingSheet{
		SupplierId: supplierA.Id,
		Name:       "SupplierA报价单",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, model.DB.Create(supplierSheetA).Error)

	supplierItemA := &model.SupplierPricingItem{
		PricingSheetId: supplierSheetA.Id,
		VendorType:     "anthropic",
		Models:         []string{modelName},
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.9, // cost = 90% of upstream price
	}
	require.NoError(t, model.DB.Create(supplierItemA).Error)

	supplierB := &model.Supplier{Name: "SupplierB", Status: model.SupplierStatusEnabled, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, model.DB.Create(supplierB).Error)

	supplierSheetB := &model.SupplierPricingSheet{
		SupplierId: supplierB.Id,
		Name:       "SupplierB报价单",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  now - 86400,
		EndTime:    now + 86400,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	require.NoError(t, model.DB.Create(supplierSheetB).Error)

	supplierItemB := &model.SupplierPricingItem{
		PricingSheetId: supplierSheetB.Id,
		VendorType:     "anthropic",
		Models:         []string{modelName},
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.6, // cost = 60% of upstream price
	}
	require.NoError(t, model.DB.Create(supplierItemB).Error)

	channelA := &model.Channel{Id: 7001, Name: "ChannelA", Key: "sk-A", Status: common.ChannelStatusEnabled}
	require.NoError(t, model.DB.Create(channelA).Error)

	channelB := &model.Channel{Id: 7002, Name: "ChannelB", Key: "sk-B", Status: common.ChannelStatusEnabled}
	require.NoError(t, model.DB.Create(channelB).Error)

	// Bind sheets to channels
	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	// Bind supplierA to channelA, supplierB to channelB
	require.NoError(t, model.DB.Create(&model.SupplierPricingSheetChannel{
		PricingSheetId: supplierSheetA.Id, ChannelId: channelA.Id, CreatedAt: now,
	}).Error)
	require.NoError(t, model.DB.Create(&model.SupplierPricingSheetChannel{
		PricingSheetId: supplierSheetB.Id, ChannelId: channelB.Id, CreatedAt: now,
	}).Error)

	// Bind enterprise sheet to both channels
	require.NoError(t, model.DB.Create(&model.EnterprisePricingSheetChannel{
		PricingSheetId: enterpriseSheet.Id, ChannelId: channelA.Id, CreatedAt: now,
	}).Error)
	require.NoError(t, model.DB.Create(&model.EnterprisePricingSheetChannel{
		PricingSheetId: enterpriseSheet.Id, ChannelId: channelB.Id, CreatedAt: now,
	}).Error)

	user := &model.User{Id: 7001, Username: "stats_user", Quota: 100000, Status: common.UserStatusEnabled, CreatedAt: now}
	require.NoError(t, model.DB.Create(user).Error)

	require.NoError(t, model.DB.Create(&model.EnterpriseUserBinding{
		EnterpriseId: enterprise.Id, UserId: user.Id, CreatedAt: now,
	}).Error)

	// -------------------------------------------------------------------------
	// Make 2 requests: channelA (supplierA cost=0.9) and channelB (supplierB cost=0.6)
	// Enterprise ratio = 0.5 applied to both.
	// -------------------------------------------------------------------------
	testCases := []struct {
		userID       int
		channelID    int
		promptTokens int
		compTokens   int
	}{
		{user.Id, channelA.Id, 1000, 200}, // via channelA, supplierA (cost=0.9)
		{user.Id, channelB.Id, 1000, 200}, // via channelB, supplierB (cost=0.6)
	}

	for _, tc := range testCases {
		info := &relaycommon.RelayInfo{
			UserId:           tc.userID,
			ChannelMeta:      &relaycommon.ChannelMeta{ChannelId: tc.channelID},
			UsingGroup:       "default",
			OriginModelName:  modelName,
			TokenId:          0,
			IsStream:         false,
			StartTime:        time.Now().Add(-2 * time.Second),
			FirstResponseTime: time.Now(),
		}
		info.UserSetting.AcceptUnsetRatioModel = false

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		c.Set("group", "default")
		c.Set("token_name", "test_token")

		_, err = helper.ModelPriceHelper(c, info, tc.promptTokens, nil)
		require.NoError(t, err)

		usage := &dto.Usage{
			PromptTokens:     tc.promptTokens,
			CompletionTokens: tc.compTokens,
			TotalTokens:      tc.promptTokens + tc.compTokens,
			PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 0},
		}

		PostTextConsumeQuota(c, info, usage, nil)
	}

	// -------------------------------------------------------------------------
	// Verify: total request count = 2
	// -------------------------------------------------------------------------
	totalLogs := countLogs(t)
	assert.Equal(t, int64(2), totalLogs, "should have 2 consume logs")

	// -------------------------------------------------------------------------
	// Verify: supplier stats aggregation
	// -------------------------------------------------------------------------
	filter := SupplierStatsFilter{
		StartTimestamp: now - 86400,
		EndTimestamp:  now + 86400,
	}

	// API 0: Overview
	overview, err := GetSupplierStatsOverview(filter)
	require.NoError(t, err)
	assert.Equal(t, int64(2), overview.RequestCount,
		"overview should show 2 requests")
	assert.Greater(t, overview.TotalCharge, float64(0),
		"total_charge should be greater than 0")
	assert.Greater(t, overview.TotalCost, float64(0),
		"total_cost should be greater than 0")
	assert.Greater(t, overview.GrossProfit, float64(0),
		"gross_profit should be greater than 0 (enterprise ratio 0.5 > supplier costs)")

	// supplier_count should be 2 (both suppliers used)
	assert.Equal(t, 2, overview.SupplierCount,
		"supplier_count should be 2 (both SupplierA and SupplierB used)")

	// channel_count should be 2 (both channels used)
	assert.Equal(t, 2, overview.ChannelCount,
		"channel_count should be 2 (both ChannelA and ChannelB used)")

	// -------------------------------------------------------------------------
	// API 1: By Supplier
	// -------------------------------------------------------------------------
	bySupplier, err := GetSupplierStatsBySupplier(filter)
	require.NoError(t, err)
	assert.Len(t, bySupplier.Items, 2,
		"should have 2 supplier rows (SupplierA and SupplierB)")

	// Find each supplier's stats
	supplierStats := make(map[string]SupplierStatsBySupplierItem)
	for _, item := range bySupplier.Items {
		supplierStats[item.SupplierName] = item
	}

	// SupplierA: cost=0.9, should have lower profit margin
	if itemA, ok := supplierStats["SupplierA"]; ok {
		assert.Greater(t, itemA.TotalCharge, float64(0))
		assert.Greater(t, itemA.TotalCost, float64(0))
		assert.Greater(t, itemA.GrossProfit, float64(0),
			"SupplierA gross_profit should be > 0 (enterprise ratio 0.5 > cost 0.9)")
	}

	// SupplierB: cost=0.6, should have higher profit margin than SupplierA
	if itemB, ok := supplierStats["SupplierB"]; ok {
		assert.Greater(t, itemB.TotalCharge, float64(0))
		assert.Greater(t, itemB.TotalCost, float64(0))
		assert.Greater(t, itemB.GrossProfit, float64(0),
			"SupplierB gross_profit should be > 0")
	}

	// -------------------------------------------------------------------------
	// API 2: By Channel
	// -------------------------------------------------------------------------
	byChannel, err := GetSupplierStatsByChannel(filter)
	require.NoError(t, err)
	assert.Len(t, byChannel.Items, 2,
		"should have 2 channel rows")

	// -------------------------------------------------------------------------
	// Verify: logs contain correct supplier_cost for each channel
	// -------------------------------------------------------------------------
	var allLogs []model.Log
	model.LOG_DB.Order("id asc").Find(&allLogs)
	assert.Len(t, allLogs, 2)

	// First log: channelA, supplierA cost=0.9
	var log1Other map[string]interface{}
	err = json.Unmarshal([]byte(allLogs[0].Other), &log1Other)
	require.NoError(t, err)
	assert.Equal(t, channelA.Id, allLogs[0].ChannelId)
	assert.Equal(t, 0.9, log1Other["supplier_cost"],
		"log1 (channelA) should have supplier_cost = 0.9")
	assert.Equal(t, float64(supplierSheetA.Id), log1Other["supplier_sheet_id"])

	// Second log: channelB, supplierB cost=0.6
	var log2Other map[string]interface{}
	err = json.Unmarshal([]byte(allLogs[1].Other), &log2Other)
	require.NoError(t, err)
	assert.Equal(t, channelB.Id, allLogs[1].ChannelId)
	assert.Equal(t, 0.6, log2Other["supplier_cost"],
		"log2 (channelB) should have supplier_cost = 0.6")
	assert.Equal(t, float64(supplierSheetB.Id), log2Other["supplier_sheet_id"])

	// Both logs should have enterprise pricing applied
	assert.Equal(t, "enterprise_pricing_sheet", log1Other["ratio_source"])
	assert.Equal(t, "enterprise_pricing_sheet", log2Other["ratio_source"])
	assert.Equal(t, 0.5, log1Other["discount_ratio"],
		"enterprise ratio 0.5 should be applied to channelA")
	assert.Equal(t, 0.5, log2Other["discount_ratio"],
		"enterprise ratio 0.5 should be applied to channelB")
}

// ============================================================================
// Test 6: No supplier pricing sheet — supplier_cost NOT written
//   Verifies graceful degradation when no supplier pricing is configured.
// ============================================================================

func TestThreeLayerBilling_NoSupplierPricing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID    = 8001
		testChannelID = 8001
		modelName     = "claude-opus-4-7-max"
	)

	err := ratio_setting.UpdateModelRatioByJSONString(`{"claude-opus-4-7-max": 1.0}`)
	require.NoError(t, err)

	now := time.Now().Unix()

	user := &model.User{
		Id:       testUserID,
		Username:  "user_no_supplier",
		Quota:     100000,
		Status:    common.UserStatusEnabled,
		CreatedAt: now,
	}
	require.NoError(t, model.DB.Create(user).Error)

	channel := &model.Channel{
		Id:     testChannelID,
		Name:   "test_channel_no_supplier",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, model.DB.Create(channel).Error)

	info := &relaycommon.RelayInfo{
		UserId:           testUserID,
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelId: testChannelID},
		UsingGroup:       "default",
		OriginModelName:  modelName,
		TokenId:          0,
		IsStream:         false,
		StartTime:        time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.UserSetting.AcceptUnsetRatioModel = false

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", "default")
	c.Set("token_name", "test_token")

	promptTokens := 1000
	completionTokens := 200

	_, err = helper.ModelPriceHelper(c, info, promptTokens, nil)
	require.NoError(t, err)

	// SupplierCost should remain 0 (no supplier pricing configured)
	assert.Equal(t, float64(0), info.PriceData.GroupRatioInfo.SupplierCost,
		"SupplierCost should be 0 when no supplier pricing is configured")
	assert.Equal(t, "", info.PriceData.GroupRatioInfo.SupplierCostType)
	assert.Equal(t, 0, info.PriceData.GroupRatioInfo.SupplierSheetId)

	usage := &dto.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 0,
		},
	}

	PostTextConsumeQuota(c, info, usage, nil)

	log := getLastLog(t)
	require.NotNil(t, log)

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)

	// supplier_cost should NOT be in the log (SupplierCost was 0)
	_, hasSupplierCost := other["supplier_cost"]
	assert.False(t, hasSupplierCost,
		"supplier_cost should NOT be in the log when no supplier pricing is configured")
	_, hasSupplierCostType := other["supplier_cost_type"]
	assert.False(t, hasSupplierCostType,
		"supplier_cost_type should NOT be in the log")
	_, hasSupplierSheetId := other["supplier_sheet_id"]
	assert.False(t, hasSupplierSheetId,
		"supplier_sheet_id should NOT be in the log")
}
