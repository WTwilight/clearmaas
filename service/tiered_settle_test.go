package service

import (
	"math"
	"math/rand"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Claude Sonnet-style tiered expression: standard vs long-context
const sonnetTieredExpr = `p <= 200000 ? tier("standard", p * 1.5 + c * 7.5) : tier("long_context", p * 3 + c * 11.25)`

// Simple flat expression
const flatExpr = `tier("default", p * 2 + c * 10)`

// Expression with cache tokens
const cacheExpr = `tier("default", p * 2 + c * 10 + cr * 0.2 + cc * 2.5 + cc1h * 4)`

// Expression with request probes
const probeExpr = `param("service_tier") == "fast" ? tier("fast", p * 4 + c * 20) : tier("normal", p * 2 + c * 10)`

const testQuotaPerUnit = 500_000.0

func makeSnapshot(expr string, groupRatio float64, estPrompt, estCompletion int) *billingexpr.BillingSnapshot {
	return &billingexpr.BillingSnapshot{
		BillingMode:               "tiered_expr",
		ExprString:                expr,
		ExprHash:                  billingexpr.ExprHashString(expr),
		GroupRatio:                groupRatio,
		EstimatedPromptTokens:     estPrompt,
		EstimatedCompletionTokens: estCompletion,
		QuotaPerUnit:              testQuotaPerUnit,
	}
}

func makeRelayInfo(expr string, groupRatio float64, estPrompt, estCompletion int) *relaycommon.RelayInfo {
	snap := makeSnapshot(expr, groupRatio, estPrompt, estCompletion)
	cost, trace, _ := billingexpr.RunExpr(expr, billingexpr.TokenParams{P: float64(estPrompt), C: float64(estCompletion)})
	quotaBeforeGroup := cost / 1_000_000 * testQuotaPerUnit
	snap.EstimatedQuotaBeforeGroup = quotaBeforeGroup
	snap.EstimatedQuotaAfterGroup = billingexpr.QuotaRound(quotaBeforeGroup * groupRatio)
	snap.EstimatedTier = trace.MatchedTier
	return &relaycommon.RelayInfo{
		TieredBillingSnapshot: snap,
		FinalPreConsumedQuota: snap.EstimatedQuotaAfterGroup,
	}
}

// ---------------------------------------------------------------------------
// Existing tests (preserved)
// ---------------------------------------------------------------------------

func TestTryTieredSettleUsesFrozenRequestInput(t *testing.T) {
	exprStr := `param("service_tier") == "fast" ? tier("fast", p * 2) : tier("normal", p)`
	relayInfo := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:               "tiered_expr",
			ExprString:                exprStr,
			ExprHash:                  billingexpr.ExprHashString(exprStr),
			GroupRatio:                1.0,
			EstimatedPromptTokens:     100,
			EstimatedCompletionTokens: 0,
			EstimatedQuotaAfterGroup:  50,
			QuotaPerUnit:              testQuotaPerUnit,
		},
		BillingRequestInput: &billingexpr.RequestInput{
			Body: []byte(`{"service_tier":"fast"}`),
		},
	}

	ok, quota, result := TryTieredSettle(relayInfo, billingexpr.TokenParams{P: 100})
	if !ok {
		t.Fatal("expected tiered settle to apply")
	}
	// fast: p*2 = 200; quota = 200 / 1M * 500K = 100
	if quota != 100 {
		t.Fatalf("quota = %d, want 100", quota)
	}
	if result == nil || result.MatchedTier != "fast" {
		t.Fatalf("matched tier = %v, want fast", result)
	}
}

func TestTryTieredSettleFallsBackToFrozenPreConsumeOnExprError(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{
		FinalPreConsumedQuota: 321,
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:              "tiered_expr",
			ExprString:               `invalid +-+ expr`,
			ExprHash:                 billingexpr.ExprHashString(`invalid +-+ expr`),
			GroupRatio:               1.0,
			EstimatedQuotaAfterGroup: 123,
		},
	}

	ok, quota, result := TryTieredSettle(relayInfo, billingexpr.TokenParams{P: 100})
	if !ok {
		t.Fatal("expected tiered settle to apply")
	}
	if quota != 321 {
		t.Fatalf("quota = %d, want 321", quota)
	}
	if result != nil {
		t.Fatalf("result = %#v, want nil", result)
	}
}

// ---------------------------------------------------------------------------
// Pre-consume vs Post-consume consistency
// ---------------------------------------------------------------------------

func TestTryTieredSettle_PreConsumeMatchesPostConsume(t *testing.T) {
	info := makeRelayInfo(flatExpr, 1.0, 1000, 500)
	params := billingexpr.TokenParams{P: 1000, C: 500}

	ok, quota, _ := TryTieredSettle(info, params)
	if !ok {
		t.Fatal("expected tiered settle")
	}
	// p*2 + c*10 = 7000; quota = 7000 / 1M * 500K = 3500
	if quota != 3500 {
		t.Fatalf("quota = %d, want 3500", quota)
	}
	if quota != info.FinalPreConsumedQuota {
		t.Fatalf("pre-consume %d != post-consume %d", info.FinalPreConsumedQuota, quota)
	}
}

func TestTryTieredSettle_PostConsumeOverPreConsume(t *testing.T) {
	info := makeRelayInfo(flatExpr, 1.0, 1000, 500)
	preConsumed := info.FinalPreConsumedQuota // 3500

	// Actual usage is higher than estimated
	params := billingexpr.TokenParams{P: 2000, C: 1000}
	ok, quota, _ := TryTieredSettle(info, params)
	if !ok {
		t.Fatal("expected tiered settle")
	}
	// p*2 + c*10 = 14000; quota = 14000 / 1M * 500K = 7000
	if quota != 7000 {
		t.Fatalf("quota = %d, want 7000", quota)
	}
	if quota <= preConsumed {
		t.Fatalf("expected supplement: actual %d should > pre-consumed %d", quota, preConsumed)
	}
}

func TestTryTieredSettle_PostConsumeUnderPreConsume(t *testing.T) {
	info := makeRelayInfo(flatExpr, 1.0, 1000, 500)
	preConsumed := info.FinalPreConsumedQuota // 3500

	// Actual usage is lower than estimated
	params := billingexpr.TokenParams{P: 100, C: 50}
	ok, quota, _ := TryTieredSettle(info, params)
	if !ok {
		t.Fatal("expected tiered settle")
	}
	// p*2 + c*10 = 700; quota = 700 / 1M * 500K = 350
	if quota != 350 {
		t.Fatalf("quota = %d, want 350", quota)
	}
	if quota >= preConsumed {
		t.Fatalf("expected refund: actual %d should < pre-consumed %d", quota, preConsumed)
	}
}

// ---------------------------------------------------------------------------
// Tiered boundary conditions
// ---------------------------------------------------------------------------

func TestTryTieredSettle_ExactBoundary(t *testing.T) {
	info := makeRelayInfo(sonnetTieredExpr, 1.0, 200000, 1000)

	// p == 200000 => standard tier (p <= 200000)
	ok, quota, result := TryTieredSettle(info, billingexpr.TokenParams{P: 200000, C: 1000})
	if !ok {
		t.Fatal("expected tiered settle")
	}
	// standard: p*1.5 + c*7.5 = 307500; quota = 307500 / 1M * 500K = 153750
	if quota != 153750 {
		t.Fatalf("quota = %d, want 153750", quota)
	}
	if result.MatchedTier != "standard" {
		t.Fatalf("tier = %s, want standard", result.MatchedTier)
	}
}

func TestTryTieredSettle_BoundaryPlusOne(t *testing.T) {
	info := makeRelayInfo(sonnetTieredExpr, 1.0, 200000, 1000)

	// p == 200001 => crosses to long_context tier
	ok, quota, result := TryTieredSettle(info, billingexpr.TokenParams{P: 200001, C: 1000})
	if !ok {
		t.Fatal("expected tiered settle")
	}
	// long_context: p*3 + c*11.25 = 611253; quota = round(611253 / 1M * 500K) = 305627
	if quota != 305627 {
		t.Fatalf("quota = %d, want 305627", quota)
	}
	if result.MatchedTier != "long_context" {
		t.Fatalf("tier = %s, want long_context", result.MatchedTier)
	}
	if !result.CrossedTier {
		t.Fatal("expected CrossedTier = true")
	}
}

func TestTryTieredSettle_ZeroTokens(t *testing.T) {
	info := makeRelayInfo(flatExpr, 1.0, 0, 0)

	ok, quota, result := TryTieredSettle(info, billingexpr.TokenParams{P: 0, C: 0})
	if !ok {
		t.Fatal("expected tiered settle")
	}
	if quota != 0 {
		t.Fatalf("quota = %d, want 0", quota)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

func TestTryTieredSettle_HugeTokens(t *testing.T) {
	info := makeRelayInfo(flatExpr, 1.0, 10000000, 5000000)

	ok, quota, _ := TryTieredSettle(info, billingexpr.TokenParams{P: 10000000, C: 5000000})
	if !ok {
		t.Fatal("expected tiered settle")
	}
	// p*2 + c*10 = 70000000; quota = 70000000 / 1M * 500K = 35000000
	if quota != 35000000 {
		t.Fatalf("quota = %d, want 35000000", quota)
	}
}

func TestTryTieredSettle_CacheTokensAffectSettlement(t *testing.T) {
	info := makeRelayInfo(cacheExpr, 1.0, 1000, 500)

	// Without cache tokens
	ok1, quota1, _ := TryTieredSettle(info, billingexpr.TokenParams{P: 1000, C: 500})
	if !ok1 {
		t.Fatal("expected tiered settle")
	}
	// p*2 + c*10 = 7000; quota = 7000 / 1M * 500K = 3500

	// With cache tokens
	ok2, quota2, _ := TryTieredSettle(info, billingexpr.TokenParams{P: 1000, C: 500, CR: 10000, CC: 5000, CC1h: 2000})
	if !ok2 {
		t.Fatal("expected tiered settle")
	}
	// 2000 + 5000 + 2000 + 12500 + 8000 = 29500; quota = 29500 / 1M * 500K = 14750

	if quota2 <= quota1 {
		t.Fatalf("cache tokens should increase quota: without=%d, with=%d", quota1, quota2)
	}
	if quota1 != 3500 {
		t.Fatalf("no-cache quota = %d, want 3500", quota1)
	}
	if quota2 != 14750 {
		t.Fatalf("cache quota = %d, want 14750", quota2)
	}
}

// ---------------------------------------------------------------------------
// Request probe tests
// ---------------------------------------------------------------------------

func TestTryTieredSettle_RequestProbeInfluencesBilling(t *testing.T) {
	info := makeRelayInfo(probeExpr, 1.0, 1000, 500)
	info.BillingRequestInput = &billingexpr.RequestInput{
		Body: []byte(`{"service_tier":"fast"}`),
	}

	ok, quota, result := TryTieredSettle(info, billingexpr.TokenParams{P: 1000, C: 500})
	if !ok {
		t.Fatal("expected tiered settle")
	}
	// fast: p*4 + c*20 = 14000; quota = 14000 / 1M * 500K = 7000
	if quota != 7000 {
		t.Fatalf("quota = %d, want 7000", quota)
	}
	if result.MatchedTier != "fast" {
		t.Fatalf("tier = %s, want fast", result.MatchedTier)
	}
}

func TestTryTieredSettle_NoRequestInput_FallsBackToDefault(t *testing.T) {
	info := makeRelayInfo(probeExpr, 1.0, 1000, 500)
	// No BillingRequestInput set — param("service_tier") returns nil, not "fast"

	ok, quota, result := TryTieredSettle(info, billingexpr.TokenParams{P: 1000, C: 500})
	if !ok {
		t.Fatal("expected tiered settle")
	}
	// normal: p*2 + c*10 = 7000; quota = 7000 / 1M * 500K = 3500
	if quota != 3500 {
		t.Fatalf("quota = %d, want 3500", quota)
	}
	if result.MatchedTier != "normal" {
		t.Fatalf("tier = %s, want normal", result.MatchedTier)
	}
}

// ---------------------------------------------------------------------------
// Group ratio tests
// ---------------------------------------------------------------------------

func TestTryTieredSettle_GroupRatioScaling(t *testing.T) {
	info := makeRelayInfo(flatExpr, 1.5, 1000, 500)

	ok, quota, _ := TryTieredSettle(info, billingexpr.TokenParams{P: 1000, C: 500})
	if !ok {
		t.Fatal("expected tiered settle")
	}
	// exprCost = 7000, quotaBeforeGroup = 3500, afterGroup = round(3500 * 1.5) = 5250
	if quota != 5250 {
		t.Fatalf("quota = %d, want 5250", quota)
	}
}

func TestTryTieredSettle_GroupRatioZero(t *testing.T) {
	info := makeRelayInfo(flatExpr, 0, 1000, 500)

	ok, quota, _ := TryTieredSettle(info, billingexpr.TokenParams{P: 1000, C: 500})
	if !ok {
		t.Fatal("expected tiered settle")
	}
	if quota != 0 {
		t.Fatalf("quota = %d, want 0 (group ratio = 0)", quota)
	}
}

// ---------------------------------------------------------------------------
// Ratio mode (negative tests) — TryTieredSettle must return false
// ---------------------------------------------------------------------------

func TestTryTieredSettle_RatioMode_NilSnapshot(t *testing.T) {
	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: nil,
	}

	ok, _, _ := TryTieredSettle(info, billingexpr.TokenParams{P: 1000, C: 500})
	if ok {
		t.Fatal("expected TryTieredSettle to return false when snapshot is nil")
	}
}

func TestTryTieredSettle_RatioMode_WrongBillingMode(t *testing.T) {
	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode: "ratio",
			ExprString:  flatExpr,
			ExprHash:    billingexpr.ExprHashString(flatExpr),
			GroupRatio:  1.0,
		},
	}

	ok, _, _ := TryTieredSettle(info, billingexpr.TokenParams{P: 1000, C: 500})
	if ok {
		t.Fatal("expected TryTieredSettle to return false for ratio billing mode")
	}
}

func TestTryTieredSettle_RatioMode_EmptyBillingMode(t *testing.T) {
	info := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode: "",
			ExprString:  flatExpr,
			ExprHash:    billingexpr.ExprHashString(flatExpr),
			GroupRatio:  1.0,
		},
	}

	ok, _, _ := TryTieredSettle(info, billingexpr.TokenParams{P: 1000, C: 500})
	if ok {
		t.Fatal("expected TryTieredSettle to return false for empty billing mode")
	}
}

// ---------------------------------------------------------------------------
// Fallback tests
// ---------------------------------------------------------------------------

func TestTryTieredSettle_ErrorFallbackToEstimatedQuotaAfterGroup(t *testing.T) {
	info := &relaycommon.RelayInfo{
		FinalPreConsumedQuota: 0,
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:              "tiered_expr",
			ExprString:               `invalid expr!!!`,
			ExprHash:                 billingexpr.ExprHashString(`invalid expr!!!`),
			GroupRatio:               1.0,
			EstimatedQuotaAfterGroup: 999,
		},
	}

	ok, quota, result := TryTieredSettle(info, billingexpr.TokenParams{P: 100})
	if !ok {
		t.Fatal("expected tiered settle to apply")
	}
	// FinalPreConsumedQuota is 0, should fall back to EstimatedQuotaAfterGroup
	if quota != 999 {
		t.Fatalf("quota = %d, want 999", quota)
	}
	if result != nil {
		t.Fatal("result should be nil on error fallback")
	}
}

// ---------------------------------------------------------------------------
// BuildTieredTokenParams: token normalization and ratio parity tests
// ---------------------------------------------------------------------------

func tieredQuota(exprStr string, usage *dto.Usage, isClaudeSemantic bool, groupRatio float64) float64 {
	usedVars := billingexpr.UsedVars(exprStr)
	params := BuildTieredTokenParams(usage, isClaudeSemantic, usedVars)
	cost, _, _ := billingexpr.RunExpr(exprStr, params)
	return cost / 1_000_000 * testQuotaPerUnit * groupRatio
}

func ratioQuota(usage *dto.Usage, isClaudeSemantic bool, modelRatio, completionRatio, cacheRatio, imageRatio, groupRatio float64) float64 {
	dPromptTokens := decimal.NewFromInt(int64(usage.PromptTokens))
	dCacheTokens := decimal.NewFromInt(int64(usage.PromptTokensDetails.CachedTokens))
	dCcTokens := decimal.NewFromInt(int64(usage.PromptTokensDetails.CachedCreationTokens))
	dImgTokens := decimal.NewFromInt(int64(usage.PromptTokensDetails.ImageTokens))
	dCompletionTokens := decimal.NewFromInt(int64(usage.CompletionTokens))
	dModelRatio := decimal.NewFromFloat(modelRatio)
	dCompletionRatio := decimal.NewFromFloat(completionRatio)
	dCacheRatio := decimal.NewFromFloat(cacheRatio)
	dImageRatio := decimal.NewFromFloat(imageRatio)
	dGroupRatio := decimal.NewFromFloat(groupRatio)

	baseTokens := dPromptTokens
	if !isClaudeSemantic {
		baseTokens = baseTokens.Sub(dCacheTokens)
		baseTokens = baseTokens.Sub(dCcTokens)
		baseTokens = baseTokens.Sub(dImgTokens)
	}

	cachedTokensWithRatio := dCacheTokens.Mul(dCacheRatio)
	imageTokensWithRatio := dImgTokens.Mul(dImageRatio)
	promptQuota := baseTokens.Add(cachedTokensWithRatio).Add(imageTokensWithRatio)
	completionQuota := dCompletionTokens.Mul(dCompletionRatio)
	ratio := dModelRatio.Mul(dGroupRatio)

	result := promptQuota.Add(completionQuota).Mul(ratio)
	f, _ := result.Float64()
	return f
}

func TestBuildTieredTokenParams_GPT_WithCache(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 200,
			TextTokens:   800,
		},
	}
	expr := `tier("base", p * 2.5 + c * 15 + cr * 0.25)`
	got := tieredQuota(expr, usage, false, 1.0)
	// P=800, C=500, CR=200 → (800*2.5 + 500*15 + 200*0.25) * 0.5 = 4775
	want := 4775.0
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("quota = %f, want %f", got, want)
	}
}

func TestBuildTieredTokenParams_GPT_NoCacheVar(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 200,
			TextTokens:   800,
		},
	}
	expr := `tier("base", p * 2.5 + c * 15)`
	got := tieredQuota(expr, usage, false, 1.0)
	// No cr → P=1000 (cache stays in P), C=500 → (1000*2.5 + 500*15) * 0.5 = 5000
	want := 5000.0
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("quota = %f, want %f", got, want)
	}
}

func TestBuildTieredTokenParams_GPT_WithImage(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		PromptTokensDetails: dto.InputTokenDetails{
			ImageTokens: 200,
			TextTokens:  800,
		},
	}
	expr := `tier("base", p * 2 + c * 8 + img * 2.5)`
	got := tieredQuota(expr, usage, false, 1.0)
	// P=800, C=500, Img=200 → (800*2 + 500*8 + 200*2.5) * 0.5 = 3050
	want := 3050.0
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("quota = %f, want %f", got, want)
	}
}

func TestBuildTieredTokenParams_Claude_WithCache(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     800,
		CompletionTokens: 500,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 200,
			TextTokens:   800,
		},
	}
	expr := `tier("base", p * 3 + c * 15 + cr * 0.3)`
	got := tieredQuota(expr, usage, true, 1.0)
	// Claude: P=800 (no subtraction), C=500, CR=200 → (800*3 + 500*15 + 200*0.3) * 0.5 = 4980
	want := 4980.0
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("quota = %f, want %f", got, want)
	}
}

func TestBuildTieredTokenParams_GPT_AudioOutput(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 600,
		CompletionTokenDetails: dto.OutputTokenDetails{
			AudioTokens: 100,
			TextTokens:  500,
		},
	}
	expr := `tier("base", p * 2 + c * 10 + ao * 50)`
	got := tieredQuota(expr, usage, false, 1.0)
	// C=600-100=500, AO=100 → (1000*2 + 500*10 + 100*50) * 0.5 = 6000
	want := 6000.0
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("quota = %f, want %f", got, want)
	}
}

func TestBuildTieredTokenParams_GPT_AudioOutputNoVar(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 600,
		CompletionTokenDetails: dto.OutputTokenDetails{
			AudioTokens: 100,
			TextTokens:  500,
		},
	}
	expr := `tier("base", p * 2 + c * 10)`
	got := tieredQuota(expr, usage, false, 1.0)
	// No ao → C=600 (audio stays in C) → (1000*2 + 600*10) * 0.5 = 4000
	want := 4000.0
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("quota = %f, want %f", got, want)
	}
}

func TestBuildTieredTokenParams_ParityWithRatio(t *testing.T) {
	// GPT-5.4 prices: input=$2.5, output=$15, cacheRead=$0.25
	// Ratio equivalents: modelRatio=1.25, completionRatio=6, cacheRatio=0.1
	usage := &dto.Usage{
		PromptTokens:     10000,
		CompletionTokens: 2000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 3000,
			TextTokens:   7000,
		},
	}
	expr := `tier("base", p * 2.5 + c * 15 + cr * 0.25)`

	for _, gr := range []float64{1.0, 1.5, 2.0, 0.5} {
		tq := tieredQuota(expr, usage, false, gr)
		rq := ratioQuota(usage, false, 1.25, 6, 0.1, 0, gr)

		if math.Abs(tq-rq) > 0.01 {
			t.Fatalf("groupRatio=%v: tiered=%f ratio=%f (mismatch)", gr, tq, rq)
		}
	}
}

func TestBuildTieredTokenParams_ParityWithRatio_Image(t *testing.T) {
	// gpt-image-1-mini prices: input=$2, output=$8, image=$2.5
	// Ratio equivalents: modelRatio=1, completionRatio=4, imageRatio=1.25
	usage := &dto.Usage{
		PromptTokens:     5000,
		CompletionTokens: 4000,
		PromptTokensDetails: dto.InputTokenDetails{
			ImageTokens: 1000,
			TextTokens:  4000,
		},
	}
	expr := `tier("base", p * 2 + c * 8 + img * 2.5)`

	tq := tieredQuota(expr, usage, false, 1.0)
	rq := ratioQuota(usage, false, 1.0, 4, 0, 1.25, 1.0)

	if math.Abs(tq-rq) > 0.01 {
		t.Fatalf("tiered=%f ratio=%f (mismatch)", tq, rq)
	}
}

// ---------------------------------------------------------------------------
// BuildTieredTokenParams: Len computation tests
// ---------------------------------------------------------------------------

func TestBuildTieredTokenParams_Len_GPT(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     10000,
		CompletionTokens: 2000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 3000,
			TextTokens:   7000,
		},
	}
	expr := `tier("base", p * 2.5 + c * 15 + cr * 0.25)`
	usedVars := billingexpr.UsedVars(expr)
	params := BuildTieredTokenParams(usage, false, usedVars)

	// Non-Claude: Len = raw PromptTokens
	if params.Len != 10000 {
		t.Fatalf("Len = %f, want 10000 (raw PromptTokens)", params.Len)
	}
	// P should be reduced by cache
	if params.P != 7000 {
		t.Fatalf("P = %f, want 7000 (PromptTokens - CachedTokens)", params.P)
	}
}

func TestBuildTieredTokenParams_Len_Claude(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     5000,
		CompletionTokens: 2000,
		UsageSemantic:    "anthropic",
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 3000,
			TextTokens:   5000,
		},
		ClaudeCacheCreation5mTokens: 1000,
		ClaudeCacheCreation1hTokens: 500,
	}
	expr := `tier("base", p * 3 + c * 15 + cr * 0.3 + cc * 3.75 + cc1h * 6)`
	usedVars := billingexpr.UsedVars(expr)
	params := BuildTieredTokenParams(usage, true, usedVars)

	// Claude: Len = PromptTokens + CachedTokens + CacheCreation5m + CacheCreation1h
	wantLen := float64(5000 + 3000 + 1000 + 500)
	if params.Len != wantLen {
		t.Fatalf("Len = %f, want %f (text + cache read + cache creation)", params.Len, wantLen)
	}
	// Claude: P is not reduced (isClaudeUsageSemantic = true)
	if params.P != 5000 {
		t.Fatalf("P = %f, want 5000 (no subtraction for Claude)", params.P)
	}
}

func TestBuildTieredTokenParams_Len_TierCondition(t *testing.T) {
	// Test that len-based tier conditions work correctly when p is reduced by cache
	usage := &dto.Usage{
		PromptTokens:     300000,
		CompletionTokens: 5000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 250000,
			TextTokens:   50000,
		},
	}
	expr := `len <= 200000 ? tier("standard", p * 3 + c * 15 + cr * 0.3) : tier("long_context", p * 6 + c * 22.5 + cr * 0.6)`
	usedVars := billingexpr.UsedVars(expr)
	params := BuildTieredTokenParams(usage, false, usedVars)

	// Len = 300000 (raw prompt), P = 50000 (300000 - 250000 cache)
	if params.Len != 300000 {
		t.Fatalf("Len = %f, want 300000", params.Len)
	}
	if params.P != 50000 {
		t.Fatalf("P = %f, want 50000", params.P)
	}

	// Run expression: len=300000 > 200000, so long_context tier
	cost, trace, err := billingexpr.RunExpr(expr, params)
	if err != nil {
		t.Fatal(err)
	}
	if trace.MatchedTier != "long_context" {
		t.Fatalf("tier = %s, want long_context (len=300000 but p=50000)", trace.MatchedTier)
	}
	// long_context: 50000*6 + 5000*22.5 + 250000*0.6
	wantCost := 50000.0*6 + 5000*22.5 + 250000*0.6
	if math.Abs(cost-wantCost) > 1e-6 {
		t.Fatalf("cost = %f, want %f", cost, wantCost)
	}
}

// ---------------------------------------------------------------------------
// Stress test: 1000 concurrent goroutines, complex tiered expr vs ratio,
// random token counts, verify correctness and measure performance
// ---------------------------------------------------------------------------

const complexTieredExpr = `p <= 200000 ? tier("standard", p * 3 + c * 15 + cr * 0.3 + cc * 3.75 + cc1h * 6 + img * 3 + img_o * 30 + ai * 10 + ao * 40) : tier("long_context", p * 6 + c * 22.5 + cr * 0.6 + cc * 7.5 + cc1h * 12 + img * 6 + img_o * 60 + ai * 20 + ao * 80)`

func randomUsage(rng *rand.Rand) *dto.Usage {
	cacheRead := int(rng.Float64() * 50000)
	cacheCreate := int(rng.Float64() * 10000)
	imgIn := int(rng.Float64() * 5000)
	audioIn := int(rng.Float64() * 3000)
	prompt := int(rng.Float64()*300000) + cacheRead + cacheCreate + imgIn + audioIn

	imgOut := int(rng.Float64() * 2000)
	audioOut := int(rng.Float64() * 1000)
	completion := int(rng.Float64()*50000) + imgOut + audioOut

	return &dto.Usage{
		PromptTokens:     prompt,
		CompletionTokens: completion,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         cacheRead,
			CachedCreationTokens: cacheCreate,
			ImageTokens:          imgIn,
			AudioTokens:          audioIn,
			TextTokens:           prompt - cacheRead - cacheCreate - imgIn - audioIn,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{
			ImageTokens: imgOut,
			AudioTokens: audioOut,
			TextTokens:  completion - imgOut - audioOut,
		},
	}
}

func TestStress_TieredBilling_1000Concurrent(t *testing.T) {
	usedVars := billingexpr.UsedVars(complexTieredExpr)

	var wg sync.WaitGroup
	errCh := make(chan string, 1000)

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(seed))

			for j := 0; j < 100; j++ {
				usage := randomUsage(rng)
				groupRatio := 0.5 + rng.Float64()*2.0

				params := BuildTieredTokenParams(usage, false, usedVars)
				cost, trace, err := billingexpr.RunExpr(complexTieredExpr, params)
				if err != nil {
					errCh <- err.Error()
					return
				}
				if cost < 0 {
					errCh <- "negative cost"
					return
				}

				quota := billingexpr.QuotaRound(cost / 1_000_000 * testQuotaPerUnit * groupRatio)
				if quota < 0 {
					errCh <- "negative quota"
					return
				}

				_ = trace.MatchedTier
			}
		}(int64(i))
	}

	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Fatal(e)
	}
}

func BenchmarkTieredBilling_ComplexExpr(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	usedVars := billingexpr.UsedVars(complexTieredExpr)
	usages := make([]*dto.Usage, 1000)
	for i := range usages {
		usages[i] = randomUsage(rng)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage := usages[i%len(usages)]
		params := BuildTieredTokenParams(usage, false, usedVars)
		billingexpr.RunExpr(complexTieredExpr, params)
	}
}

func BenchmarkRatioBilling_Equivalent(b *testing.B) {
	rng := rand.New(rand.NewSource(42))
	usages := make([]*dto.Usage, 1000)
	for i := range usages {
		usages[i] = randomUsage(rng)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage := usages[i%len(usages)]
		ratioQuota(usage, false, 1.5, 5.0, 0.1, 1.0, 1.5)
	}
}

func BenchmarkTieredBilling_Parallel(b *testing.B) {
	usedVars := billingexpr.UsedVars(complexTieredExpr)

	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(rand.Int63()))
		for pb.Next() {
			usage := randomUsage(rng)
			params := BuildTieredTokenParams(usage, false, usedVars)
			billingexpr.RunExpr(complexTieredExpr, params)
		}
	})
}

func BenchmarkRatioBilling_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		rng := rand.New(rand.NewSource(rand.Int63()))
		for pb.Next() {
			usage := randomUsage(rng)
			ratioQuota(usage, false, 1.5, 5.0, 0.1, 1.0, 1.5)
		}
	})
}

// ---------------------------------------------------------------------------
// BuildTieredTokenParams: Claude semantic cache split
// ---------------------------------------------------------------------------

func TestBuildTieredTokenParams_ClaudeSemantic_CC5mOnly(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:                   100000,
		CompletionTokens:               5000,
		UsageSemantic:                  "anthropic",
		ClaudeCacheCreation5mTokens:    50000,
		ClaudeCacheCreation1hTokens:    0,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         10000,
			CachedCreationTokens: 0,
			ImageTokens:          0,
			AudioTokens:          0,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{},
	}
	// usedVars does NOT include "cr", "cc", "cc1h", etc.
	usedVars := map[string]bool{}

	params := BuildTieredTokenParams(usage, true, usedVars)

	// Claude semantic: cc1h from ClaudeCacheCreation1hTokens = 0
	// cc5m from ClaudeCacheCreation5mTokens = 50000
	require.Equal(t, float64(100000), params.P)
	require.Equal(t, float64(5000), params.C)
	// len = p + cr + cc5m + cc1h = 100000 + 10000 + 50000 + 0 = 160000
	require.Equal(t, float64(160000), params.Len)
	require.Equal(t, float64(10000), params.CR)
	require.Equal(t, float64(50000), params.CC)
	require.Equal(t, float64(0), params.CC1h)
}

func TestBuildTieredTokenParams_ClaudeSemantic_CC5mAndCC1h(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:                   100000,
		CompletionTokens:               5000,
		UsageSemantic:                  "anthropic",
		ClaudeCacheCreation5mTokens:    50000,
		ClaudeCacheCreation1hTokens:    20000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         10000,
			CachedCreationTokens: 0,
			ImageTokens:          0,
			AudioTokens:          0,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{},
	}
	usedVars := map[string]bool{}

	params := BuildTieredTokenParams(usage, true, usedVars)

	// len = p + cr + cc5m + cc1h = 100000 + 10000 + 50000 + 20000 = 180000
	require.Equal(t, float64(180000), params.Len)
	require.Equal(t, float64(50000), params.CC)
	require.Equal(t, float64(20000), params.CC1h)
}

func TestBuildTieredTokenParams_ClaudeSemantic_BothZero(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:                   100000,
		CompletionTokens:               5000,
		UsageSemantic:                  "anthropic",
		ClaudeCacheCreation5mTokens:    0,
		ClaudeCacheCreation1hTokens:    0,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         0,
			CachedCreationTokens: 0,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{},
	}
	usedVars := map[string]bool{}

	params := BuildTieredTokenParams(usage, true, usedVars)

	require.Equal(t, float64(100000), params.Len)
	require.Equal(t, float64(0), params.CC)
	require.Equal(t, float64(0), params.CC1h)
}

func TestBuildTieredTokenParams_NonClaude_CacheReferenced(t *testing.T) {
	// Non-Claude but usedVars includes cr/cc → p should subtract cache tokens
	usage := &dto.Usage{
		PromptTokens:                   100000,
		CompletionTokens:               5000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         20000,
			CachedCreationTokens: 10000,
			ImageTokens:          0,
			AudioTokens:          0,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{},
	}
	usedVars := map[string]bool{"cr": true, "cc": true}

	params := BuildTieredTokenParams(usage, false, usedVars)

	// p -= cr + cc = 100000 - 20000 - 10000 = 70000
	require.Equal(t, float64(70000), params.P)
	require.Equal(t, float64(20000), params.CR)
	require.Equal(t, float64(10000), params.CC)
}

func TestBuildTieredTokenParams_NonClaude_CacheNotReferenced(t *testing.T) {
	// Non-Claude and usedVars does NOT include cr/cc → p unchanged
	usage := &dto.Usage{
		PromptTokens:                   100000,
		CompletionTokens:               5000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         20000,
			CachedCreationTokens: 10000,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{},
	}
	usedVars := map[string]bool{}

	params := BuildTieredTokenParams(usage, false, usedVars)

	// p unchanged = 100000 (cache not referenced in expression)
	require.Equal(t, float64(100000), params.P)
	require.Equal(t, float64(20000), params.CR)
	require.Equal(t, float64(10000), params.CC)
}

func TestBuildTieredTokenParams_NonClaude_ImageReferenced(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:                   10000,
		CompletionTokens:               5000,
		PromptTokensDetails: dto.InputTokenDetails{
			ImageTokens: 2000,
			AudioTokens: 0,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{},
	}
	usedVars := map[string]bool{"img": true, "ai": true}

	params := BuildTieredTokenParams(usage, false, usedVars)

	// p -= img = 10000 - 2000 = 8000
	require.Equal(t, float64(8000), params.P)
	require.Equal(t, float64(2000), params.Img)
}

func TestBuildTieredTokenParams_NonClaude_ImageOutputReferenced(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:                   1000,
		CompletionTokens:               5000,
		PromptTokensDetails: dto.InputTokenDetails{
			ImageTokens: 0,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{
			ImageTokens: 3000,
			AudioTokens: 0,
		},
	}
	usedVars := map[string]bool{"img_o": true, "ao": true}

	params := BuildTieredTokenParams(usage, false, usedVars)

	// c -= img_o = 5000 - 3000 = 2000
	require.Equal(t, float64(2000), params.C)
	require.Equal(t, float64(3000), params.ImgO)
}

func TestBuildTieredTokenParams_NegativeClamping(t *testing.T) {
	// When p or c go negative after subtraction, they should be clamped to 0
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		PromptTokensDetails: dto.InputTokenDetails{
			// cr > promptTokens
			CachedTokens: 2000,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{},
	}
	usedVars := map[string]bool{"cr": true}

	params := BuildTieredTokenParams(usage, false, usedVars)

	// p = 1000 - 2000 = -1000 → clamped to 0
	require.Equal(t, float64(0), params.P)
}

func TestBuildTieredTokenParams_NegativeCompletionClamping(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     1000,
		CompletionTokens: 500,
		PromptTokensDetails: dto.InputTokenDetails{
			ImageTokens: 0,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{
			// img_o > completionTokens
			ImageTokens: 1000,
		},
	}
	usedVars := map[string]bool{"img_o": true}

	params := BuildTieredTokenParams(usage, false, usedVars)

	// c = 500 - 1000 = -500 → clamped to 0
	require.Equal(t, float64(0), params.C)
}

func TestBuildTieredTokenParams_ClaudeLenIncludesCacheTokens(t *testing.T) {
	// For Claude: len = p + cr + cc5m + cc1h (Claude uses separate counters)
	usage := &dto.Usage{
		PromptTokens:                   80000,
		CompletionTokens:               5000,
		UsageSemantic:                  "anthropic",
		ClaudeCacheCreation5mTokens:    50000,
		ClaudeCacheCreation1hTokens:    0,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 10000,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{},
	}
	usedVars := map[string]bool{}

	params := BuildTieredTokenParams(usage, true, usedVars)

	// len = 80000 + 10000 + 50000 + 0 = 140000
	require.Equal(t, float64(140000), params.Len)
}

func TestBuildTieredTokenParams_NonClaudeLenEqualsPrompt(t *testing.T) {
	// For non-Claude: len = p (promptTokens already includes everything)
	usage := &dto.Usage{
		PromptTokens:                   100000,
		CompletionTokens:               5000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens: 20000,
		},
		CompletionTokenDetails: dto.OutputTokenDetails{},
	}
	usedVars := map[string]bool{}

	params := BuildTieredTokenParams(usage, false, usedVars)

	// len = p = 100000 (not 100000 + 20000)
	require.Equal(t, float64(100000), params.Len)
}

// ---------------------------------------------------------------------------
// TryTieredSettle: additional branch coverage
// ---------------------------------------------------------------------------

func TestTryTieredSettle_SnapshotNil(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{
		TieredBillingSnapshot: nil,
	}
	ok, quota, result := TryTieredSettle(relayInfo, billingexpr.TokenParams{P: 100})
	require.False(t, ok)
	require.Equal(t, 0, quota)
	require.Nil(t, result)
}

func TestTryTieredSettle_WrongBillingMode(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode: "ratio", // not "tiered_expr"
		},
	}
	ok, quota, result := TryTieredSettle(relayInfo, billingexpr.TokenParams{P: 100})
	require.False(t, ok)
	require.Equal(t, 0, quota)
	require.Nil(t, result)
}

func TestTryTieredSettle_ValidCrossTier(t *testing.T) {
	expr := `p <= 200000 ? tier("standard", p * 1.5 + c * 7.5) : tier("long_context", p * 3.0 + c * 11.25)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:               "tiered_expr",
		ExprString:                expr,
		ExprHash:                  billingexpr.ExprHashString(expr),
		GroupRatio:                1.0,
		EstimatedPromptTokens:     100000,
		EstimatedCompletionTokens: 5000,
		EstimatedQuotaBeforeGroup: (100000*1.5 + 5000*7.5) / 1_000_000 * 500_000,
		EstimatedQuotaAfterGroup: billingexpr.QuotaRound((100000*1.5 + 5000*7.5) / 1_000_000 * 500_000),
		EstimatedTier:             "standard", // pre-consumed under standard tier
		QuotaPerUnit:              500_000,
	}
	relayInfo := &relaycommon.RelayInfo{
		TieredBillingSnapshot: snap,
	}

	// Actual: long context (p > 200000)
	ok, quota, result := TryTieredSettle(relayInfo, billingexpr.TokenParams{P: 300000, C: 10000})
	require.True(t, ok)
	require.NotNil(t, result)
	require.True(t, result.CrossedTier)
	require.Equal(t, "long_context", result.MatchedTier)
	// long_context: 300000*3.0 + 10000*11.25 = 1012500 / 1M * 500K = 506.25 → round = 506
	require.Equal(t, 506, quota)
}

func TestTryTieredSettle_ValidSameTier(t *testing.T) {
	expr := `tier("default", p * 1 + c * 2)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:              "tiered_expr",
		ExprString:               expr,
		ExprHash:                 billingexpr.ExprHashString(expr),
		GroupRatio:               1.0,
		EstimatedTier:            "default",
		QuotaPerUnit:             500_000,
	}
	relayInfo := &relaycommon.RelayInfo{
		TieredBillingSnapshot: snap,
	}

	ok, quota, result := TryTieredSettle(relayInfo, billingexpr.TokenParams{P: 1000, C: 500})
	require.True(t, ok)
	require.NotNil(t, result)
	require.False(t, result.CrossedTier)
	require.Equal(t, "default", result.MatchedTier)
	// cost = 1000*1 + 500*2 = 2000; quota = 2000 / 1M * 500K = 1
	require.Equal(t, 1, quota)
}

func TestTryTieredSettle_ExprErrorWithZeroFallback(t *testing.T) {
	relayInfo := &relaycommon.RelayInfo{
		FinalPreConsumedQuota: 0, // zero fallback
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{
			BillingMode:               "tiered_expr",
			ExprString:                `invalid +++ expr`,
			ExprHash:                  billingexpr.ExprHashString(`invalid +++ expr`),
			GroupRatio:               1.0,
			EstimatedQuotaAfterGroup: 999,
		},
	}

	ok, quota, result := TryTieredSettle(relayInfo, billingexpr.TokenParams{P: 100})
	require.True(t, ok)
	// quota <= 0 → fallback to EstimatedQuotaAfterGroup = 999
	require.Equal(t, 999, quota)
	require.Nil(t, result)
}

func TestTryTieredSettle_UsesFrozenRequestInput(t *testing.T) {
	// Verify that TryTieredSettle uses relayInfo.BillingRequestInput, not the live request
	expr := `param("mode") == "premium" ? tier("premium", p * 2) : tier("normal", p)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:              "tiered_expr",
		ExprString:               expr,
		ExprHash:                 billingexpr.ExprHashString(expr),
		GroupRatio:               1.0,
		EstimatedTier:            "normal",
		QuotaPerUnit:             500_000,
	}
	relayInfo := &relaycommon.RelayInfo{
		TieredBillingSnapshot: snap,
		BillingRequestInput: &billingexpr.RequestInput{
			Body: []byte(`{"mode":"premium"}`),
		},
	}

	ok, quota, result := TryTieredSettle(relayInfo, billingexpr.TokenParams{P: 1000})
	require.True(t, ok)
	require.NotNil(t, result)
	require.Equal(t, "premium", result.MatchedTier)
	// premium: p*2 = 2000; quota = 2000 / 1M * 500K = 1
	require.Equal(t, 1, quota)
}

func TestTryTieredSettle_NoRequestInput(t *testing.T) {
	// When BillingRequestInput is nil, should not panic
	expr := `tier("default", p)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   expr,
		ExprHash:     billingexpr.ExprHashString(expr),
		GroupRatio:   1.0,
		EstimatedTier: "default",
		QuotaPerUnit:  500_000,
	}
	relayInfo := &relaycommon.RelayInfo{
		TieredBillingSnapshot:   snap,
		BillingRequestInput:    nil,
	}

	ok, quota, result := TryTieredSettle(relayInfo, billingexpr.TokenParams{P: 1000})
	require.True(t, ok)
	require.NotNil(t, result)
	require.Equal(t, "default", result.MatchedTier)
	require.Equal(t, 1, quota) // 1000 / 1M * 500K = 0.5 → round = 1
}
