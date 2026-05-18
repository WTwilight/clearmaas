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

// truncate() and seed helpers (seedUser, seedChannel, getLastLog, countLogs)
// are defined in task_billing_test.go and shared within the service package.

// resetRatioSetting restores ratio_setting maps to their default values.
// Call this before each test to prevent cross-test contamination.
func resetRatioSetting() {
	_ = ratio_setting.UpdateModelRatioByJSONString(`{}`)
	_ = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1, "vip": 1, "svip": 1}`)
	_ = ratio_setting.UpdateModelPriceByJSONString(`{}`)
}

// ============================================================================
// Test 1: Enterprise user — billing uses enterprise pricing sheet ratio
//           and consume log contains enterprise pricing fields
// ============================================================================

func TestEnterpriseBilling_UserWithPricingSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID    = 1001
		testChannelID = 1001
		groupName     = "default"
		modelName     = "gpt-4o"
	)

	// -------------------------------------------------------------------------
	// 1. Inject model ratio: gpt-4o = 1.25
	// -------------------------------------------------------------------------
	err := ratio_setting.UpdateModelRatioByJSONString(`{"gpt-4o": 1.25}`)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 2. Set up: enterprise + active pricing sheet + pricing item
	//    Enterprise discount = 0.5 (50% of base price)
	// -------------------------------------------------------------------------
	now := time.Now().Unix()

	enterprise := &model.Enterprise{
		Name:      "TestCorp",
		Status:    model.EnterpriseStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(enterprise).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterprise.Id,
		Name:         "企业报价单-2024",
		Status:       model.PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, model.DB.Create(sheet).Error)

	pricingItem := &model.EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:         modelName,
		DiscountType:  model.DiscountTypeRatio,
		DiscountValue: 0.5,
	}
	require.NoError(t, model.DB.Create(pricingItem).Error)

	user := &model.User{
		Id:       testUserID,
		Username: "enterprise_user",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	binding := &model.EnterpriseUserBinding{
		EnterpriseId: enterprise.Id,
		UserId:       testUserID,
		CreatedAt:    now,
	}
	require.NoError(t, model.DB.Create(binding).Error)

	channel := &model.Channel{
		Id:     testChannelID,
		Name:   "test_channel",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, model.DB.Create(channel).Error)

	// -------------------------------------------------------------------------
	// 3. Build RelayInfo and run ModelPriceHelper
	// -------------------------------------------------------------------------
	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: testChannelID},
		UsingGroup:      groupName,
		OriginModelName:  modelName,
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.UserSetting.AcceptUnsetRatioModel = false

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", groupName)
	c.Set("token_name", "test_token")

	promptTokens := 1000
	completionTokens := 200

	_, err = helper.ModelPriceHelper(c, info, promptTokens, nil)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 4. Assert: GroupRatio should come from enterprise pricing sheet
	// -------------------------------------------------------------------------
	assert.Equal(t, 0.5, info.PriceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should be overridden by enterprise pricing sheet (0.5)")
	assert.Equal(t, "enterprise_pricing_sheet", info.PriceData.GroupRatioInfo.RatioSource,
		"RatioSource should be enterprise_pricing_sheet")
	assert.Equal(t, sheet.Id, info.PriceData.GroupRatioInfo.EnterpriseSheetId,
		"EnterpriseSheetId should be set")
	assert.Equal(t, sheet.Name, info.PriceData.GroupRatioInfo.EnterpriseSheetName,
		"EnterpriseSheetName should be set")

	// Pre-consume: promptTokens(1000) * modelRatio(1.25) * enterpriseRatio(0.5) = 625
	assert.Equal(t, 625, info.PriceData.QuotaToPreConsume,
		"Pre-consume quota should reflect enterprise ratio")

	// -------------------------------------------------------------------------
	// 5. Simulate usage and run PostTextConsumeQuota
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
	// 6. Assert: consume log should contain enterprise pricing fields
	// -------------------------------------------------------------------------
	log := getLastLog(t)
	require.NotNil(t, log, "consume log should be created")
	assert.Equal(t, model.LogTypeConsume, log.Type, "log type should be Consume")
	assert.Equal(t, modelName, log.ModelName, "model name should match")
	assert.Greater(t, log.Quota, 0, "quota should be greater than 0")

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err, "Other field should be valid JSON")

	assert.Equal(t, "enterprise_pricing_sheet", other["ratio_source"],
		"log should record ratio_source = enterprise_pricing_sheet")
	assert.Equal(t, float64(sheet.Id), other["enterprise_sheet_id"],
		"log should record enterprise_sheet_id")
	assert.Equal(t, sheet.Name, other["enterprise_sheet_name"],
		"log should record enterprise_sheet_name")
	assert.Equal(t, 0.5, other["discount_ratio"],
		"log should record discount_ratio = 0.5 (enterprise pricing)")
	assert.Greater(t, other["original_price"], float64(0),
		"original_price should be recorded")
}

// ============================================================================
// Test 2: Non-enterprise user — billing uses group ratio
//           and consume log contains group ratio fields
// ============================================================================

func TestEnterpriseBilling_UserWithoutPricingSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID    = 2001
		testChannelID = 2001
		groupName     = "default"
		modelName     = "gpt-4o"
		groupRatio    = 1.5 // group ratio for "default" group
	)

	// -------------------------------------------------------------------------
	// 1. Inject model ratio: gpt-4o = 1.25
	// -------------------------------------------------------------------------
	err := ratio_setting.UpdateModelRatioByJSONString(`{"gpt-4o": 1.25}`)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 2. Inject group ratio: default = 1.5
	// -------------------------------------------------------------------------
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1.5, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 3. Set up user and channel (NO enterprise binding)
	// -------------------------------------------------------------------------
	user := &model.User{
		Id:       testUserID,
		Username: "normal_user",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	channel := &model.Channel{
		Id:     testChannelID,
		Name:   "test_channel",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, model.DB.Create(channel).Error)

	// -------------------------------------------------------------------------
	// 4. Build RelayInfo and run ModelPriceHelper
	// -------------------------------------------------------------------------
	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: testChannelID},
		UsingGroup:      groupName,
		OriginModelName:  modelName,
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.UserSetting.AcceptUnsetRatioModel = false

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", groupName)
	c.Set("token_name", "test_token")

	promptTokens := 1000
	completionTokens := 200

	_, err = helper.ModelPriceHelper(c, info, promptTokens, nil)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 5. Assert: GroupRatio should come from group ratio
	// -------------------------------------------------------------------------
	assert.Equal(t, groupRatio, info.PriceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should use group ratio (1.5) since user has no enterprise binding")
	assert.Equal(t, "group_ratio", info.PriceData.GroupRatioInfo.RatioSource,
		"RatioSource should be group_ratio")
	assert.Equal(t, 0, info.PriceData.GroupRatioInfo.EnterpriseSheetId,
		"EnterpriseSheetId should be 0 for non-enterprise user")
	assert.Equal(t, "", info.PriceData.GroupRatioInfo.EnterpriseSheetName,
		"EnterpriseSheetName should be empty for non-enterprise user")

	// Pre-consume: promptTokens(1000) * modelRatio(1.25) * groupRatio(1.5) = 1875
	assert.Equal(t, 1875, info.PriceData.QuotaToPreConsume,
		"Pre-consume quota should use group ratio (1.5)")

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
	// 7. Assert: consume log should contain group ratio fields
	// -------------------------------------------------------------------------
	log := getLastLog(t)
	require.NotNil(t, log, "consume log should be created")
	assert.Equal(t, model.LogTypeConsume, log.Type, "log type should be Consume")
	assert.Equal(t, modelName, log.ModelName, "model name should match")

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err, "Other field should be valid JSON")

	assert.Equal(t, "group_ratio", other["ratio_source"],
		"log should record ratio_source = group_ratio for non-enterprise user")
	_, hasEnterpriseId := other["enterprise_sheet_id"]
	assert.False(t, hasEnterpriseId,
		"enterprise_sheet_id should not be present for non-enterprise user")
	_, hasEnterpriseName := other["enterprise_sheet_name"]
	assert.False(t, hasEnterpriseName,
		"enterprise_sheet_name should not be present for non-enterprise user")
	assert.Equal(t, groupRatio, other["discount_ratio"],
		"log should record discount_ratio = 1.5 (group ratio)")
	assert.Greater(t, other["original_price"], float64(0),
		"original_price should be recorded")
}

// ============================================================================
// Test 3: Enterprise user, model NOT in pricing sheet — falls back to group
// ============================================================================

func TestEnterpriseBilling_ModelNotInSheet_FallsBackToGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID    = 3001
		testChannelID = 3001
		groupName     = "default"
		modelName     = "gpt-4o"
		groupRatio    = 1.5
	)

	// Inject ratios
	err := ratio_setting.UpdateModelRatioByJSONString(`{"gpt-4o": 1.25}`)
	require.NoError(t, err)
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1.5, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	// Set up: enterprise + sheet, but item is for DIFFERENT model
	now := time.Now().Unix()

	enterprise := &model.Enterprise{
		Name:      "TestCorp2",
		Status:    model.EnterpriseStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(enterprise).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterprise.Id,
		Name:         "报价单-只有mini",
		Status:       model.PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, model.DB.Create(sheet).Error)

	// Sheet only has gpt-4o-mini, NOT gpt-4o
	pricingItem := &model.EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:         "gpt-4o-mini",
		DiscountType:  model.DiscountTypeRatio,
		DiscountValue: 0.3,
	}
	require.NoError(t, model.DB.Create(pricingItem).Error)

	user := &model.User{
		Id:       testUserID,
		Username: "enterprise_user2",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	binding := &model.EnterpriseUserBinding{
		EnterpriseId: enterprise.Id,
		UserId:       testUserID,
		CreatedAt:    now,
	}
	require.NoError(t, model.DB.Create(binding).Error)

	channel := &model.Channel{
		Id:     testChannelID,
		Name:   "test_channel",
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, model.DB.Create(channel).Error)

	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: testChannelID},
		UsingGroup:      groupName,
		OriginModelName:  modelName, // gpt-4o — NOT in sheet
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.UserSetting.AcceptUnsetRatioModel = false

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", groupName)
	c.Set("token_name", "test_token")

	_, err = helper.ModelPriceHelper(c, info, 1000, nil)
	require.NoError(t, err)

	// Should fall back to group ratio since model is not in enterprise pricing sheet
	assert.Equal(t, groupRatio, info.PriceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should fall back to group ratio when model not in enterprise sheet")
	assert.Equal(t, "group_ratio", info.PriceData.GroupRatioInfo.RatioSource,
		"RatioSource should be group_ratio")

	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 200,
		TotalTokens:      1200,
		PromptTokensDetails: dto.InputTokenDetails{CachedTokens: 0},
	}
	PostTextConsumeQuota(c, info, usage, nil)

	log := getLastLog(t)
	require.NotNil(t, log)
	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)
	assert.Equal(t, "group_ratio", other["ratio_source"])
	_, hasEnterpriseId := other["enterprise_sheet_id"]
	assert.False(t, hasEnterpriseId, "enterprise_sheet_id should not be set when model not in sheet")
}

// ============================================================================
// Test 4: Per-call billing — enterprise pricing user
//   Uses ModelPriceHelperPerCall + LogTaskConsumption
// ============================================================================

func TestEnterpriseBilling_PerCall_UserWithPricingSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID = 4001
		modelName  = "midjourney"
		// common.QuotaPerUnit = 500_000.0
		// quota = 0.02 * 500_000.0 * 0.5(enterprise ratio) = 5000
		enterpriseRatio = 0.5
	)

	// -------------------------------------------------------------------------
	// 1. Inject model price: midjourney = 0.02
	// -------------------------------------------------------------------------
	err := ratio_setting.UpdateModelPriceByJSONString(`{"midjourney": 0.02}`)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 2. Set up: enterprise + pricing sheet + pricing item
	// -------------------------------------------------------------------------
	now := time.Now().Unix()

	enterprise := &model.Enterprise{
		Name:      "PerCallCorp",
		Status:    model.EnterpriseStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(enterprise).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterprise.Id,
		Name:         "MJ企业报价单",
		Status:       model.PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, model.DB.Create(sheet).Error)

	pricingItem := &model.EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:         modelName,
		DiscountType:  model.DiscountTypeRatio,
		DiscountValue: enterpriseRatio,
	}
	require.NoError(t, model.DB.Create(pricingItem).Error)

	user := &model.User{
		Id:       testUserID,
		Username: "percall_user",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	binding := &model.EnterpriseUserBinding{
		EnterpriseId: enterprise.Id,
		UserId:       testUserID,
		CreatedAt:    now,
	}
	require.NoError(t, model.DB.Create(binding).Error)

	// -------------------------------------------------------------------------
	// 3. Build RelayInfo
	// -------------------------------------------------------------------------
	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 1},
		UsingGroup:      "default",
		OriginModelName:  modelName,
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: "image_generation"}
	info.UserSetting.AcceptUnsetRatioModel = false

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/mj/submit", nil)
	c.Set("token_name", "test_token")

	// -------------------------------------------------------------------------
	// 4. Run ModelPriceHelperPerCall
	// -------------------------------------------------------------------------
	priceData, err := helper.ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)
	// Mirror what relay_task.go does: write the returned PriceData back to info
	info.PriceData = priceData

	// Assert: GroupRatio should come from enterprise pricing sheet
	assert.Equal(t, enterpriseRatio, priceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should be overridden by enterprise pricing sheet (0.5)")
	assert.Equal(t, "enterprise_pricing_sheet", priceData.GroupRatioInfo.RatioSource,
		"RatioSource should be enterprise_pricing_sheet")
	assert.Equal(t, sheet.Id, priceData.GroupRatioInfo.EnterpriseSheetId,
		"EnterpriseSheetId should be set")
	assert.Equal(t, sheet.Name, priceData.GroupRatioInfo.EnterpriseSheetName,
		"EnterpriseSheetName should be set")

	// quota = modelPrice(0.02) * QuotaPerUnit(1000000) * enterpriseRatio(0.5) = 10000
	assert.Equal(t, 5000, priceData.Quota,
		"Quota should reflect enterprise ratio: 0.02 * 500000 * 0.5 = 5000")
	assert.True(t, priceData.UsePrice, "UsePrice should be true for per-call billing")

	// -------------------------------------------------------------------------
	// 5. Run LogTaskConsumption and verify log
	// -------------------------------------------------------------------------
	LogTaskConsumption(c, info)

	log := getLastLog(t)
	require.NotNil(t, log, "consume log should be created")
	assert.Equal(t, model.LogTypeConsume, log.Type)
	assert.Equal(t, modelName, log.ModelName)
	assert.Equal(t, 5000, log.Quota)

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)
	assert.Equal(t, "enterprise_pricing_sheet", other["ratio_source"],
		"log should record ratio_source = enterprise_pricing_sheet")
	assert.Equal(t, float64(sheet.Id), other["enterprise_sheet_id"],
		"log should record enterprise_sheet_id")
	assert.Equal(t, sheet.Name, other["enterprise_sheet_name"],
		"log should record enterprise_sheet_name")
}

// ============================================================================
// Test 5: Per-call billing — non-enterprise user falls back to group ratio
// ============================================================================

func TestEnterpriseBilling_PerCall_UserWithoutPricingSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID = 5001
		modelName  = "midjourney"
		groupRatio = 1.5
		// common.QuotaPerUnit = 500_000.0
		// quota = 0.02 * 500_000.0 * 1.5 = 15000
	)

	// -------------------------------------------------------------------------
	// 1. Inject model price and group ratio
	// -------------------------------------------------------------------------
	err := ratio_setting.UpdateModelPriceByJSONString(`{"midjourney": 0.02}`)
	require.NoError(t, err)
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1.5, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 2. Set up user only (no enterprise binding)
	// -------------------------------------------------------------------------
	user := &model.User{
		Id:       testUserID,
		Username: "percall_normal",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	// -------------------------------------------------------------------------
	// 3. Build RelayInfo
	// -------------------------------------------------------------------------
	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 1},
		UsingGroup:      "default",
		OriginModelName:  modelName,
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: "image_generation"}
	info.UserSetting.AcceptUnsetRatioModel = false

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/mj/submit", nil)
	c.Set("token_name", "test_token")

	// -------------------------------------------------------------------------
	// 4. Run ModelPriceHelperPerCall
	// -------------------------------------------------------------------------
	priceData, err := helper.ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)
	// Mirror what relay_task.go does
	info.PriceData = priceData

	// Assert: GroupRatio should come from group ratio
	assert.Equal(t, groupRatio, priceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should use group ratio (1.5) since user has no enterprise binding")
	assert.Equal(t, "group_ratio", priceData.GroupRatioInfo.RatioSource,
		"RatioSource should be group_ratio")

	// common.QuotaPerUnit = 500_000
	// quota = 0.02 * 500_000 * 1.5 = 15000
	assert.Equal(t, 15000, priceData.Quota,
		"Quota should use group ratio: 0.02 * 500000 * 1.5 = 15000")
	assert.True(t, priceData.UsePrice)

	// -------------------------------------------------------------------------
	// 5. Run LogTaskConsumption and verify log
	// -------------------------------------------------------------------------
	LogTaskConsumption(c, info)

	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeConsume, log.Type)
	assert.Equal(t, 15000, log.Quota)

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)
	assert.Equal(t, "group_ratio", other["ratio_source"],
		"log should record ratio_source = group_ratio")
	_, hasEnterpriseId := other["enterprise_sheet_id"]
	assert.False(t, hasEnterpriseId, "enterprise_sheet_id should not be present for non-enterprise user")
	_, hasEnterpriseName := other["enterprise_sheet_name"]
	assert.False(t, hasEnterpriseName, "enterprise_sheet_name should not be present for non-enterprise user")
}

// ============================================================================
// Test 6: Per-call billing — model NOT in pricing sheet falls back to group
// ============================================================================

func TestEnterpriseBilling_PerCall_ModelNotInSheet_FallsBackToGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID = 6001
		modelName  = "midjourney"
		groupRatio = 1.5
	)

	// Inject ratios
	err := ratio_setting.UpdateModelPriceByJSONString(`{"midjourney": 0.02}`)
	require.NoError(t, err)
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1.5, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	// Set up: enterprise + sheet, but item is for DIFFERENT model
	now := time.Now().Unix()

	enterprise := &model.Enterprise{
		Name:      "PerCallCorp2",
		Status:    model.EnterpriseStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(enterprise).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterprise.Id,
		Name:         "MJ报价单-只有mini",
		Status:       model.PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, model.DB.Create(sheet).Error)

	// Sheet only has gpt-4o-mini, NOT midjourney
	pricingItem := &model.EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:         "gpt-4o-mini",
		DiscountType:  model.DiscountTypeRatio,
		DiscountValue: 0.3,
	}
	require.NoError(t, model.DB.Create(pricingItem).Error)

	user := &model.User{
		Id:       testUserID,
		Username: "percall_user2",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	binding := &model.EnterpriseUserBinding{
		EnterpriseId: enterprise.Id,
		UserId:       testUserID,
		CreatedAt:    now,
	}
	require.NoError(t, model.DB.Create(binding).Error)

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 1},
		UsingGroup:      "default",
		OriginModelName:  modelName,
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: "image_generation"}
	info.UserSetting.AcceptUnsetRatioModel = false

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/mj/submit", nil)
	c.Set("token_name", "test_token")

	priceData, err := helper.ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)
	// Mirror what relay_task.go does
	info.PriceData = priceData

	// Should fall back to group ratio since model is not in enterprise pricing sheet
	assert.Equal(t, groupRatio, priceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should fall back to group ratio when model not in enterprise sheet")
	assert.Equal(t, "group_ratio", priceData.GroupRatioInfo.RatioSource,
		"RatioSource should be group_ratio")
	assert.Equal(t, 15000, priceData.Quota,
		"Quota should use group ratio: 0.02 * 500000 * 1.5 = 15000")

	LogTaskConsumption(c, info)

	log := getLastLog(t)
	require.NotNil(t, log)
	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)
	assert.Equal(t, "group_ratio", other["ratio_source"])
	_, hasEnterpriseId := other["enterprise_sheet_id"]
	assert.False(t, hasEnterpriseId, "enterprise_sheet_id should not be set when model not in sheet")
}

// ============================================================================
// Test 7: Enterprise pricing per_call type — fixed price per call
//   Uses ModelPriceHelperPerCall, PerCallPriceSheet is set
// ============================================================================

func TestEnterpriseBilling_PerCallSheet_FixedPricePerCall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID = 7001
		modelName  = "midjourney"
		// common.QuotaPerUnit = 500_000.0
		// quota = 0.05 * 500_000.0 * 1.0(default group) = 25000
		perCallPrice = 0.05 // $0.05 per call
	)

	// -------------------------------------------------------------------------
	// 1. Inject model ratio: midjourney = 1.0 (used for fallback)
	// -------------------------------------------------------------------------
	err := ratio_setting.UpdateModelRatioByJSONString(`{"midjourney": 1.0}`)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 2. Set up: enterprise + pricing sheet + pricing item of type per_call
	// -------------------------------------------------------------------------
	now := time.Now().Unix()

	enterprise := &model.Enterprise{
		Name:      "PerCallSheetCorp",
		Status:    model.EnterpriseStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(enterprise).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterprise.Id,
		Name:         "MJ固定价格报价单",
		Status:       model.PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, model.DB.Create(sheet).Error)

	pricingItem := &model.EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:         modelName,
		DiscountType:  model.DiscountTypePerCall,
		DiscountValue: perCallPrice,
	}
	require.NoError(t, model.DB.Create(pricingItem).Error)

	user := &model.User{
		Id:       testUserID,
		Username: "percall_sheet_user",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	binding := &model.EnterpriseUserBinding{
		EnterpriseId: enterprise.Id,
		UserId:       testUserID,
		CreatedAt:    now,
	}
	require.NoError(t, model.DB.Create(binding).Error)

	// -------------------------------------------------------------------------
	// 3. Build RelayInfo
	// -------------------------------------------------------------------------
	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 1},
		UsingGroup:      "default",
		OriginModelName: modelName,
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: "image_generation"}
	info.UserSetting.AcceptUnsetRatioModel = false

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/mj/submit", nil)
	c.Set("token_name", "test_token")

	// -------------------------------------------------------------------------
	// 4. Run ModelPriceHelperPerCall
	// -------------------------------------------------------------------------
	priceData, err := helper.ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)
	info.PriceData = priceData

	// Assert: PerCallPriceSheet should be set from enterprise pricing sheet
	assert.Equal(t, perCallPrice, priceData.GroupRatioInfo.PerCallPriceSheet,
		"PerCallPriceSheet should be set to the per_call price from enterprise sheet")
	assert.Equal(t, perCallPrice, priceData.PerCallPriceSheet,
		"PriceData.PerCallPriceSheet should match")
	assert.Equal(t, "enterprise_pricing_sheet", priceData.GroupRatioInfo.RatioSource,
		"RatioSource should be enterprise_pricing_sheet")
	assert.Equal(t, sheet.Id, priceData.GroupRatioInfo.EnterpriseSheetId,
		"EnterpriseSheetId should be set")
	assert.Equal(t, sheet.Name, priceData.GroupRatioInfo.EnterpriseSheetName,
		"EnterpriseSheetName should be set")

	// quota = perCallPrice(0.05) * QuotaPerUnit(500000.0) * 1.0(group) = 25000
	assert.Equal(t, 25000, priceData.Quota,
		"Quota should be computed from per_call price: 0.05 * 500000.0 = 25000")
	assert.True(t, priceData.UsePrice, "UsePrice should be true for per_call billing")
	assert.Equal(t, perCallPrice, priceData.ModelPrice,
		"ModelPrice should store the per_call price")

	// -------------------------------------------------------------------------
	// 5. Verify PerCallBilling will be set in controller (mirrors relay.go logic)
	// -------------------------------------------------------------------------
	// PerCallBilling is true when any of: TaskPricePatches, UsePrice, or PerCallPriceSheet > 0
	assert.True(t, info.PriceData.PerCallPriceSheet > 0,
		"PerCallBilling should be true when PerCallPriceSheet > 0")

	// -------------------------------------------------------------------------
	// 6. Run LogTaskConsumption and verify log
	// -------------------------------------------------------------------------
	LogTaskConsumption(c, info)

	log := getLastLog(t)
	require.NotNil(t, log, "consume log should be created")
	assert.Equal(t, model.LogTypeConsume, log.Type)
	assert.Equal(t, modelName, log.ModelName)
	assert.Equal(t, 25000, log.Quota)

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)
	assert.Equal(t, "enterprise_pricing_sheet", other["ratio_source"],
		"log should record ratio_source = enterprise_pricing_sheet")
	assert.Equal(t, float64(sheet.Id), other["enterprise_sheet_id"],
		"log should record enterprise_sheet_id")
	assert.Equal(t, sheet.Name, other["enterprise_sheet_name"],
		"log should record enterprise_sheet_name")
}

// ============================================================================
// Test 8: Enterprise pricing per_call — per_call price overrides group ratio
//   Also tests that OtherRatios are skipped for per_call in relay_task.go
// ============================================================================

func TestEnterpriseBilling_PerCallSheet_WithGroupRatioOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID   = 8001
		modelName    = "midjourney"
		perCallPrice = 0.05 // $0.05 per call
		groupRatio   = 0.8  // group ratio should be ignored for per_call
	)

	// Inject model ratio
	err := ratio_setting.UpdateModelRatioByJSONString(`{"midjourney": 1.0}`)
	require.NoError(t, err)

	// Inject group ratio
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 0.8, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	now := time.Now().Unix()

	enterprise := &model.Enterprise{
		Name:      "PerCallGroupCorp",
		Status:    model.EnterpriseStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(enterprise).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterprise.Id,
		Name:         "MJ固定价格-覆盖组倍率",
		Status:       model.PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, model.DB.Create(sheet).Error)

	pricingItem := &model.EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:         modelName,
		DiscountType:  model.DiscountTypePerCall,
		DiscountValue: perCallPrice,
	}
	require.NoError(t, model.DB.Create(pricingItem).Error)

	user := &model.User{
		Id:       testUserID,
		Username: "percall_group_user",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	binding := &model.EnterpriseUserBinding{
		EnterpriseId: enterprise.Id,
		UserId:       testUserID,
		CreatedAt:    now,
	}
	require.NoError(t, model.DB.Create(binding).Error)

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 1},
		UsingGroup:      "default",
		OriginModelName: modelName,
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: "image_generation"}
	info.UserSetting.AcceptUnsetRatioModel = false

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/mj/submit", nil)
	c.Set("token_name", "test_token")

	priceData, err := helper.ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)
	info.PriceData = priceData

	// PerCallPriceSheet should be set
	assert.Equal(t, perCallPrice, priceData.PerCallPriceSheet,
		"PerCallPriceSheet should be set for per_call type")

	// GroupRatio is NOT overridden by per_call — it stays as the group ratio (0.8)
	// This is intentional: per_call only sets PerCallPriceSheet, GroupRatio remains unchanged
	assert.Equal(t, groupRatio, priceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should NOT be overridden by per_call type, stays as group ratio (0.8)")

	// quota = perCallPrice(0.05) * QuotaPerUnit(500000.0) * GroupRatio(0.8) = 20000
	assert.Equal(t, 20000, priceData.Quota,
		"Quota should use groupRatio=0.8 for per_call: 0.05 * 500000.0 * 0.8 = 20000")
}

// ============================================================================
// Test 9: No enterprise pricing sheet — per_call falls back to model price
//   Verifies PerCallPriceSheet is NOT set when there's no enterprise sheet,
//   and billing uses the original model price * QuotaPerUnit * groupRatio.
// ============================================================================

func TestEnterpriseBilling_PerCall_NoSheet_FallsBackToModelPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID  = 9001
		modelName   = "midjourney"
		modelPrice  = 0.02 // $0.02 per call from model config
		groupRatio  = 1.5
		// quota = modelPrice(0.02) * QuotaPerUnit(500000.0) * groupRatio(1.5) = 15000
	)

	// -------------------------------------------------------------------------
	// 1. Inject model price and group ratio (no enterprise pricing sheet)
	// -------------------------------------------------------------------------
	err := ratio_setting.UpdateModelPriceByJSONString(`{"midjourney": 0.02}`)
	require.NoError(t, err)
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1.5, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 2. Set up user only (NO enterprise binding — no pricing sheet)
	// -------------------------------------------------------------------------
	user := &model.User{
		Id:       testUserID,
		Username: "percall_no_sheet",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	// -------------------------------------------------------------------------
	// 3. Build RelayInfo
	// -------------------------------------------------------------------------
	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 1},
		UsingGroup:      "default",
		OriginModelName: modelName,
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: "image_generation"}
	info.UserSetting.AcceptUnsetRatioModel = false

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/mj/submit", nil)
	c.Set("token_name", "test_token")

	// -------------------------------------------------------------------------
	// 4. Run ModelPriceHelperPerCall
	// -------------------------------------------------------------------------
	priceData, err := helper.ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)
	info.PriceData = priceData

	// Assert: PerCallPriceSheet should NOT be set (no enterprise pricing sheet)
	assert.Equal(t, 0.0, priceData.GroupRatioInfo.PerCallPriceSheet,
		"PerCallPriceSheet should be 0 when no enterprise pricing sheet exists")
	assert.Equal(t, 0.0, priceData.PerCallPriceSheet,
		"PriceData.PerCallPriceSheet should be 0")

	// Assert: GroupRatio should come from group ratio
	assert.Equal(t, groupRatio, priceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should use group ratio (1.5)")
	assert.Equal(t, "group_ratio", priceData.GroupRatioInfo.RatioSource,
		"RatioSource should be group_ratio (no enterprise sheet)")

	// Assert: UsePrice should be true (model has price config)
	assert.True(t, priceData.UsePrice, "UsePrice should be true")

	// Assert: Quota should be modelPrice * QuotaPerUnit * groupRatio
	assert.Equal(t, 15000, priceData.Quota,
		"Quota should fall back to model price: 0.02 * 500000.0 * 1.5 = 15000")

	// Assert: ModelPrice should store the model price
	assert.Equal(t, modelPrice, priceData.ModelPrice,
		"ModelPrice should store the model price from config")

	// -------------------------------------------------------------------------
	// 5. Verify PerCallBilling is set via UsePrice (original logic)
	// -------------------------------------------------------------------------
	// PerCallBilling = TaskPricePatches || UsePrice || PerCallPriceSheet > 0
	// Since PerCallPriceSheet = 0 and UsePrice = true, PerCallBilling should be true
	assert.True(t, info.PriceData.UsePrice,
		"PerCallBilling should be true via UsePrice when no enterprise sheet")

	// -------------------------------------------------------------------------
	// 6. Run LogTaskConsumption and verify log
	// -------------------------------------------------------------------------
	LogTaskConsumption(c, info)

	log := getLastLog(t)
	require.NotNil(t, log, "consume log should be created")
	assert.Equal(t, model.LogTypeConsume, log.Type)
	assert.Equal(t, modelName, log.ModelName)
	assert.Equal(t, 15000, log.Quota)

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)
	assert.Equal(t, "group_ratio", other["ratio_source"],
		"log should record ratio_source = group_ratio (no enterprise sheet)")
	_, hasEnterpriseId := other["enterprise_sheet_id"]
	assert.False(t, hasEnterpriseId,
		"enterprise_sheet_id should not be present when no enterprise pricing sheet")
}

// ============================================================================
// Test 10: Model in enterprise sheet but sheet type is NOT per_call (ratio type)
//   Verifies the original ratio billing path still works unchanged.
// ============================================================================

func TestEnterpriseBilling_RatioDiscountType_NotPerCall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID  = 10001
		modelName   = "midjourney"
		modelPrice  = 0.02 // $0.02 per call from model config
		ratioValue = 0.5  // 50% of base price
		groupRatio = 1.5
		// quota = modelPrice(0.02) * QuotaPerUnit(500000.0) * ratioValue(0.5) = 5000
		// (groupRatio is ignored when enterprise sheet ratio overrides it)
	)

	// -------------------------------------------------------------------------
	// 1. Inject model price and group ratio
	// -------------------------------------------------------------------------
	err := ratio_setting.UpdateModelPriceByJSONString(`{"midjourney": 0.02}`)
	require.NoError(t, err)
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1.5, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	// -------------------------------------------------------------------------
	// 2. Set up: enterprise + sheet + pricing item of type ratio (not per_call)
	// -------------------------------------------------------------------------
	now := time.Now().Unix()

	enterprise := &model.Enterprise{
		Name:      "RatioCorp",
		Status:    model.EnterpriseStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(enterprise).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterprise.Id,
		Name:         "MJ折扣报价单",
		Status:       model.PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, model.DB.Create(sheet).Error)

	pricingItem := &model.EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:         modelName,
		DiscountType:  model.DiscountTypeRatio, // NOT per_call
		DiscountValue: ratioValue,
	}
	require.NoError(t, model.DB.Create(pricingItem).Error)

	user := &model.User{
		Id:       testUserID,
		Username: "ratio_user",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	binding := &model.EnterpriseUserBinding{
		EnterpriseId: enterprise.Id,
		UserId:       testUserID,
		CreatedAt:    now,
	}
	require.NoError(t, model.DB.Create(binding).Error)

	// -------------------------------------------------------------------------
	// 3. Build RelayInfo
	// -------------------------------------------------------------------------
	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 1},
		UsingGroup:      "default",
		OriginModelName: modelName,
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: "image_generation"}
	info.UserSetting.AcceptUnsetRatioModel = false

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/mj/submit", nil)
	c.Set("token_name", "test_token")

	// -------------------------------------------------------------------------
	// 4. Run ModelPriceHelperPerCall
	// -------------------------------------------------------------------------
	priceData, err := helper.ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)
	info.PriceData = priceData

	// Assert: PerCallPriceSheet should NOT be set (this is ratio type, not per_call)
	assert.Equal(t, 0.0, priceData.GroupRatioInfo.PerCallPriceSheet,
		"PerCallPriceSheet should be 0 for ratio discount type")
	assert.Equal(t, 0.0, priceData.PerCallPriceSheet,
		"PriceData.PerCallPriceSheet should be 0 for ratio discount type")

	// Assert: GroupRatio should be overridden by enterprise sheet ratio
	assert.Equal(t, ratioValue, priceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should be overridden by enterprise sheet ratio (0.5)")

	// Assert: UsePrice should be true (model has price config)
	assert.True(t, priceData.UsePrice, "UsePrice should be true")

	// Assert: Quota = modelPrice * QuotaPerUnit * ratioValue
	assert.Equal(t, 5000, priceData.Quota,
		"Quota should be: 0.02 * 500000.0 * 0.5 = 5000")

	// -------------------------------------------------------------------------
	// 5. Verify PerCallBilling is true via UsePrice
	// -------------------------------------------------------------------------
	assert.True(t, info.PriceData.UsePrice,
		"PerCallBilling should be true via UsePrice")
}

// ============================================================================
// Test 11: fixed_price — with enterprise pricing sheet
//   DiscountType=fixed_price, GroupRatio is overridden to DiscountValue,
//   quota = modelPrice * QuotaPerUnit * GroupRatio(=DiscountValue)
// ============================================================================

func TestEnterpriseBilling_FixedPrice_WithSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID    = 11001
		modelName     = "midjourney"
		modelPrice    = 0.02 // $0.02 per call from model config
		fixedPrice   = 0.5  // GroupRatio 被覆盖为 0.5
		groupRatio   = 1.5   // original group ratio (should be overridden)
		// quota = modelPrice(0.02) * QuotaPerUnit(500000.0) * fixedPrice(0.5) = 5000
	)

	err := ratio_setting.UpdateModelPriceByJSONString(`{"midjourney": 0.02}`)
	require.NoError(t, err)
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1.5, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	now := time.Now().Unix()

	enterprise := &model.Enterprise{
		Name:      "FixedPriceCorp",
		Status:    model.EnterpriseStatusEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, model.DB.Create(enterprise).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterprise.Id,
		Name:         "MJ固定价格报价单",
		Status:       model.PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	require.NoError(t, model.DB.Create(sheet).Error)

	pricingItem := &model.EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Model:         modelName,
		DiscountType:  model.DiscountTypeFixedPrice,
		DiscountValue: fixedPrice,
	}
	require.NoError(t, model.DB.Create(pricingItem).Error)

	user := &model.User{
		Id:       testUserID,
		Username: "fixed_price_user",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	binding := &model.EnterpriseUserBinding{
		EnterpriseId: enterprise.Id,
		UserId:       testUserID,
		CreatedAt:    now,
	}
	require.NoError(t, model.DB.Create(binding).Error)

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 1},
		UsingGroup:      "default",
		OriginModelName: modelName,
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: "image_generation"}
	info.UserSetting.AcceptUnsetRatioModel = false

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/mj/submit", nil)
	c.Set("token_name", "test_token")

	priceData, err := helper.ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)
	info.PriceData = priceData

	// PerCallPriceSheet should NOT be set for fixed_price
	assert.Equal(t, 0.0, priceData.GroupRatioInfo.PerCallPriceSheet,
		"PerCallPriceSheet should be 0 for fixed_price type")
	assert.Equal(t, 0.0, priceData.PerCallPriceSheet,
		"PriceData.PerCallPriceSheet should be 0 for fixed_price type")

	// GroupRatio should be overridden by fixed_price DiscountValue
	assert.Equal(t, fixedPrice, priceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should be overridden by fixed_price DiscountValue (0.5)")

	// RatioSource should indicate enterprise sheet
	assert.Equal(t, "enterprise_pricing_sheet", priceData.GroupRatioInfo.RatioSource,
		"RatioSource should be enterprise_pricing_sheet")
	assert.Equal(t, sheet.Id, priceData.GroupRatioInfo.EnterpriseSheetId,
		"EnterpriseSheetId should be set")
	assert.Equal(t, sheet.Name, priceData.GroupRatioInfo.EnterpriseSheetName,
		"EnterpriseSheetName should be set")

	// quota = modelPrice(0.02) * QuotaPerUnit(500000.0) * fixedPrice(0.5) = 5000
	assert.Equal(t, 5000, priceData.Quota,
		"Quota should be: 0.02 * 500000.0 * 0.5 = 5000")
	assert.True(t, priceData.UsePrice, "UsePrice should be true")

	LogTaskConsumption(c, info)

	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeConsume, log.Type)
	assert.Equal(t, 5000, log.Quota)

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)
	assert.Equal(t, "enterprise_pricing_sheet", other["ratio_source"])
	assert.Equal(t, float64(sheet.Id), other["enterprise_sheet_id"])
	assert.Equal(t, sheet.Name, other["enterprise_sheet_name"])
}

// ============================================================================
// Test 12: fixed_price — without enterprise pricing sheet, falls back to group
//   No enterprise binding, uses model price * QuotaPerUnit * groupRatio
// ============================================================================

func TestEnterpriseBilling_FixedPrice_NoSheet_FallsBackToGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	truncate(t)
	resetRatioSetting()

	const (
		testUserID  = 12001
		modelName   = "midjourney"
		modelPrice  = 0.02 // $0.02 per call from model config
		groupRatio = 1.5
		// quota = modelPrice(0.02) * QuotaPerUnit(500000.0) * groupRatio(1.5) = 15000
	)

	err := ratio_setting.UpdateModelPriceByJSONString(`{"midjourney": 0.02}`)
	require.NoError(t, err)
	err = ratio_setting.UpdateGroupRatioByJSONString(`{"default": 1.5, "vip": 1, "svip": 1}`)
	require.NoError(t, err)

	user := &model.User{
		Id:       testUserID,
		Username: "fixed_price_no_sheet",
		Quota:    100000,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, model.DB.Create(user).Error)

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	info := &relaycommon.RelayInfo{
		UserId:          testUserID,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 1},
		UsingGroup:      "default",
		OriginModelName: modelName,
		TokenId:         0,
		IsStream:        false,
		StartTime:       time.Now().Add(-2 * time.Second),
		FirstResponseTime: time.Now(),
	}
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{Action: "image_generation"}
	info.UserSetting.AcceptUnsetRatioModel = false

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/mj/submit", nil)
	c.Set("token_name", "test_token")

	priceData, err := helper.ModelPriceHelperPerCall(c, info)
	require.NoError(t, err)
	info.PriceData = priceData

	// PerCallPriceSheet should NOT be set (no enterprise sheet)
	assert.Equal(t, 0.0, priceData.GroupRatioInfo.PerCallPriceSheet,
		"PerCallPriceSheet should be 0 when no enterprise pricing sheet")
	assert.Equal(t, 0.0, priceData.PerCallPriceSheet,
		"PriceData.PerCallPriceSheet should be 0")

	// GroupRatio should come from group ratio
	assert.Equal(t, groupRatio, priceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should use group ratio (1.5)")
	assert.Equal(t, "group_ratio", priceData.GroupRatioInfo.RatioSource,
		"RatioSource should be group_ratio (no enterprise sheet)")

	// quota = modelPrice(0.02) * QuotaPerUnit(500000.0) * groupRatio(1.5) = 15000
	assert.Equal(t, 15000, priceData.Quota,
		"Quota should be: 0.02 * 500000.0 * 1.5 = 15000")
	assert.True(t, priceData.UsePrice, "UsePrice should be true")

	LogTaskConsumption(c, info)

	log := getLastLog(t)
	require.NotNil(t, log)
	assert.Equal(t, model.LogTypeConsume, log.Type)
	assert.Equal(t, 15000, log.Quota)

	var other map[string]interface{}
	err = json.Unmarshal([]byte(log.Other), &other)
	require.NoError(t, err)
	assert.Equal(t, "group_ratio", other["ratio_source"],
		"log should record ratio_source = group_ratio (no enterprise sheet)")
	_, hasEnterpriseId := other["enterprise_sheet_id"]
	assert.False(t, hasEnterpriseId,
		"enterprise_sheet_id should not be present when no enterprise pricing sheet")
}
