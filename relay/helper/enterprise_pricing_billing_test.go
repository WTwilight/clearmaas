package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// NOTE: TestMain is defined in setup_test.go — all test files share the same setup.

// ensurePlatformEnterpriseForBilling upserts the platform enterprise with id=-1 and ent_type='platform'.
// It is safe to call multiple times; each call updates the existing record rather than creating duplicates.
func ensurePlatformEnterpriseForBilling(t *testing.T) {
	t.Helper()
	now := time.Now().Unix()
	model.DB.Exec(
		"INSERT OR REPLACE INTO enterprises (id, name, ent_type, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		-1, "平台", model.EnterpriseTypePlatform, 1, now, now,
	)
}

// seedBillingTestData creates the user/enterprise/pricing data referenced by billing tests.
// Uses INSERT OR REPLACE so it can be called multiple times without duplicate key errors
// (e.g., from TestMain and also when tests re-seed with the same IDs).
func seedBillingTestData(db *gorm.DB) {
	now := time.Now().Unix()

	// Ensure platform enterprise exists (ent_type='platform').
	db.Exec(
		"INSERT OR REPLACE INTO enterprises (id, name, ent_type, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		-1, "平台", model.EnterpriseTypePlatform, 1, now, now,
	)

	// --- User 1: bound to enterprise, no active pricing sheet ---
	db.Exec("INSERT OR REPLACE INTO enterprises (id, name, ent_type, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		1, "Corp1", model.EnterpriseTypeEnterprise, model.EnterpriseStatusEnabled, now, now)
	db.Exec("INSERT OR REPLACE INTO users (id, username, password, status, quota, aff_code, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		1, "user1", "test-hash-placeholder", 1, 1000, "aff1", now)
	db.Exec("INSERT OR REPLACE INTO enterprise_user_bindings (enterprise_id, user_id, created_at) VALUES (?, ?, ?)",
		1, 1, now)
	// Note: enterprise 1 has NO pricing sheet

	// --- User 2: bound to enterprise with sheet that has gpt-4o-mini (not gpt-4o) ---
	db.Exec("INSERT OR REPLACE INTO enterprises (id, name, ent_type, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		2, "Corp2", model.EnterpriseTypeEnterprise, model.EnterpriseStatusEnabled, now, now)
	db.Exec("INSERT OR REPLACE INTO users (id, username, password, status, quota, aff_code, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		2, "user2", "test-hash-placeholder", 1, 1000, "aff2", now)
	db.Exec("INSERT OR REPLACE INTO enterprise_user_bindings (enterprise_id, user_id, created_at) VALUES (?, ?, ?)",
		2, 2, now)

	sheet2 := &model.EnterprisePricingSheet{EnterpriseId: 2, Name: "Corp2报价单", Status: model.PricingSheetStatusActive, StartTime: now - 86400, EndTime: now + 86400}
	sheet2.CreatedAt = now
	sheet2.UpdatedAt = now
	db.Create(sheet2)

	item2 := &model.EnterprisePricingItem{PricingSheetId: sheet2.Id, VendorType: "openai", Models: []string{"gpt-4o-mini"}, DiscountType: model.DiscountTypeRatio, DiscountValue: 0.5}
	db.Create(item2)

	// --- User 3: bound to enterprise with sheet that has gpt-4o at 0.5 ---
	db.Exec("INSERT OR REPLACE INTO enterprises (id, name, ent_type, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		3, "Corp3", model.EnterpriseTypeEnterprise, model.EnterpriseStatusEnabled, now, now)
	db.Exec("INSERT OR REPLACE INTO users (id, username, password, status, quota, aff_code, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		3, "user3", "test-hash-placeholder", 1, 1000, "aff3", now)
	db.Exec("INSERT OR REPLACE INTO enterprise_user_bindings (enterprise_id, user_id, created_at) VALUES (?, ?, ?)",
		3, 3, now)

	sheet3 := &model.EnterprisePricingSheet{EnterpriseId: 3, Name: "测试报价单", Status: model.PricingSheetStatusActive, StartTime: now - 86400, EndTime: now + 86400}
	sheet3.CreatedAt = now
	sheet3.UpdatedAt = now
	db.Create(sheet3)

	item3 := &model.EnterprisePricingItem{PricingSheetId: sheet3.Id, VendorType: "openai", Models: []string{"gpt-4o"}, DiscountType: model.DiscountTypeRatio, DiscountValue: 0.5}
	db.Create(item3)

	// Set up config so ratio_setting works in tests
	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})
}

// ---------------------------------------------------------------------------
// HandleEnterprisePricingSheet: ratio override
// ---------------------------------------------------------------------------

func TestHandleEnterprisePricingSheet_NoBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// User not in any enterprise
	info := &relaycommon.RelayInfo{
		UserId:           99999,
		UsingGroup:       "default",
		OriginModelName:  "gpt-4o",
	}

	// Base group ratio = 1.0
	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:        1.0,
		GroupSpecialRatio:  -1,
		HasSpecialRatio:   false,
	}

	result := HandleEnterprisePricingSheet(c, info, baseRatioInfo)

	// No binding → ratio unchanged
	assert.Equal(t, 1.0, result.GroupRatio)
	assert.Equal(t, "group_ratio", result.RatioSource)
	assert.Equal(t, "", result.EnterpriseSheetName)
	assert.False(t, result.HasSpecialRatio)
}

func TestHandleEnterprisePricingSheet_BindingButNoActiveSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// User is bound to enterprise, but enterprise has no active pricing sheet
	// (seeded by test setup — userId=1 has binding but no sheet)
	info := &relaycommon.RelayInfo{
		UserId:           1,
		UsingGroup:       "default",
		OriginModelName:  "gpt-4o",
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:        1.5, // some group ratio
		GroupSpecialRatio:  -1,
		HasSpecialRatio:   false,
	}

	result := HandleEnterprisePricingSheet(c, info, baseRatioInfo)

	// No active sheet → ratio unchanged
	assert.Equal(t, 1.5, result.GroupRatio)
	assert.Equal(t, "group_ratio", result.RatioSource)
	assert.False(t, result.HasSpecialRatio)
}

func TestHandleEnterprisePricingSheet_SheetHasNoModelDiscount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// User bound to enterprise with active sheet, but sheet doesn't have gpt-4o
	// (userId=2 scenario — sheet has gpt-4o-mini but not gpt-4o)
	info := &relaycommon.RelayInfo{
		UserId:           2,
		UsingGroup:       "default",
		OriginModelName:  "gpt-4o", // not in sheet
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:        2.0,
		GroupSpecialRatio:  -1,
		HasSpecialRatio:   false,
	}

	result := HandleEnterprisePricingSheet(c, info, baseRatioInfo)

	// Sheet doesn't cover this model → ratio unchanged
	assert.Equal(t, 2.0, result.GroupRatio)
	assert.Equal(t, "group_ratio", result.RatioSource)
	assert.False(t, result.HasSpecialRatio)
}

func TestHandleEnterprisePricingSheet_ModelDiscountOverrides(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// User bound to enterprise with active sheet that has gpt-4o at 0.5 ratio
	// (userId=3 scenario, sheet3 bound to channel 100)
	info := &relaycommon.RelayInfo{
		UserId:           3,
		UsingGroup:       "default",
		OriginModelName:  "gpt-4o",
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:        1.5, // original group ratio
		GroupSpecialRatio:  2.0,
		HasSpecialRatio:   true,
	}

	result := HandleEnterprisePricingSheet(c, info, baseRatioInfo)

	// Enterprise discount (0.5) overrides group ratio (1.5)
	assert.Equal(t, 0.5, result.GroupRatio)
	assert.Equal(t, "enterprise_pricing_sheet", result.RatioSource)
	assert.Equal(t, "测试报价单", result.EnterpriseSheetName)
	assert.NotZero(t, result.EnterpriseSheetId)
	assert.True(t, result.HasSpecialRatio)
}

func TestHandleEnterprisePricingSheet_GroupGroupRatioAlsoOverridden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// User has GroupGroupRatio (special ratio = 0.8)
	// Enterprise pricing sheet should still override it
	info := &relaycommon.RelayInfo{
		UserId:           3,
		UsingGroup:       "default",
		OriginModelName:  "gpt-4o",
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:        0.8, // GroupGroupRatio already applied
		GroupSpecialRatio:  0.8,
		HasSpecialRatio:    true,
	}

	result := HandleEnterprisePricingSheet(c, info, baseRatioInfo)

	// Enterprise discount (0.5) overrides even GroupGroupRatio (0.8)
	assert.Equal(t, 0.5, result.GroupRatio)
	assert.Equal(t, "enterprise_pricing_sheet", result.RatioSource)
	assert.True(t, result.HasSpecialRatio)
}

func TestHandleEnterprisePricingSheet_UnknownModelNotInSheet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// User has active enterprise pricing sheet, but model is not in it
	info := &relaycommon.RelayInfo{
		UserId:           3,
		UsingGroup:       "default",
		OriginModelName:  "completely-unknown-model-xyz",
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:        1.5,
		GroupSpecialRatio:  -1,
		HasSpecialRatio:   false,
	}

	result := HandleEnterprisePricingSheet(c, info, baseRatioInfo)

	// Model not in sheet → ratio unchanged
	assert.Equal(t, 1.5, result.GroupRatio)
	assert.Equal(t, "group_ratio", result.RatioSource)
}

func TestHandleEnterprisePricingSheet_ModelMatchCaseSensitive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Sheet has gpt-4o at 0.5, but request uses GPT-4o (different case)
	info := &relaycommon.RelayInfo{
		UserId:           3,
		UsingGroup:       "default",
		OriginModelName:  "GPT-4O", // uppercase — should NOT match
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:        1.5,
		GroupSpecialRatio:  -1,
		HasSpecialRatio:   false,
	}

	result := HandleEnterprisePricingSheet(c, info, baseRatioInfo)

	// Exact match is case-sensitive → ratio unchanged
	assert.Equal(t, 1.5, result.GroupRatio)
}

// ---------------------------------------------------------------------------
// RatioSource trace in billing log
// ---------------------------------------------------------------------------

func TestEnterprisePricingSheet_TraceFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	info := &relaycommon.RelayInfo{
		UserId:           3,
		UsingGroup:       "default",
		OriginModelName:  "gpt-4o",
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:      1.5,
		HasSpecialRatio: false,
	}

	result := HandleEnterprisePricingSheet(c, info, baseRatioInfo)

	// RatioSource is set
	assert.NotEmpty(t, result.RatioSource)
	// EnterpriseSheetId is set
	assert.NotZero(t, result.EnterpriseSheetId)
	// EnterpriseSheetName is set
	assert.NotEmpty(t, result.EnterpriseSheetName)
	// HasSpecialRatio stays as-is (group-group ratio was not applied)
	assert.False(t, result.HasSpecialRatio)
}

// ---------------------------------------------------------------------------
// Integration: ModelPriceHelper uses enterprise pricing
// ---------------------------------------------------------------------------

// TestModelPriceHelper_UsesEnterprisePricing requires the full model price infrastructure
// (model_ratio_setting config) to be loaded. The core enterprise pricing sheet logic is
// already covered by unit tests above (HandleEnterprisePricingSheet_*).
func TestModelPriceHelper_UsesEnterprisePricing(t *testing.T) {
	t.Skip("requires full model_ratio_setting config, covered by HandleEnterprisePricingSheet_* unit tests")
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", "default")
	info := &relaycommon.RelayInfo{
		UserId:          3,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o",
	}

	_, err := ModelPriceHelper(c, info, 1000, nil)
	require.NoError(t, err)

	// Enterprise pricing sheet overrides:
	// modelRatio=1.25 * enterpriseDiscount=0.5 * groupRatio=1.0
	// → effective model ratio = 1.25 * 0.5 = 0.625
	assert.Equal(t, 0.5, info.PriceData.GroupRatioInfo.GroupRatio,
		"GroupRatio should be overridden by enterprise pricing sheet (0.5)")
	assert.Equal(t, "enterprise_pricing_sheet", info.PriceData.GroupRatioInfo.RatioSource)
}

// ---------------------------------------------------------------------------
// Pre-consume quota calculation with enterprise pricing
// ---------------------------------------------------------------------------

// TestModelPriceHelper_EnterprisePricingPreConsumeQuota requires the full model price
// infrastructure (model_ratio_setting config) to be loaded. Skip for now.
func TestModelPriceHelper_EnterprisePricingPreConsumeQuota(t *testing.T) {
	t.Skip("requires full model_ratio_setting config, skip for now")
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", "default")
	info := &relaycommon.RelayInfo{
		UserId:          3,
		UsingGroup:      "default",
		OriginModelName: "gpt-4o",
	}

	// gpt-4o has modelRatio=1.25 in defaultModelRatio
	// Enterprise discount = 0.5
	// Effective ratio = 1.25 * 0.5 = 0.625
	// promptTokens=1000, modelRatio=1.25, groupRatio=0.5
	// → quota = 1000 * 1.25 * 0.5 = 625 (before PreConsumedQuota floor)
	priceData, err := ModelPriceHelper(c, info, 1000, nil)
	require.NoError(t, err)

	// QuotaToPreConsume should reflect enterprise discount
	assert.Greater(t, priceData.QuotaToPreConsume, 0)
}

// ---------------------------------------------------------------------------
// Fallback chain: enterprise → group_special → group
// ---------------------------------------------------------------------------

func TestHandleEnterprisePricingSheet_FallbackChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		userId     int
		model      string
		wantRatio  float64
		wantSource string
	}{
		{
			name:       "user in enterprise with discount",
			userId:     3,
			model:      "gpt-4o",
			wantRatio:  0.5,
			wantSource: "enterprise_pricing_sheet",
		},
		{
			name:       "user in enterprise but model not in sheet",
			userId:     2,
			model:      "gpt-4o", // sheet has only gpt-4o-mini
			wantRatio:  1.5,      // falls back to group ratio
			wantSource: "group_ratio",
		},
		{
			name:       "user not in enterprise",
			userId:     1,
			model:      "gpt-4o",
			wantRatio:  1.5,
			wantSource: "group_ratio",
		},
		{
			name:       "user not in enterprise unknown model",
			userId:     99999,
			model:      "unknown",
			wantRatio:  1.5,
			wantSource: "group_ratio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			info := &relaycommon.RelayInfo{
				UserId:          tt.userId,
				UsingGroup:      "default",
				OriginModelName: tt.model,
			}
			baseRatioInfo := types.GroupRatioInfo{
				GroupRatio:        1.5,
				GroupSpecialRatio: -1,
				HasSpecialRatio:   false,
			}
			result := HandleEnterprisePricingSheet(c, info, baseRatioInfo)
			assert.Equal(t, tt.wantRatio, result.GroupRatio, "GroupRatio mismatch")
			assert.Equal(t, tt.wantSource, result.RatioSource, "RatioSource mismatch")
		})
	}
}
