package helper

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/types"
	"github.com/glebarez/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestMain sets up the shared test DB with seeded enterprise pricing data.
// Each user has a different setup:
//   - userId=1: bound to enterprise but no active pricing sheet
//   - userId=2: bound to enterprise with sheet that has gpt-4o-mini at 0.5
//   - userId=3: bound to enterprise with sheet that has gpt-4o at 0.5
func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db: " + err.Error())
	}

	model.DB = db
	model.LOG_DB = db

	common.UsingSQLite = true
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true

	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&model.Enterprise{},
		&model.EnterprisePricingSheet{},
		&model.EnterprisePricingItem{},
		&model.EnterpriseUserBinding{},
		&model.User{},
	); err != nil {
		panic("failed to migrate: " + err.Error())
	}

	// Seed test data for billing integration tests
	seedBillingTestData(db)

	os.Exit(m.Run())
}

// seedBillingTestData creates the user/enterprise/pricing data referenced by billing tests.
func seedBillingTestData(db *gorm.DB) {
	now := time.Now().Unix()

	// --- User 1: bound to enterprise, no active pricing sheet ---
	e1 := &model.Enterprise{Name: "Corp1", Status: model.EnterpriseStatusEnabled}
	e1.CreatedAt = now
	e1.UpdatedAt = now
	db.Create(e1)

	u1 := &model.User{Id: 1, Username: "user1", Status: 1, Quota: 1000, AffCode: "aff1"}
	u1.CreatedAt = now
	db.Create(u1)

	b1 := &model.EnterpriseUserBinding{EnterpriseId: e1.Id, UserId: u1.Id}
	b1.CreatedAt = now
	db.Create(b1)
	// Note: e1 has NO pricing sheet

	// --- User 2: bound to enterprise with sheet that has gpt-4o-mini (not gpt-4o) ---
	e2 := &model.Enterprise{Name: "Corp2", Status: model.EnterpriseStatusEnabled}
	e2.CreatedAt = now
	e2.UpdatedAt = now
	db.Create(e2)

	u2 := &model.User{Id: 2, Username: "user2", Status: 1, Quota: 1000, AffCode: "aff2"}
	u2.CreatedAt = now
	db.Create(u2)

	b2 := &model.EnterpriseUserBinding{EnterpriseId: e2.Id, UserId: u2.Id}
	b2.CreatedAt = now
	db.Create(b2)

	sheet2 := &model.EnterprisePricingSheet{
		EnterpriseId: e2.Id,
		Name:        "Corp2报价单",
		Status:      model.PricingSheetStatusActive,
		StartTime:   now - 86400,
		EndTime:     now + 86400,
	}
	sheet2.CreatedAt = now
	sheet2.UpdatedAt = now
	db.Create(sheet2)

	item2 := &model.EnterprisePricingItem{
		PricingSheetId: sheet2.Id,
		Model:          "gpt-4o-mini",
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.5,
	}
	db.Create(item2)

	// --- User 3: bound to enterprise with sheet that has gpt-4o at 0.5 ---
	e3 := &model.Enterprise{Name: "Corp3", Status: model.EnterpriseStatusEnabled}
	e3.CreatedAt = now
	e3.UpdatedAt = now
	db.Create(e3)

	u3 := &model.User{Id: 3, Username: "user3", Status: 1, Quota: 1000, AffCode: "aff3"}
	u3.CreatedAt = now
	db.Create(u3)

	b3 := &model.EnterpriseUserBinding{EnterpriseId: e3.Id, UserId: u3.Id}
	b3.CreatedAt = now
	db.Create(b3)

	sheet3 := &model.EnterprisePricingSheet{
		EnterpriseId: e3.Id,
		Name:        "测试报价单",
		Status:      model.PricingSheetStatusActive,
		StartTime:   now - 86400,
		EndTime:     now + 86400,
	}
	sheet3.CreatedAt = now
	sheet3.UpdatedAt = now
	db.Create(sheet3)

	item3 := &model.EnterprisePricingItem{
		PricingSheetId: sheet3.Id,
		Model:          "gpt-4o",
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.5,
	}
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
		UserId:       99999,
		UsingGroup:   "default",
		OriginModelName: "gpt-4o",
	}

	// Base group ratio = 1.0
	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:      1.0,
		GroupSpecialRatio: -1,
		HasSpecialRatio: false,
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
		UserId:       1,
		UsingGroup:   "default",
		OriginModelName: "gpt-4o",
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:      1.5, // some group ratio
		GroupSpecialRatio: -1,
		HasSpecialRatio: false,
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
		UserId:       2,
		UsingGroup:   "default",
		OriginModelName: "gpt-4o", // not in sheet
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:      2.0,
		GroupSpecialRatio: -1,
		HasSpecialRatio: false,
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
	// (userId=3 scenario)
	info := &relaycommon.RelayInfo{
		UserId:       3,
		UsingGroup:   "default",
		OriginModelName: "gpt-4o",
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:      1.5, // original group ratio
		GroupSpecialRatio: 2.0,
		HasSpecialRatio: true,
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
		UserId:       3,
		UsingGroup:   "default",
		OriginModelName: "gpt-4o",
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:      0.8, // GroupGroupRatio already applied
		GroupSpecialRatio: 0.8,
		HasSpecialRatio: true,
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
		UserId:       3,
		UsingGroup:   "default",
		OriginModelName: "completely-unknown-model-xyz",
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:      1.5,
		GroupSpecialRatio: -1,
		HasSpecialRatio: false,
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
		UserId:       3,
		UsingGroup:   "default",
		OriginModelName: "GPT-4O", // uppercase — should NOT match
	}

	baseRatioInfo := types.GroupRatioInfo{
		GroupRatio:      1.5,
		GroupSpecialRatio: -1,
		HasSpecialRatio: false,
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
		UserId:       3,
		UsingGroup:   "default",
		OriginModelName: "gpt-4o",
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

func TestModelPriceHelper_UsesEnterprisePricing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", "default")

	info := &relaycommon.RelayInfo{
		UserId:       3,
		UsingGroup:   "default",
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

func TestModelPriceHelper_EnterprisePricingPreConsumeQuota(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("group", "default")

	info := &relaycommon.RelayInfo{
		UserId:       3,
		UsingGroup:   "default",
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
		name         string
		userId       int
		model        string
		wantRatio    float64
		wantSource   string
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
				UserId:         tt.userId,
				UsingGroup:     "default",
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
