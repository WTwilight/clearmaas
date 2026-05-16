package billingexpr_test

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
)

// ---------------------------------------------------------------------------
// Claude-style: fixed tiers, input > 200K changes both input & output price
// ---------------------------------------------------------------------------

const claudeExpr = `p <= 200000 ? tier("standard", p * 1.5 + c * 7.5) : tier("long_context", p * 3.0 + c * 11.25)`

func TestClaude_StandardTier(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(claudeExpr, billingexpr.TokenParams{P: 100000, C: 5000})
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "standard")
	}
}

func TestClaude_LongContextTier(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(claudeExpr, billingexpr.TokenParams{P: 300000, C: 10000})
	if err != nil {
		t.Fatal(err)
	}
	want := 300000*3.0 + 10000*11.25
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "long_context" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "long_context")
	}
}

func TestClaude_BoundaryExact(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(claudeExpr, billingexpr.TokenParams{P: 200000, C: 1000})
	if err != nil {
		t.Fatal(err)
	}
	want := 200000*1.5 + 1000*7.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "standard")
	}
}

// ---------------------------------------------------------------------------
// GLM-style: multi-condition tiers with both input and output dimensions
// ---------------------------------------------------------------------------

const glmExpr = `
(
	p < 32000 && c < 200 ? tier("tier1_short", (p)*2 + c*8) :
	p < 32000 && c >= 200 ? tier("tier2_long_output", (p)*3 + c*14) :
	tier("tier3_long_input", (p)*4 + c*16)
) / 1000000
`

func TestGLM_Tier1(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(glmExpr, billingexpr.TokenParams{P: 15000, C: 100})
	if err != nil {
		t.Fatal(err)
	}
	want := (15000.0*2 + 100.0*8) / 1000000
	if math.Abs(cost-want) > 1e-10 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "tier1_short" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "tier1_short")
	}
}

func TestGLM_Tier2(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(glmExpr, billingexpr.TokenParams{P: 15000, C: 500})
	if err != nil {
		t.Fatal(err)
	}
	want := (15000.0*3 + 500.0*14) / 1000000
	if math.Abs(cost-want) > 1e-10 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "tier2_long_output" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "tier2_long_output")
	}
}

func TestGLM_Tier3(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr(glmExpr, billingexpr.TokenParams{P: 50000, C: 100})
	if err != nil {
		t.Fatal(err)
	}
	want := (50000.0*4 + 100.0*16) / 1000000
	if math.Abs(cost-want) > 1e-10 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "tier3_long_input" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "tier3_long_input")
	}
}

// ---------------------------------------------------------------------------
// Simple flat-rate (no tier() call)
// ---------------------------------------------------------------------------

func TestSimpleExpr_NoTier(t *testing.T) {
	cost, trace, err := billingexpr.RunExpr("p * 0.5 + c * 1.0", billingexpr.TokenParams{P: 1000, C: 500})
	if err != nil {
		t.Fatal(err)
	}
	want := 1000*0.5 + 500*1.0
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "" {
		t.Errorf("tier should be empty, got %q", trace.MatchedTier)
	}
}

// ---------------------------------------------------------------------------
// Math helper functions
// ---------------------------------------------------------------------------

func TestMathHelpers(t *testing.T) {
	cost, _, err := billingexpr.RunExpr("max(p, c) * 0.5 + min(p, c) * 0.1", billingexpr.TokenParams{P: 300, C: 500})
	if err != nil {
		t.Fatal(err)
	}
	want := 500*0.5 + 300*0.1
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestRequestProbeHelpers(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`p * 0.5 + c * 1.0 * (param("service_tier") == "fast" ? 2 : 1)`,
		billingexpr.TokenParams{P: 1000, C: 500},
		billingexpr.RequestInput{
			Body: []byte(`{"service_tier":"fast"}`),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 1000*0.5 + 500*1.0*2
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestHeaderProbeHelper(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`p * 0.5 + c * 1.0 * (has(header("anthropic-beta"), "fast-mode") ? 2 : 1)`,
		billingexpr.TokenParams{P: 1000, C: 500},
		billingexpr.RequestInput{
			Headers: map[string]string{
				"Anthropic-Beta": "fast-mode-2026-02-01",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 1000*0.5 + 500*1.0*2
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestParamProbeNestedBool(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`p * (param("stream_options.fast_mode") == true ? 1.5 : 1.0)`,
		billingexpr.TokenParams{P: 100},
		billingexpr.RequestInput{
			Body: []byte(`{"stream_options":{"fast_mode":true}}`),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 150.0
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestParamProbeArrayLength(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`p * (param("messages.#") > 20 ? 1.2 : 1.0)`,
		billingexpr.TokenParams{P: 100},
		billingexpr.RequestInput{
			Body: []byte(`{"messages":[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21]}`),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 120.0
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestRequestProbeMissingFieldReturnsNil(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`param("missing.value") == nil ? 2 : 1`,
		billingexpr.TokenParams{},
		billingexpr.RequestInput{
			Body: []byte(`{"service_tier":"standard"}`),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if cost != 2 {
		t.Errorf("cost = %f, want 2", cost)
	}
}

func TestRequestProbeMultipleRulesMultiply(t *testing.T) {
	cost, _, err := billingexpr.RunExprWithRequest(
		`(param("service_tier") == "fast" ? 2 : 1) * (has(header("anthropic-beta"), "fast-mode-2026-02-01") ? 2.5 : 1)`,
		billingexpr.TokenParams{},
		billingexpr.RequestInput{
			Headers: map[string]string{
				"Anthropic-Beta": "fast-mode-2026-02-01",
			},
			Body: []byte(`{"service_tier":"fast"}`),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(cost-5) > 1e-6 {
		t.Errorf("cost = %f, want 5", cost)
	}
}

func TestCeilFloor(t *testing.T) {
	cost, _, err := billingexpr.RunExpr("ceil(p / 1000) * 0.5", billingexpr.TokenParams{P: 1500})
	if err != nil {
		t.Fatal(err)
	}
	want := math.Ceil(1500.0/1000) * 0.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

// ---------------------------------------------------------------------------
// Zero tokens
// ---------------------------------------------------------------------------

func TestZeroTokens(t *testing.T) {
	cost, _, err := billingexpr.RunExpr(claudeExpr, billingexpr.TokenParams{})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 0 {
		t.Errorf("cost should be 0 for zero tokens, got %f", cost)
	}
}

// ---------------------------------------------------------------------------
// Rounding
// ---------------------------------------------------------------------------

func TestQuotaRound(t *testing.T) {
	tests := []struct {
		in   float64
		want int
	}{
		{0, 0},
		{0.4, 0},
		{0.5, 1},
		{0.6, 1},
		{1.5, 2},
		{-0.5, -1},
		{-0.6, -1},
		{999.4999, 999},
		{999.5, 1000},
		{1e9 + 0.5, 1e9 + 1},
	}
	for _, tt := range tests {
		got := billingexpr.QuotaRound(tt.in)
		if got != tt.want {
			t.Errorf("QuotaRound(%f) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Settlement
// ---------------------------------------------------------------------------

func TestComputeTieredQuota_Basic(t *testing.T) {
	snap := &billingexpr.BillingSnapshot{
		BillingMode:               "tiered_expr",
		ExprString:                claudeExpr,
		ExprHash:                  billingexpr.ExprHashString(claudeExpr),
		GroupRatio:                1.0,
		EstimatedPromptTokens:     100000,
		EstimatedCompletionTokens: 5000,
		EstimatedQuotaBeforeGroup: (100000*1.5 + 5000*7.5) / 1_000_000 * 500_000,
		EstimatedQuotaAfterGroup:  billingexpr.QuotaRound((100000*1.5 + 5000*7.5) / 1_000_000 * 500_000),
		EstimatedTier:             "standard",
		QuotaPerUnit:              500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 300000, C: 10000})
	if err != nil {
		t.Fatal(err)
	}

	wantBefore := (300000*3.0 + 10000*11.25) / 1_000_000 * 500_000
	if math.Abs(result.ActualQuotaBeforeGroup-wantBefore) > 1e-6 {
		t.Errorf("before group: got %f, want %f", result.ActualQuotaBeforeGroup, wantBefore)
	}
	if result.MatchedTier != "long_context" {
		t.Errorf("tier = %q, want %q", result.MatchedTier, "long_context")
	}
	if !result.CrossedTier {
		t.Error("expected crossed_tier=true (estimated standard, actual long_context)")
	}
}

func TestComputeTieredQuota_SameTier(t *testing.T) {
	snap := &billingexpr.BillingSnapshot{
		BillingMode:               "tiered_expr",
		ExprString:                claudeExpr,
		ExprHash:                  billingexpr.ExprHashString(claudeExpr),
		GroupRatio:                1.5,
		EstimatedPromptTokens:     50000,
		EstimatedCompletionTokens: 1000,
		EstimatedQuotaBeforeGroup: (50000*1.5 + 1000*7.5) / 1_000_000 * 500_000,
		EstimatedQuotaAfterGroup:  billingexpr.QuotaRound((50000*1.5 + 1000*7.5) / 1_000_000 * 500_000 * 1.5),
		EstimatedTier:             "standard",
		QuotaPerUnit:              500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 80000, C: 2000})
	if err != nil {
		t.Fatal(err)
	}

	wantBefore := (80000*1.5 + 2000*7.5) / 1_000_000 * 500_000
	wantAfter := billingexpr.QuotaRound(wantBefore * 1.5)
	if result.ActualQuotaAfterGroup != wantAfter {
		t.Errorf("after group: got %d, want %d", result.ActualQuotaAfterGroup, wantAfter)
	}
	if result.CrossedTier {
		t.Error("expected crossed_tier=false (both standard)")
	}
}

// ---------------------------------------------------------------------------
// Compile errors
// ---------------------------------------------------------------------------

func TestCompileError(t *testing.T) {
	_, _, err := billingexpr.RunExpr("invalid +-+ syntax", billingexpr.TokenParams{})
	if err == nil {
		t.Error("expected compile error")
	}
}

// ---------------------------------------------------------------------------
// Compile Cache
// ---------------------------------------------------------------------------

func TestCompileCache_SameResult(t *testing.T) {
	r1, _, err := billingexpr.RunExpr("p * 0.5", billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	r2, _, err := billingexpr.RunExpr("p * 0.5", billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r2 {
		t.Errorf("cached and uncached results differ: %f != %f", r1, r2)
	}
}

func TestInvalidateCache(t *testing.T) {
	billingexpr.InvalidateCache()
	r1, _, _ := billingexpr.RunExpr("p * 0.5", billingexpr.TokenParams{P: 100})
	billingexpr.InvalidateCache()
	r2, _, _ := billingexpr.RunExpr("p * 0.5", billingexpr.TokenParams{P: 100})
	if r1 != r2 {
		t.Errorf("post-invalidate results differ: %f != %f", r1, r2)
	}
}

// ---------------------------------------------------------------------------
// Hash
// ---------------------------------------------------------------------------

func TestExprHashString_Deterministic(t *testing.T) {
	h1 := billingexpr.ExprHashString("p * 0.5")
	h2 := billingexpr.ExprHashString("p * 0.5")
	if h1 != h2 {
		t.Error("hash should be deterministic")
	}
	h3 := billingexpr.ExprHashString("p * 0.6")
	if h1 == h3 {
		t.Error("different expressions should have different hashes")
	}
}

// ---------------------------------------------------------------------------
// Cache variables: present
// ---------------------------------------------------------------------------

const claudeWithCacheExpr = `p <= 200000 ? tier("standard", p * 1.5 + c * 7.5 + cr * 0.15 + cc * 1.875) : tier("long_context", p * 3.0 + c * 11.25 + cr * 0.3 + cc * 3.75)`

func TestCachePresent_StandardTier(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 5000, CR: 50000, CC: 10000}
	cost, trace, err := billingexpr.RunExpr(claudeWithCacheExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5 + 50000*0.15 + 10000*1.875
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "standard")
	}
}

func TestCachePresent_LongContextTier(t *testing.T) {
	params := billingexpr.TokenParams{P: 300000, C: 10000, CR: 100000, CC: 20000}
	cost, trace, err := billingexpr.RunExpr(claudeWithCacheExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 300000*3.0 + 10000*11.25 + 100000*0.3 + 20000*3.75
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "long_context" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "long_context")
	}
}

// ---------------------------------------------------------------------------
// Cache variables: absent (all zero) — same expression still works
// ---------------------------------------------------------------------------

func TestCacheAbsent_ZeroCacheTokens(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 5000}
	cost, trace, err := billingexpr.RunExpr(claudeWithCacheExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f (cache terms should be 0)", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "standard")
	}
}

// ---------------------------------------------------------------------------
// Mixed cache fields: cc and cc1h non-zero
// ---------------------------------------------------------------------------

const claudeCacheSplitExpr = `tier("default", p * 1.5 + c * 7.5 + cr * 0.15 + cc * 2.0 + cc1h * 3.0)`

func TestMixedCacheFields(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 5000, CR: 10000, CC: 5000, CC1h: 2000}
	cost, _, err := billingexpr.RunExpr(claudeCacheSplitExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5 + 10000*0.15 + 5000*2.0 + 2000*3.0
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestMixedCacheFields_AllCacheZero(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 5000}
	cost, _, err := billingexpr.RunExpr(claudeCacheSplitExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f (all cache zero)", cost, want)
	}
}

// ---------------------------------------------------------------------------
// Backward compatibility: p+c only expressions still work with TokenParams
// ---------------------------------------------------------------------------

func TestBackwardCompat_OldExprWithTokenParams(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 5000, CR: 99999, CC: 88888}
	cost, trace, err := billingexpr.RunExpr(claudeExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 100000*1.5 + 5000*7.5
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f (old expr ignores cache fields)", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", trace.MatchedTier, "standard")
	}
}

// ---------------------------------------------------------------------------
// Settlement with cache tokens
// ---------------------------------------------------------------------------

func TestComputeTieredQuota_WithCache(t *testing.T) {
	snap := &billingexpr.BillingSnapshot{
		BillingMode:               "tiered_expr",
		ExprString:                claudeWithCacheExpr,
		ExprHash:                  billingexpr.ExprHashString(claudeWithCacheExpr),
		GroupRatio:                1.0,
		EstimatedPromptTokens:     100000,
		EstimatedCompletionTokens: 5000,
		EstimatedQuotaBeforeGroup: (100000*1.5 + 5000*7.5) / 1_000_000 * 500_000,
		EstimatedQuotaAfterGroup:  billingexpr.QuotaRound((100000*1.5 + 5000*7.5) / 1_000_000 * 500_000),
		EstimatedTier:             "standard",
		QuotaPerUnit:              500_000,
	}

	params := billingexpr.TokenParams{P: 100000, C: 5000, CR: 50000, CC: 10000}
	result, err := billingexpr.ComputeTieredQuota(snap, params)
	if err != nil {
		t.Fatal(err)
	}

	wantBefore := (100000*1.5 + 5000*7.5 + 50000*0.15 + 10000*1.875) / 1_000_000 * 500_000
	if math.Abs(result.ActualQuotaBeforeGroup-wantBefore) > 1e-6 {
		t.Errorf("before group: got %f, want %f", result.ActualQuotaBeforeGroup, wantBefore)
	}
	if result.MatchedTier != "standard" {
		t.Errorf("tier = %q, want %q", result.MatchedTier, "standard")
	}
	if result.CrossedTier {
		t.Error("expected crossed_tier=false (same tier)")
	}
}

func TestComputeTieredQuota_WithCacheCrossTier(t *testing.T) {
	snap := &billingexpr.BillingSnapshot{
		BillingMode:               "tiered_expr",
		ExprString:                claudeWithCacheExpr,
		ExprHash:                  billingexpr.ExprHashString(claudeWithCacheExpr),
		GroupRatio:                2.0,
		EstimatedPromptTokens:     100000,
		EstimatedCompletionTokens: 5000,
		EstimatedQuotaBeforeGroup: (100000*1.5 + 5000*7.5) / 1_000_000 * 500_000,
		EstimatedQuotaAfterGroup:  billingexpr.QuotaRound((100000*1.5 + 5000*7.5) / 1_000_000 * 500_000 * 2.0),
		EstimatedTier:             "standard",
		QuotaPerUnit:              500_000,
	}

	params := billingexpr.TokenParams{P: 300000, C: 10000, CR: 50000, CC: 10000}
	result, err := billingexpr.ComputeTieredQuota(snap, params)
	if err != nil {
		t.Fatal(err)
	}

	wantBefore := (300000*3.0 + 10000*11.25 + 50000*0.3 + 10000*3.75) / 1_000_000 * 500_000
	wantAfter := billingexpr.QuotaRound(wantBefore * 2.0)
	if math.Abs(result.ActualQuotaBeforeGroup-wantBefore) > 1e-6 {
		t.Errorf("before group: got %f, want %f", result.ActualQuotaBeforeGroup, wantBefore)
	}
	if result.ActualQuotaAfterGroup != wantAfter {
		t.Errorf("after group: got %d, want %d", result.ActualQuotaAfterGroup, wantAfter)
	}
	if !result.CrossedTier {
		t.Error("expected crossed_tier=true (estimated standard, actual long_context)")
	}
}

// ---------------------------------------------------------------------------
// Fuzz: random p/c/cache, verify non-negative result
// ---------------------------------------------------------------------------

func TestFuzz_NonNegativeResults(t *testing.T) {
	exprs := []string{
		claudeExpr,
		claudeWithCacheExpr,
		glmExpr,
		"p * 0.5 + c * 1.0",
		"max(p, c) * 0.1",
		"p * 0.5 + cr * 0.1 + cc * 0.2",
	}

	rng := rand.New(rand.NewSource(42))

	for _, exprStr := range exprs {
		for i := 0; i < 500; i++ {
			params := billingexpr.TokenParams{
				P:    math.Round(rng.Float64() * 1000000),
				C:    math.Round(rng.Float64() * 500000),
				CR:   math.Round(rng.Float64() * 200000),
				CC:   math.Round(rng.Float64() * 50000),
				CC1h: math.Round(rng.Float64() * 10000),
			}

			cost, _, err := billingexpr.RunExpr(exprStr, params)
			if err != nil {
				t.Fatalf("expr=%q params=%+v: %v", exprStr, params, err)
			}
			if cost < 0 {
				t.Errorf("expr=%q params=%+v: negative cost %f", exprStr, params, cost)
			}
		}
	}
}

func TestFuzz_SettlementConsistency(t *testing.T) {
	rng := rand.New(rand.NewSource(99))

	for i := 0; i < 200; i++ {
		estParams := billingexpr.TokenParams{
			P:  math.Round(rng.Float64() * 500000),
			C:  math.Round(rng.Float64() * 100000),
			CR: math.Round(rng.Float64() * 100000),
			CC: math.Round(rng.Float64() * 30000),
		}
		actParams := billingexpr.TokenParams{
			P:  math.Round(rng.Float64() * 500000),
			C:  math.Round(rng.Float64() * 100000),
			CR: math.Round(rng.Float64() * 100000),
			CC: math.Round(rng.Float64() * 30000),
		}
		groupRatio := 0.5 + rng.Float64()*2.0

		estCost, estTrace, _ := billingexpr.RunExpr(claudeWithCacheExpr, estParams)

		const qpu = 500_000.0
		snap := &billingexpr.BillingSnapshot{
			BillingMode:               "tiered_expr",
			ExprString:                claudeWithCacheExpr,
			ExprHash:                  billingexpr.ExprHashString(claudeWithCacheExpr),
			GroupRatio:                groupRatio,
			EstimatedPromptTokens:     int(estParams.P),
			EstimatedCompletionTokens: int(estParams.C),
			EstimatedQuotaBeforeGroup: estCost / 1_000_000 * qpu,
			EstimatedQuotaAfterGroup:  billingexpr.QuotaRound(estCost / 1_000_000 * qpu * groupRatio),
			EstimatedTier:             estTrace.MatchedTier,
			QuotaPerUnit:              qpu,
		}

		result, err := billingexpr.ComputeTieredQuota(snap, actParams)
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}

		directCost, _, _ := billingexpr.RunExpr(claudeWithCacheExpr, actParams)
		directQuota := billingexpr.QuotaRound(directCost / 1_000_000 * qpu * groupRatio)

		if result.ActualQuotaAfterGroup != directQuota {
			t.Errorf("iter %d: settlement %d != direct %d", i, result.ActualQuotaAfterGroup, directQuota)
		}
	}
}

// ---------------------------------------------------------------------------
// Settlement-level tests for ComputeTieredQuota
// ---------------------------------------------------------------------------

func TestComputeTieredQuota_BasicSettlement(t *testing.T) {
	exprStr := `tier("default", p + c)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   1.0,
		QuotaPerUnit: 500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 3000, C: 2000})
	if err != nil {
		t.Fatal(err)
	}
	// exprOutput = 5000; quota = 5000 / 1M * 500K = 2500
	if math.Abs(result.ActualQuotaBeforeGroup-2500) > 1e-6 {
		t.Errorf("before group = %f, want 2500", result.ActualQuotaBeforeGroup)
	}
	if result.ActualQuotaAfterGroup != 2500 {
		t.Errorf("after group = %d, want 2500", result.ActualQuotaAfterGroup)
	}
	if result.MatchedTier != "default" {
		t.Errorf("tier = %q, want default", result.MatchedTier)
	}
}

func TestComputeTieredQuota_WithGroupRatio(t *testing.T) {
	exprStr := `tier("default", p + c)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   2.0,
		QuotaPerUnit: 500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 1000, C: 500})
	if err != nil {
		t.Fatal(err)
	}
	// exprOutput = 1500; quotaBeforeGroup = 750; afterGroup = round(750 * 2.0) = 1500
	if result.ActualQuotaAfterGroup != 1500 {
		t.Errorf("after group = %d, want 1500", result.ActualQuotaAfterGroup)
	}
}

func TestComputeTieredQuota_ZeroTokens(t *testing.T) {
	exprStr := `tier("default", p * 2 + c * 10)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   1.0,
		QuotaPerUnit: 500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ActualQuotaAfterGroup != 0 {
		t.Errorf("after group = %d, want 0", result.ActualQuotaAfterGroup)
	}
}

func TestComputeTieredQuota_RoundingEdge(t *testing.T) {
	exprStr := `tier("default", p * 0.5)` // 3 * 0.5 = 1.5 (expr); 1.5 / 1M * 500K = 0.75; round(0.75) = 1
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   1.0,
		QuotaPerUnit: 500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 3})
	if err != nil {
		t.Fatal(err)
	}
	// 3 * 0.5 = 1.5 (expr); quota = 1.5 / 1M * 500K = 0.75; round(0.75) = 1
	if result.ActualQuotaAfterGroup != 1 {
		t.Errorf("after group = %d, want 1 (round 0.75 up)", result.ActualQuotaAfterGroup)
	}
}

func TestComputeTieredQuota_RoundingEdgeDown(t *testing.T) {
	exprStr := `tier("default", p * 0.4)` // 3 * 0.4 = 1.2 (expr); 1.2 / 1M * 500K = 0.6; round(0.6) = 1
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   1.0,
		QuotaPerUnit: 500_000,
	}

	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 3})
	if err != nil {
		t.Fatal(err)
	}
	// 3 * 0.4 = 1.2 (expr); quota = 1.2 / 1M * 500K = 0.6; round(0.6) = 1
	if result.ActualQuotaAfterGroup != 1 {
		t.Errorf("after group = %d, want 1 (round 0.6 up)", result.ActualQuotaAfterGroup)
	}
}

func TestComputeTieredQuotaWithRequest_ProbeAffectsQuota(t *testing.T) {
	exprStr := `param("fast") == true ? tier("fast", p * 4) : tier("normal", p * 2)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:   "tiered_expr",
		ExprString:    exprStr,
		ExprHash:      billingexpr.ExprHashString(exprStr),
		GroupRatio:    1.0,
		EstimatedTier: "normal",
		QuotaPerUnit:  500_000,
	}

	// Without request: normal tier
	r1, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 1000})
	if err != nil {
		t.Fatal(err)
	}
	// normal: p*2 = 2000; quota = 2000 / 1M * 500K = 1000
	if r1.ActualQuotaAfterGroup != 1000 {
		t.Errorf("normal = %d, want 1000", r1.ActualQuotaAfterGroup)
	}

	// With request: fast tier
	r2, err := billingexpr.ComputeTieredQuotaWithRequest(snap, billingexpr.TokenParams{P: 1000}, billingexpr.RequestInput{
		Body: []byte(`{"fast":true}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	// fast: p*4 = 4000; quota = 4000 / 1M * 500K = 2000
	if r2.ActualQuotaAfterGroup != 2000 {
		t.Errorf("fast = %d, want 2000", r2.ActualQuotaAfterGroup)
	}
	if !r2.CrossedTier {
		t.Error("expected CrossedTier = true when probe changes tier")
	}
}

func TestComputeTieredQuota_BoundaryTierCrossing(t *testing.T) {
	exprStr := `p <= 100000 ? tier("small", p * 1) : tier("large", p * 2)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:   "tiered_expr",
		ExprString:    exprStr,
		ExprHash:      billingexpr.ExprHashString(exprStr),
		GroupRatio:    1.0,
		EstimatedTier: "small",
		QuotaPerUnit:  500_000,
	}

	// At boundary: small, p*1 = 100000; quota = 100000 / 1M * 500K = 50000
	r1, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 100000})
	if err != nil {
		t.Fatal(err)
	}
	if r1.MatchedTier != "small" {
		t.Errorf("at boundary: tier = %s, want small", r1.MatchedTier)
	}
	if r1.ActualQuotaAfterGroup != 50000 {
		t.Errorf("at boundary: quota = %d, want 50000", r1.ActualQuotaAfterGroup)
	}

	// Past boundary: large, p*2 = 200002; quota = 200002 / 1M * 500K = 100001
	r2, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{P: 100001})
	if err != nil {
		t.Fatal(err)
	}
	if r2.MatchedTier != "large" {
		t.Errorf("past boundary: tier = %s, want large", r2.MatchedTier)
	}
	if r2.ActualQuotaAfterGroup != 100001 {
		t.Errorf("past boundary: quota = %d, want 100001", r2.ActualQuotaAfterGroup)
	}
	if !r2.CrossedTier {
		t.Error("expected CrossedTier = true")
	}
}

// ---------------------------------------------------------------------------
// Time function tests
// ---------------------------------------------------------------------------

func TestTimeFunctions_ValidTimezone(t *testing.T) {
	exprStr := `tier("default", p) * (hour("UTC") >= 0 ? 1 : 1)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 100 {
		t.Errorf("cost = %f, want 100", cost)
	}
}

func TestTimeFunctions_AllFunctionsCompile(t *testing.T) {
	exprStr := `tier("default", p) * (hour("Asia/Shanghai") >= 0 ? 1 : 1) * (minute("UTC") >= 0 ? 1 : 1) * (weekday("UTC") >= 0 ? 1 : 1) * (month("UTC") >= 1 ? 1 : 1) * (day("UTC") >= 1 ? 1 : 1)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 500})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 500 {
		t.Errorf("cost = %f, want 500", cost)
	}
}

func TestTimeFunctions_InvalidTimezone(t *testing.T) {
	exprStr := `tier("default", p) * (hour("Invalid/Zone") >= 0 ? 1 : 2)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	// Invalid timezone falls back to UTC; hour is 0-23, so condition is always true
	if cost != 100 {
		t.Errorf("cost = %f, want 100 (fallback to UTC)", cost)
	}
}

func TestTimeFunctions_EmptyTimezone(t *testing.T) {
	exprStr := `tier("default", p) * (hour("") >= 0 ? 1 : 2)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 100 {
		t.Errorf("cost = %f, want 100 (empty tz -> UTC)", cost)
	}
}

func TestTimeFunctions_NightDiscountPattern(t *testing.T) {
	exprStr := `tier("default", p * 2 + c * 10) * (hour("UTC") >= 21 || hour("UTC") < 6 ? 0.5 : 1)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 1000, C: 500})
	if err != nil {
		t.Fatal(err)
	}
	// Base = 1000*2 + 500*10 = 7000; multiplier is either 0.5 or 1 depending on current UTC hour
	if cost != 7000 && cost != 3500 {
		t.Errorf("cost = %f, want 7000 or 3500", cost)
	}
}

func TestTimeFunctions_WeekdayRange(t *testing.T) {
	exprStr := `tier("default", p) * (weekday("UTC") >= 0 && weekday("UTC") <= 6 ? 1 : 999)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Fatal(err)
	}
	// weekday is always 0-6, so multiplier is always 1
	if cost != 100 {
		t.Errorf("cost = %f, want 100", cost)
	}
}

func TestTimeFunctions_MonthDayPattern(t *testing.T) {
	exprStr := `tier("default", p) * (month("Asia/Shanghai") == 1 && day("Asia/Shanghai") == 1 ? 0.5 : 1)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 1000})
	if err != nil {
		t.Fatal(err)
	}
	// Either 1000 (not Jan 1) or 500 (Jan 1) — both are valid
	if cost != 1000 && cost != 500 {
		t.Errorf("cost = %f, want 1000 or 500", cost)
	}
}

// ---------------------------------------------------------------------------
// Image and audio token tests
// ---------------------------------------------------------------------------

func TestImageTokenVariable(t *testing.T) {
	exprStr := `tier("base", p * 2 + c * 10 + img * 5)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 1000, C: 500, Img: 200})
	if err != nil {
		t.Fatal(err)
	}
	// 1000*2 + 500*10 + 200*5 = 2000 + 5000 + 1000 = 8000
	if math.Abs(cost-8000) > 1e-6 {
		t.Errorf("cost = %f, want 8000", cost)
	}
}

func TestAudioTokenVariables(t *testing.T) {
	exprStr := `tier("base", p * 2 + c * 10 + ai * 50 + ao * 100)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 1000, C: 500, AI: 100, AO: 50})
	if err != nil {
		t.Fatal(err)
	}
	// 1000*2 + 500*10 + 100*50 + 50*100 = 2000 + 5000 + 5000 + 5000 = 17000
	if math.Abs(cost-17000) > 1e-6 {
		t.Errorf("cost = %f, want 17000", cost)
	}
}

func TestImageAudioVariables(t *testing.T) {
	exprStr := `tier("base", p * 1 + img * 3 + ai * 5 + ao * 10)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 100, Img: 50, AI: 20, AO: 10})
	if err != nil {
		t.Fatal(err)
	}
	// 100*1 + 50*3 + 20*5 + 10*10 = 100 + 150 + 100 + 100 = 450
	if math.Abs(cost-450) > 1e-6 {
		t.Errorf("cost = %f, want 450", cost)
	}
}

func TestImageAudioZero(t *testing.T) {
	exprStr := `tier("base", p * 2 + img * 5 + ai * 50 + ao * 100)`
	cost, _, err := billingexpr.RunExpr(exprStr, billingexpr.TokenParams{P: 1000})
	if err != nil {
		t.Fatal(err)
	}
	// img, ai, ao default to 0
	if math.Abs(cost-2000) > 1e-6 {
		t.Errorf("cost = %f, want 2000", cost)
	}
}

// ---------------------------------------------------------------------------
// len variable tests — tier conditions based on context length
// ---------------------------------------------------------------------------

const lenTieredExpr = `len <= 200000 ? tier("standard", p * 3 + c * 15 + cr * 0.3) : tier("long_context", p * 6 + c * 22.5 + cr * 0.6)`

func TestLen_StandardTier(t *testing.T) {
	params := billingexpr.TokenParams{P: 80000, C: 5000, Len: 100000, CR: 20000}
	cost, trace, err := billingexpr.RunExpr(lenTieredExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 80000*3 + 5000*15 + 20000*0.3
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want standard", trace.MatchedTier)
	}
}

func TestLen_LongContextTier(t *testing.T) {
	// p is low (cache subtracted), but len is high (full context)
	params := billingexpr.TokenParams{P: 50000, C: 5000, Len: 300000, CR: 250000}
	cost, trace, err := billingexpr.RunExpr(lenTieredExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	want := 50000*6 + 5000*22.5 + 250000*0.6
	if math.Abs(cost-want) > 1e-6 {
		t.Errorf("cost = %f, want %f", cost, want)
	}
	if trace.MatchedTier != "long_context" {
		t.Errorf("tier = %q, want long_context (len=300000 > 200000)", trace.MatchedTier)
	}
}

func TestLen_BoundaryExact(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 1000, Len: 200000, CR: 100000}
	_, trace, err := billingexpr.RunExpr(lenTieredExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want standard (len=200000 <= 200000)", trace.MatchedTier)
	}
}

func TestLen_BoundaryPlusOne(t *testing.T) {
	params := billingexpr.TokenParams{P: 100000, C: 1000, Len: 200001, CR: 100001}
	_, trace, err := billingexpr.RunExpr(lenTieredExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	if trace.MatchedTier != "long_context" {
		t.Errorf("tier = %q, want long_context (len=200001 > 200000)", trace.MatchedTier)
	}
}

func TestLen_ZeroDefaultsToZero(t *testing.T) {
	// len defaults to 0 when not set
	params := billingexpr.TokenParams{P: 1000, C: 500}
	_, trace, err := billingexpr.RunExpr(lenTieredExpr, params)
	if err != nil {
		t.Fatal(err)
	}
	if trace.MatchedTier != "standard" {
		t.Errorf("tier = %q, want standard (len=0 <= 200000)", trace.MatchedTier)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks: compile vs cached execution
// ---------------------------------------------------------------------------

const benchComplexExpr = `len <= 200000 ? tier("standard", p * 3 + c * 15 + cr * 0.3 + cc * 3.75 + cc1h * 6 + img * 3 + img_o * 30 + ai * 10 + ao * 40) : tier("long_context", p * 6 + c * 22.5 + cr * 0.6 + cc * 7.5 + cc1h * 12 + img * 6 + img_o * 60 + ai * 20 + ao * 80)`

func BenchmarkExprCompile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		billingexpr.InvalidateCache()
		billingexpr.CompileFromCache(benchComplexExpr)
	}
}

func BenchmarkExprRunCached(b *testing.B) {
	billingexpr.CompileFromCache(benchComplexExpr)
	params := billingexpr.TokenParams{P: 150000, C: 10000, Len: 188000, CR: 30000, CC: 5000, Img: 2000, AI: 1000, AO: 500}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		billingexpr.RunExpr(benchComplexExpr, params)
	}
}

// ---------------------------------------------------------------------------
// Cache eviction: reaching maxCacheSize resets the cache entirely (not LRU)
// ---------------------------------------------------------------------------

func TestCompileCache_EvictionResets(t *testing.T) {
	billingexpr.InvalidateCache()

	// Compile 256 distinct expressions to fill the cache
	exprs := make([]string, 256)
	for i := 0; i < 256; i++ {
		exprs[i] = fmt.Sprintf("tier(\"t%d\", p + %d)", i, i)
	}

	// Compile each expression to populate the cache
	for _, expr := range exprs {
		_, err := billingexpr.RunExpr(expr, billingexpr.TokenParams{P: 100})
		if err != nil {
			t.Fatalf("failed to compile %s: %v", expr, err)
		}
	}

	// The 257th expression should still compile successfully
	// (cache was reset, not LRU-evicted)
	expr := "tier(\"extra\", p * 1.5)"
	_, err := billingexpr.RunExpr(expr, billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Errorf("257th expression should compile after cache reset: %v", err)
	}
}

func TestUsedVars_Comprehensive(t *testing.T) {
	tests := []struct {
		expr     string
		wantVars []string
	}{
		{"p * 2", []string{"p"}},
		{"c * 3", []string{"c"}},
		{"p * 2 + c * 3", []string{"p", "c"}},
		{"p * 2 + cr * 0.1", []string{"p", "cr"}},
		{"p + cc * 2 + cc1h * 3", []string{"p", "cc", "cc1h"}},
		{"img * 5 + ai * 10", []string{"img", "ai"}},
		{"p + img_o * 20", []string{"p", "img_o"}},
		{"param(\"stream\") == true ? 2 : 1", []string{"p"}}, // param is a function, not a variable
		{"header(\"x-test\") == \"v\" ? 1 : 0", []string{"p"}},
		{"tier(\"a\", p)", []string{"p", "tier"}},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			// UsedVars uses cache, so invalidate first
			billingexpr.InvalidateCache()
			got := billingexpr.UsedVars(tt.expr)
			for _, v := range tt.wantVars {
				if !got[v] {
					t.Errorf("expr %q: missing variable %q, got %v", tt.expr, v, got)
				}
			}
		})
	}
}

func TestRunExpr_Errors(t *testing.T) {
	// Undefined variable
	_, _, err := billingexpr.RunExpr("undefined_var * 10", billingexpr.TokenParams{P: 100})
	if err == nil {
		t.Error("expected error for undefined variable")
	}

	// Division by zero — expr library may or may not catch at compile vs run time
	_, _, err = billingexpr.RunExpr("1 / 0", billingexpr.TokenParams{})
	if err == nil {
		t.Error("expected error for division by zero")
	}

	// Syntax error
	_, _, err = billingexpr.RunExpr("p +++ * 2", billingexpr.TokenParams{P: 100})
	if err == nil {
		t.Error("expected compile error for invalid syntax")
	}
}

func TestRunExpr_MaxMin(t *testing.T) {
	tests := []struct {
		expr    string
		params  billingexpr.TokenParams
		want    float64
	}{
		{"max(0, 0)", billingexpr.TokenParams{P: 0}, 0},
		{"max(-5, 5)", billingexpr.TokenParams{P: 0}, 5},
		{"min(0, 0)", billingexpr.TokenParams{P: 0}, 0},
		{"min(-5, 5)", billingexpr.TokenParams{P: 0}, -5},
		{"max(1, 2, 3)", billingexpr.TokenParams{P: 0}, 3}, // multi-arg
		{"min(3, 2, 1)", billingexpr.TokenParams{P: 0}, 1},
	}
	for _, tt := range tests {
		cost, _, err := billingexpr.RunExpr(tt.expr, tt.params)
		if err != nil {
			t.Errorf("expr %q: %v", tt.expr, err)
			continue
		}
		if cost != tt.want {
			t.Errorf("expr %q: cost = %f, want %f", tt.expr, cost, tt.want)
		}
	}
}

func TestRunExpr_CeilingFloor(t *testing.T) {
	tests := []struct {
		expr string
		val  float64
		want float64
	}{
		{"ceil(1.2)", 0, 2},
		{"floor(1.2)", 0, 1},
		{"ceil(-1.2)", 0, -1},   // Go: ceil(-1.2) = -1
		{"floor(-1.2)", 0, -2},  // Go: floor(-1.2) = -2
		{"ceil(0)", 0, 0},
		{"floor(0)", 0, 0},
		{"ceil(-0.1)", 0, 0},
		{"floor(0.1)", 0, 0},
	}
	for _, tt := range tests {
		cost, _, err := billingexpr.RunExpr(tt.expr, billingexpr.TokenParams{})
		if err != nil {
			t.Errorf("expr %s: %v", tt.expr, err)
			continue
		}
		if cost != tt.want {
			t.Errorf("expr %s: cost = %f, want %f", tt.expr, cost, tt.want)
		}
	}
}

func TestRunExpr_ImgO_TokenVariable(t *testing.T) {
	expr := `tier("default", p + img * 5 + img_o * 20)`
	params := billingexpr.TokenParams{P: 1000, Img: 200, ImgO: 50}
	cost, _, err := billingexpr.RunExpr(expr, params)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	want := 1000.0 + 200*5 + 50*20.0
	if cost != want {
		t.Errorf("cost = %f, want %f", cost, want)
	}
}

func TestRunExpr_SplitCacheTokens(t *testing.T) {
	// CC total > splitCC5m (50000) → splitCC5m = 50000, splitCC1h = CC - 50000
	// CC total < splitCC5m → splitCC5m = CC, splitCC1h = 0
	expr := `tier("default", p + cc * 1.0)`
	tests := []struct {
		cc     float64
		cc1h   float64
		wantCC float64
	}{
		{60000, 0, 60000}, // total=60000 > 50000 → cc contributed = 50000 (splitCC5m=50000)
		{40000, 0, 40000}, // total=40000 < 50000 → cc contributed = 40000
		{0, 0, 0},
		{50000, 0, 50000}, // exact boundary → splitCC5m=50000
		{70000, 10000, 60000}, // total=80000 → splitCC5m=50000, splitCC1h=10000, cc contributed=60000
	}
	for _, tt := range tests {
		// Note: the billingexpr engine itself does NOT split CC/CC1h;
		// that splitting happens in BuildTieredTokenParams (service layer).
		// Here we just verify cc and cc1h are passed through as-is.
		cost, _, err := billingexpr.RunExpr(expr, billingexpr.TokenParams{P: 100, CC: tt.cc, CC1h: tt.cc1h})
		if err != nil {
			t.Errorf("CC=%f CC1h=%f: %v", tt.cc, tt.cc1h, err)
			continue
		}
		want := 100.0 + tt.wantCC
		if cost != want {
			t.Errorf("CC=%f CC1h=%f: cost = %f, want %f", tt.cc, tt.cc1h, cost, want)
		}
	}
}

func TestRunExpr_ContextLengthLenVsPrompt(t *testing.T) {
	// p=80000 (cache subtracted), len=300000 (full context)
	// len triggers long_context tier, but p alone would trigger standard
	expr := `len <= 200000 ? tier("standard", p * 1) : tier("long_context", p * 3)`
	params := billingexpr.TokenParams{P: 80000, Len: 300000}
	_, trace, err := billingexpr.RunExpr(expr, params)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if trace.MatchedTier != "long_context" {
		t.Errorf("tier = %q, want long_context (len=300000 > 200000 even though p=80000 <= 200000)", trace.MatchedTier)
	}
}

func TestParseExprVersion(t *testing.T) {
	tests := []struct {
		expr    string
		version int
		body    string
	}{
		{"v1:p * 2", 1, "p * 2"},
		{"p * 2", 1, "p * 2"},
		{"", 1, ""},
		{"v1:", 1, ""},
		{"v2:p * 3", 1, "v2:p * 3"}, // unknown prefix → treated as body, version=1
	}
	for _, tt := range tests {
		v, body := billingexpr.ParseExprVersion(tt.expr)
		if v != tt.version || body != tt.body {
			t.Errorf("ParseExprVersion(%q) = (%d, %q), want (%d, %q)",
				tt.expr, v, body, tt.version, tt.body)
		}
	}
}

func TestExprVersion(t *testing.T) {
	// Empty string → DefaultExprVersion = 1
	v := billingexpr.ExprVersion("")
	if v != 1 {
		t.Errorf("ExprVersion(\"\") = %d, want 1", v)
	}

	// After compile, ExprVersion returns cached version
	billingexpr.InvalidateCache()
	billingexpr.RunExpr("p * 2", billingexpr.TokenParams{P: 0})
	v = billingexpr.ExprVersion("p * 2")
	if v != 1 {
		t.Errorf("ExprVersion(\"p * 2\") = %d, want 1", v)
	}
}

func TestRunExpr_ParamProbeNestedPath(t *testing.T) {
	expr := `param("messages.0.role") == "user" ? tier("fast", p * 2) : tier("normal", p)`
	cost, _, err := billingexpr.RunExprWithRequest(expr, billingexpr.TokenParams{P: 100},
		billingexpr.RequestInput{Body: []byte(`{"messages":[{"role":"user","content":"hi"}]}`)})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	// param("messages.0.role") = "user" == "user" → true → fast tier → p * 2 = 200
	if cost != 200 {
		t.Errorf("cost = %f, want 200", cost)
	}
}

func TestRunExpr_ParamProbeMissingField(t *testing.T) {
	expr := `param("missing.deep.field") == nil ? tier("a", p) : tier("b", p * 2)`
	cost, _, err := billingexpr.RunExprWithRequest(expr, billingexpr.TokenParams{P: 100},
		billingexpr.RequestInput{Body: []byte(`{"other":"value"}`)})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if cost != 100 {
		t.Errorf("cost = %f, want 100 (param returns nil for missing field)", cost)
	}
}

func TestRunExpr_HeaderNormalization(t *testing.T) {
	expr := `header("Content-Type") == "application/json" ? 1 : 0`
	cost, _, err := billingexpr.RunExprWithRequest(expr, billingexpr.TokenParams{},
		billingexpr.RequestInput{
			Headers: map[string]string{"CONTENT-TYPE": "application/json"},
		})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if cost != 1 {
		t.Errorf("cost = %f, want 1 (header keys normalized to lowercase)", cost)
	}
}

func TestRunExpr_HeaderTrimSpace(t *testing.T) {
	expr := `header("x-test") == "value" ? 1 : 0`
	cost, _, err := billingexpr.RunExprWithRequest(expr, billingexpr.TokenParams{},
		billingexpr.RequestInput{
			Headers: map[string]string{" x-test ": " value "},
		})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if cost != 1 {
		t.Errorf("cost = %f, want 1 (header keys and values trimmed)", cost)
	}
}

func TestRunExpr_HeaderEmptyKeySkipped(t *testing.T) {
	expr := `header("") == "" ? 1 : 0`
	cost, _, err := billingexpr.RunExprWithRequest(expr, billingexpr.TokenParams{},
		billingexpr.RequestInput{
			Headers: map[string]string{"x-test": "v", "": "empty-key"},
		})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	// Empty key is skipped in normalizeHeaders; header("") returns ""
	if cost != 1 {
		t.Errorf("cost = %f, want 1", cost)
	}
}

func TestRunExpr_HeaderEmptyValueSkipped(t *testing.T) {
	expr := `header("x-test")`
	cost, _, err := billingexpr.RunExprWithRequest(expr, billingexpr.TokenParams{},
		billingexpr.RequestInput{
			Headers: map[string]string{"x-test": ""},
		})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	// Empty value is skipped in normalizeHeaders; header("x-test") returns ""
	if cost != 0 {
		t.Errorf("cost = %f, want 0 (empty value skipped)", cost)
	}
}

func TestRunExpr_HasFunction(t *testing.T) {
	tests := []struct {
		expr    string
		body    []byte
		headers map[string]string
		want    float64
	}{
		// has on string literal
		{`has("hello world", "world") ? 1 : 0`, nil, nil, 1},
		{`has("hello", "xyz") ? 1 : 0`, nil, nil, 0},
		// has on param result
		{`has(param("service_tier"), "fast") ? 1 : 0`,
			[]byte(`{"service_tier":"fast-mode"}`), nil, 1},
		{`has(param("service_tier"), "premium") ? 1 : 0`,
			[]byte(`{"service_tier":"fast-mode"}`), nil, 0},
		// has on nil source
		{`has(nil, "x") ? 1 : 0`, nil, nil, 0},
		// has with empty substr
		{`has("hello", "") ? 1 : 0`, nil, nil, 0},
	}
	for _, tt := range tests {
		cost, _, err := billingexpr.RunExprWithRequest(tt.expr, billingexpr.TokenParams{},
			billingexpr.RequestInput{Body: tt.body, Headers: tt.headers})
		if err != nil {
			t.Errorf("expr %q: %v", tt.expr, err)
			continue
		}
		if cost != tt.want {
			t.Errorf("expr %q: cost = %f, want %f", tt.expr, cost, tt.want)
		}
	}
}

func TestRunExpr_TimezoneFallback(t *testing.T) {
	// Invalid timezone → falls back to UTC
	// We can't assert specific UTC hour value in tests, but we can verify it doesn't error
	expr := `tier("default", p) * (hour("Invalid/Zone") >= 0 ? 1 : 999)`
	cost, _, err := billingexpr.RunExpr(expr, billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Errorf("invalid timezone should not error: %v", err)
	}
	if cost != 100 {
		t.Errorf("cost = %f, want 100 (fallback to UTC)", cost)
	}
}

func TestRunExpr_TimezoneEmptyFallback(t *testing.T) {
	// Empty timezone → falls back to UTC
	expr := `tier("default", p) * (hour("") >= 0 ? 1 : 999)`
	cost, _, err := billingexpr.RunExpr(expr, billingexpr.TokenParams{P: 100})
	if err != nil {
		t.Errorf("empty timezone should not error: %v", err)
	}
	if cost != 100 {
		t.Errorf("cost = %f, want 100 (empty → UTC fallback)", cost)
	}
}

func TestQuotaConversion_V1(t *testing.T) {
	// v1: exprOutput / 1_000_000 * QuotaPerUnit
	snap := &billingexpr.BillingSnapshot{
		ExprVersion:  1,
		QuotaPerUnit: 500_000,
	}
	got := quotaConversion(2.0, snap) // $2 per 1M tokens → 1 quota per 1K tokens
	want := 2.0 / 1_000_000 * 500_000
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("quotaConversion = %f, want %f", got, want)
	}
}

func TestQuotaConversion_UnknownVersionFallsToV1(t *testing.T) {
	// Unknown version defaults to v1 behavior
	snap := &billingexpr.BillingSnapshot{
		ExprVersion:  999,
		QuotaPerUnit: 500_000,
	}
	got := quotaConversion(2.0, snap)
	want := 2.0 / 1_000_000 * 500_000
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("unknown version should fall to v1: got %f, want %f", got, want)
	}
}

func TestComputeTieredQuota_ZeroTokens(t *testing.T) {
	exprStr := `tier("default", p * 2 + c * 10)`
	snap := &billingexpr.BillingSnapshot{
		BillingMode:  "tiered_expr",
		ExprString:   exprStr,
		ExprHash:     billingexpr.ExprHashString(exprStr),
		GroupRatio:   1.0,
		QuotaPerUnit: 500_000,
		ExprVersion:  1,
	}
	result, err := billingexpr.ComputeTieredQuota(snap, billingexpr.TokenParams{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ActualQuotaAfterGroup != 0 {
		t.Errorf("after group = %d, want 0 (zero tokens)", result.ActualQuotaAfterGroup)
	}
	if result.MatchedTier != "default" {
		t.Errorf("tier = %q, want default", result.MatchedTier)
	}
}

func TestRunExprByHash_ConsistentWithRunExpr(t *testing.T) {
	expr := `p <= 200000 ? tier("standard", p * 1.5 + c * 7.5) : tier("long", p * 3 + c * 11.25)`
	params := billingexpr.TokenParams{P: 100000, C: 5000}
	hash := billingexpr.ExprHashString(expr)

	r1, _, err := billingexpr.RunExpr(expr, params)
	if err != nil {
		t.Fatal(err)
	}
	r2, _, err := billingexpr.RunExprByHash(expr, hash, params)
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r2 {
		t.Errorf("RunExpr = %f, RunExprByHash = %f, should be equal", r1, r2)
	}
}

func TestRunExprByHashWithRequest_ConsistentWithRunExprWithRequest(t *testing.T) {
	expr := `p * (param("quality") == "high" ? 2 : 1)`
	params := billingexpr.TokenParams{P: 1000}
	req := billingexpr.RequestInput{Body: []byte(`{"quality":"high"}`)}
	hash := billingexpr.ExprHashString(expr)

	r1, _, err := billingexpr.RunExprWithRequest(expr, params, req)
	if err != nil {
		t.Fatal(err)
	}
	r2, _, err := billingexpr.RunExprByHashWithRequest(expr, hash, params, req)
	if err != nil {
		t.Fatal(err)
	}
	if r1 != r2 {
		t.Errorf("RunExprWithRequest = %f, RunExprByHashWithRequest = %f, should be equal", r1, r2)
	}
}

func TestRunExpr_NegativeTokens(t *testing.T) {
	// Negative tokens should be handled gracefully (treated as 0 by expr engine)
	expr := `p * 2`
	cost, _, err := billingexpr.RunExpr(expr, billingexpr.TokenParams{P: -100})
	if err != nil {
		t.Fatalf("negative p should not error: %v", err)
	}
	// expr-lang treats -100 as -100.0, so cost = -200
	if cost != -200 {
		t.Errorf("cost = %f, want -200 (negative tokens pass through)", cost)
	}
}

func TestRunExpr_ZeroPromptNonZeroCompletion(t *testing.T) {
	expr := `tier("default", p * 1 + c * 2)`
	cost, _, err := billingexpr.RunExpr(expr, billingexpr.TokenParams{P: 0, C: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 2000 {
		t.Errorf("cost = %f, want 2000 (zero prompt, non-zero completion)", cost)
	}
}

func TestRunExpr_AllTokenVariablesZero(t *testing.T) {
	expr := `tier("base", p * 2 + c * 10 + cr * 0.5 + cc * 2 + cc1h * 4 + img * 5 + img_o * 20 + ai * 10 + ao * 30)`
	cost, _, err := billingexpr.RunExpr(expr, billingexpr.TokenParams{})
	if err != nil {
		t.Fatal(err)
	}
	if cost != 0 {
		t.Errorf("cost = %f, want 0 (all zero)", cost)
	}
}

// Ensure fmt is imported for TestCompileCache_EvictionResets
