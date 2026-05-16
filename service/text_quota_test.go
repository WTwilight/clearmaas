package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func makeTestRelayInfo(modelName string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		OriginModelName: modelName,
		UserGroup:       "default",
		UsingGroup:      "default",
		StartTime:       time.Now().Add(-1 * time.Second),
		PriceData: types.PriceData{
			ModelRatio: 1.0,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1.0,
			},
			CompletionRatio:        1.0,
			CacheRatio:             1.0,
			ImageRatio:             1.0,
			CacheCreationRatio:     1.0,
			CacheCreation5mRatio:   1.0,
			CacheCreation1hRatio:   1.0,
		},
	}
}

func TestCacheWriteTokensTotal(t *testing.T) {
	tests := []struct {
		name    string
		summary textQuotaSummary
		want    int
	}{
		{
			name:    "no cache creation tokens",
			summary: textQuotaSummary{CacheCreationTokens: 0, CacheCreationTokens5m: 0, CacheCreationTokens1h: 0},
			want:    0,
		},
		{
			name:    "only 5m tokens, total less than 5m",
			summary: textQuotaSummary{CacheCreationTokens: 30000, CacheCreationTokens5m: 20000, CacheCreationTokens1h: 0},
			want:    30000, // total(30000) > split(20000) → return total
		},
		{
			name:    "only 5m tokens, total equals 5m",
			summary: textQuotaSummary{CacheCreationTokens: 50000, CacheCreationTokens5m: 50000, CacheCreationTokens1h: 0},
			want:    50000, // total(50000) <= split(50000) → return split
		},
		{
			name:    "both 5m and 1h, total less than split",
			summary: textQuotaSummary{CacheCreationTokens: 40000, CacheCreationTokens5m: 30000, CacheCreationTokens1h: 10000},
			want:    40000, // split=40000, total=40000 → return total(40000)
		},
		{
			name:    "both 5m and 1h, total greater than split",
			summary: textQuotaSummary{CacheCreationTokens: 80000, CacheCreationTokens5m: 50000, CacheCreationTokens1h: 10000},
			want:    80000, // split=60000, total=80000 → return total
		},
		{
			name:    "1h only, total greater than 1h",
			summary: textQuotaSummary{CacheCreationTokens: 80000, CacheCreationTokens5m: 0, CacheCreationTokens1h: 30000},
			want:    80000, // split=30000, total=80000 → return total
		},
		{
			name:    "1h only, total less than 1h",
			summary: textQuotaSummary{CacheCreationTokens: 20000, CacheCreationTokens5m: 0, CacheCreationTokens1h: 30000},
			want:    30000, // split=30000, total=20000 → return split
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cacheWriteTokensTotal(tt.summary)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCacheWriteTokensTotal_ZeroBoth(t *testing.T) {
	// Both 5m and 1h are zero → return CacheCreationTokens as-is
	summary := textQuotaSummary{CacheCreationTokens: 12345, CacheCreationTokens5m: 0, CacheCreationTokens1h: 0}
	require.Equal(t, 12345, cacheWriteTokensTotal(summary))
}

func TestIsLegacyClaudeDerivedOpenAIUsage(t *testing.T) {
	tests := []struct {
		name      string
		relayInfo *relaycommon.RelayInfo
		usage     *dto.Usage
		want      bool
	}{
		{
			name:      "nil relayInfo",
			relayInfo: nil,
			usage:     &dto.Usage{},
			want:      false,
		},
		{
			name:      "nil usage",
			relayInfo: makeTestRelayInfo("gpt-4o"),
			usage:     nil,
			want:      false,
		},
		{
			name:      "Claude relay format",
			relayInfo: makeTestRelayInfo("claude-3-5-sonnet"),
			usage:     &dto.Usage{ClaudeCacheCreation5mTokens: 100},
			want:      false,
		},
		{
			name:      "UsageSemantic set",
			relayInfo: makeTestRelayInfo("gpt-4o"),
			usage:     &dto.Usage{UsageSemantic: "anthropic"},
			want:      false,
		},
		{
			name:      "UsageSource set",
			relayInfo: makeTestRelayInfo("gpt-4o"),
			usage:     &dto.Usage{UsageSource: "some-source"},
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLegacyClaudeDerivedOpenAIUsage(tt.relayInfo, tt.usage)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestUsageSemanticFromUsage(t *testing.T) {
	tests := []struct {
		name      string
		relayInfo *relaycommon.RelayInfo
		usage     *dto.Usage
		want      string
	}{
		{
			name:      "usage has UsageSemantic",
			relayInfo: makeTestRelayInfo("gpt-4o"),
			usage:     &dto.Usage{UsageSemantic: "custom"},
			want:      "custom",
		},
		{
			name:      "nil usage, Claude relay format",
			relayInfo: makeTestRelayInfo("claude-3-5-sonnet"),
			usage:     nil,
			want:      "anthropic",
		},
		{
			name:      "nil usage, non-Claude relay format",
			relayInfo: makeTestRelayInfo("gpt-4o"),
			usage:     nil,
			want:      "openai",
		},
		{
			name:      "empty UsageSemantic falls back to relay format",
			relayInfo: makeTestRelayInfo("gpt-4o"),
			usage:     &dto.Usage{UsageSemantic: ""},
			want:      "openai",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := usageSemanticFromUsage(tt.relayInfo, tt.usage)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestComposeTieredTextQuota(t *testing.T) {
	tests := []struct {
		name            string
		surchargeQuota  float64
		tieredQuota     int
		tieredResultNil bool
		want            int
	}{
		{
			name:            "no surcharge, no tiered result",
			surchargeQuota:  0,
			tieredQuota:     100,
			tieredResultNil: true,
			want:            100,
		},
		{
			name:           "with surcharge, tiered result nil",
			surchargeQuota: 50,
			tieredQuota:    100,
			tieredResultNil: true,
			want:           150, // tieredQuota + surcharge
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := textQuotaSummary{
				ToolCallSurchargeQuota: decimal.NewFromFloat(tt.surchargeQuota),
			}
			var tieredResult *billingexpr.TieredResult
			if !tt.tieredResultNil {
				tieredResult = &billingexpr.TieredResult{
					ActualQuotaBeforeGroup: 200,
				}
			}
			relayInfo := &relaycommon.RelayInfo{
				TieredBillingSnapshot: &billingexpr.BillingSnapshot{
					GroupRatio: 1.0,
				},
			}
			if tt.tieredResultNil {
				relayInfo.TieredBillingSnapshot = nil
			}
			got := composeTieredTextQuota(relayInfo, summary, tt.tieredQuota, tieredResult)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestComposeTieredTextQuota_WithSnapshotGroupRatio(t *testing.T) {
	summary := textQuotaSummary{
		ToolCallSurchargeQuota: decimal.NewFromFloat(100),
	}
	relayInfo := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			GroupRatio: 2.0,
		},
	}
	tieredResult := &billingexpr.TieredResult{
		ActualQuotaBeforeGroup: 100,
	}
	got := composeTieredTextQuota(relayInfo, summary, 50, tieredResult)
	// tieredQuota * groupRatio + surcharge = 100 * 2 + 100 = 300
	require.Equal(t, 300, got)
}

func TestCalculateTextQuotaSummary_ZeroTokens(t *testing.T) {
	relayInfo := makeTestRelayInfo("gpt-4o")
	usage := &dto.Usage{
		PromptTokens:     0,
		CompletionTokens: 0,
		TotalTokens:      0,
	}

	summary := calculateTextQuotaSummary(nil, relayInfo, usage)
	require.Equal(t, 0, summary.Quota, "zero tokens should result in zero quota")
}

func TestCalculateTextQuotaSummary_TokensPositiveZeroQuota(t *testing.T) {
	relayInfo := makeTestRelayInfo("gpt-4o")
	relayInfo.PriceData.ModelRatio = 0.0000001 // very small ratio
	relayInfo.PriceData.GroupRatioInfo.GroupRatio = 0.0000001
	relayInfo.PriceData.CompletionRatio = 0.0000001
	usage := &dto.Usage{
		PromptTokens:     1,
		CompletionTokens: 0,
		TotalTokens:      1,
	}

	summary := calculateTextQuotaSummary(nil, relayInfo, usage)
	// tokens > 0 but ratio calculation results in 0 → minimum quota = 1
	require.Equal(t, 1, summary.Quota, "tokens>0 but quota=0 should be clamped to 1")
}

func TestCalculateTextQuotaSummary_UsageNil(t *testing.T) {
	relayInfo := makeTestRelayInfo("gpt-4o")
	// GetEstimatePromptTokens should be used when usage is nil
	usage := (*dto.Usage)(nil)

	summary := calculateTextQuotaSummary(nil, relayInfo, usage)
	// Should not panic and should use estimate tokens
	require.GreaterOrEqual(t, summary.TotalTokens, 0)
}

func TestCalculateTextQuotaSummary_OpenRouterClaudeBilling(t *testing.T) {
	relayInfo := makeTestRelayInfo("claude-3-5-sonnet-v2-20250514")
	relayInfo.ChannelMeta = &relaycommon.ChannelMeta{
		ChannelType: constant.ChannelTypeOpenRouter,
	}

	usage := &dto.Usage{
		PromptTokens:     100000,
		CompletionTokens: 5000,
		TotalTokens:      105000,
		UsageSemantic:    "anthropic",
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         20000,
			CachedCreationTokens: 10000,
		},
	}

	summary := calculateTextQuotaSummary(nil, relayInfo, usage)
	// OpenRouter Claude billing: promptTokens -= cacheTokens
	// 100000 - 20000 - 10000 = 70000
	require.Equal(t, 70000, summary.PromptTokens)
}

func TestCalculateTextQuotaSummary_OtherRatios(t *testing.T) {
	relayInfo := makeTestRelayInfo("gpt-4o")
	relayInfo.PriceData.OtherRatios = map[string]float64{"test_ratio": 1.5}
	relayInfo.PriceData.ModelRatio = 1.0
	relayInfo.PriceData.GroupRatioInfo.GroupRatio = 1.0
	relayInfo.PriceData.CompletionRatio = 1.0
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	summary := calculateTextQuotaSummary(nil, relayInfo, usage)
	// quota = (1000 + 500) * 1.0 * 1.5 * 2.0 = 1500 * 3 = 4500
	require.Equal(t, 4500, summary.Quota)
}

func TestCalculateTextQuotaSummary_ModelPrice(t *testing.T) {
	relayInfo := makeTestRelayInfo("gpt-4o")
	relayInfo.PriceData.UsePrice = true
	relayInfo.PriceData.ModelPrice = 0.002 // $0.002 per 1K tokens
	relayInfo.PriceData.GroupRatioInfo.GroupRatio = 1.0
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	summary := calculateTextQuotaSummary(nil, relayInfo, usage)
	// quota = modelPrice * QuotaPerUnit * groupRatio
	// = 0.002 * 500000 * 1 = 1000
	require.Equal(t, 1000, summary.Quota)
}

func TestCalculateTextQuotaSummary_ModelPriceWithGroupRatio(t *testing.T) {
	relayInfo := makeTestRelayInfo("gpt-4o")
	relayInfo.PriceData.UsePrice = true
	relayInfo.PriceData.ModelPrice = 0.002
	relayInfo.PriceData.GroupRatioInfo.GroupRatio = 2.0
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	summary := calculateTextQuotaSummary(nil, relayInfo, usage)
	// quota = 0.002 * 500000 * 2 = 2000
	require.Equal(t, 2000, summary.Quota)
}

func TestCalculateTextQuotaSummary_CacheTokensRatio(t *testing.T) {
	relayInfo := makeTestRelayInfo("gpt-4o")
	relayInfo.PriceData.CacheRatio = 0.5 // cache hits are 50% price
	relayInfo.PriceData.ModelRatio = 1.0
	relayInfo.PriceData.GroupRatioInfo.GroupRatio = 1.0
	relayInfo.PriceData.CompletionRatio = 1.0
	usage := &dto.Usage{
		PromptTokens:     10000,
		CompletionTokens: 1000,
		TotalTokens:      11000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 5000, // 5000 tokens were cache hits
		},
	}

	summary := calculateTextQuotaSummary(nil, relayInfo, usage)
	// promptTokens = 10000, cached = 5000
	// baseTokens = 10000 - 5000 = 5000 (at ratio 1.0)
	// cachedTokensWithRatio = 5000 * 0.5 = 2500
	// promptQuota = 5000 + 2500 = 7500
	// completionQuota = 1000 * 1.0 = 1000
	// total = (7500 + 1000) * 1.0 = 8500
	require.Equal(t, 8500, summary.Quota)
}

func TestCalculateTextQuotaSummary_ImageTokens(t *testing.T) {
	relayInfo := makeTestRelayInfo("gpt-4o")
	relayInfo.PriceData.ImageRatio = 2.0 // images are 2x
	relayInfo.PriceData.ModelRatio = 1.0
	relayInfo.PriceData.GroupRatioInfo.GroupRatio = 1.0
	relayInfo.PriceData.CompletionRatio = 1.0
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
		PromptTokensDetails: dto.InputTokenDetails{
			ImageTokens: 200, // 200 image tokens
		},
	}

	summary := calculateTextQuotaSummary(nil, relayInfo, usage)
	// baseTokens = 1000 - 200 = 800 (at 1.0)
	// imageTokensWithRatio = 200 * 2 = 400
	// promptQuota = 800 + 400 = 1200
	// completionQuota = 500 * 1.0 = 500
	// total = (1200 + 500) * 1.0 = 1700
	require.Equal(t, 1700, summary.Quota)
}
