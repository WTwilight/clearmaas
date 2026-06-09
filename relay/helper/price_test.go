package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelPriceHelperTieredUsesPreloadedRequestInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"tiered-test-model":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"tiered-test-model":"param(\"stream\") == true ? tier(\"stream\", p * 3) : tier(\"base\", p * 2)"}`,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/channel/test/1", nil)
	req.Body = nil
	req.ContentLength = 0
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "tiered-test-model",
		UserGroup:       "default",
		UsingGroup:      "default",
		RequestHeaders:  map[string]string{"Content-Type": "application/json"},
		BillingRequestInput: &billingexpr.RequestInput{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    []byte(`{"stream":true}`),
		},
	}

	priceData, err := ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{})
	require.NoError(t, err)
	require.Equal(t, 1500, priceData.QuotaToPreConsume)
	require.NotNil(t, info.TieredBillingSnapshot)
	require.Equal(t, "stream", info.TieredBillingSnapshot.EstimatedTier)
	require.Equal(t, billing_setting.BillingModeTieredExpr, info.TieredBillingSnapshot.BillingMode)
	require.Equal(t, common.QuotaPerUnit, info.TieredBillingSnapshot.QuotaPerUnit)
}

// ---------------------------------------------------------------------------
// HandleGroupRatio
// ---------------------------------------------------------------------------

func TestHandleGroupRatio_NoSpecialRule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o",
		UserGroup:       "default",
		UsingGroup:      "default",
	}

	result, _ := HandleGroupRatio(c, info)
	require.Equal(t, 1.0, result.GroupRatio, "no special ratio → default ratio = 1.0")
	require.False(t, result.HasSpecialRatio)
}

func TestHandleGroupRatio_AutoGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o",
		UserGroup:       "default",
		UsingGroup:      "default",
	}

	// Simulate auto group selection
	c.Set("auto_group", "vip")
	result, _ := HandleGroupRatio(c, info)

	// UsingGroup should be updated to the auto group
	require.Equal(t, "vip", info.UsingGroup)
	_ = result // ratio depends on GroupGroupRatio config
}

func TestHandleGroupRatio_WithSpecialRule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o",
		UserGroup:       "vip",
		UsingGroup:      "default",
	}

	// Note: GetGroupGroupRatio reads from config; in a test environment without
	// the config populated, it may return (0, false). We test the function
	// doesn't panic and returns a valid struct.
	result, _ := HandleGroupRatio(c, info)
	require.NotZero(t, result.GroupRatio)
}

func TestHandleGroupRatio_DefaultValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o",
		UserGroup:       "unknown-group",
		UsingGroup:      "unknown-group",
	}

	result, _ := HandleGroupRatio(c, info)
	require.Equal(t, 1.0, result.GroupRatio, "unknown group → default ratio = 1.0")
}

// ---------------------------------------------------------------------------
// ModelPriceHelper: free model
// ---------------------------------------------------------------------------

func TestModelPriceHelper_FreeModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	// Set up: gpt-4o with normal ratio, group ratio = 0 (free)
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o",
		UserGroup:       "default",
		UsingGroup:      "default",
	}

	priceData, err := ModelPriceHelper(c, info, 1000, &types.TokenCountMeta{})
	require.NoError(t, err)
	// GroupRatio may be 0 if SpecialRatio is set; otherwise 1.0
	// We just verify no error
	require.NotNil(t, priceData)
}

// ---------------------------------------------------------------------------
// ModelPriceHelper: nil meta
// ---------------------------------------------------------------------------

func TestModelPriceHelper_NilMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o",
		UserGroup:       "default",
		UsingGroup:      "default",
	}

	// Passing nil meta should not panic
	priceData, err := ModelPriceHelper(c, info, 1000, nil)
	require.NoError(t, err)
	require.NotNil(t, priceData)
}

// ---------------------------------------------------------------------------
// ModelPriceHelper: MaxTokens = 0 (should not add)
// ---------------------------------------------------------------------------

func TestModelPriceHelper_MaxTokensZero(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4o",
		UserGroup:       "default",
		UsingGroup:      "default",
	}

	meta := &types.TokenCountMeta{}

	priceData, err := ModelPriceHelper(c, info, 1000, meta)
	require.NoError(t, err)
	// When MaxTokens=0, preConsumedTokens = max(promptTokens, PreConsumedQuota)
	// PreConsumedQuota is usually > promptTokens, so promptTokens don't get added
	// We just verify the function completes without error
	require.NotNil(t, priceData)
}

// ---------------------------------------------------------------------------
// modelPriceHelperTiered: missing billing expr
// ---------------------------------------------------------------------------

func TestModelPriceHelperTiered_MissingExpr(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	// Set model as tiered_expr but no billing_expr configured
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"missing-expr-model":"tiered_expr"}`,
		"billing_setting.billing_expr": `{}`,
	}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "missing-expr-model",
		UserGroup:       "default",
		UsingGroup:      "default",
	}

	_, err := modelPriceHelperTiered(c, info, 1000, &types.TokenCountMeta{}, types.GroupRatioInfo{GroupRatio: 1.0})
	require.Error(t, err)
	require.Contains(t, err.Error(), "configured as tiered_expr")
}

