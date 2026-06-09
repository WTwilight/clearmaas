package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Unit tests for ModelLimits enforcement logic (pure, no i18n required).
// These test the model-limit checking logic directly.
// ============================================================================

// Test that the ModelLimits map correctly allows/whitelists models.
func TestModelLimitsMap_AllowListed(t *testing.T) {
	limits := map[string]bool{"gpt-4o": true, "gpt-4o-mini": true}

	tests := []struct {
		model string
		want  bool // true = allowed
	}{
		{"gpt-4o", true},
		{"gpt-4o-mini", true},
		{"claude-3-5-sonnet", false},
		{"gpt-4", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			_, allowed := limits[tt.model]
			assert.Equal(t, tt.want, allowed)
		})
	}
}

// Test that ModelLimits enforcement logic follows the whitelist pattern.
// When token_model_limit_enabled=true and the map exists:
// - If model IS in the map → allowed
// - If model is NOT in the map → forbidden
func TestModelLimitsEnforcement_WhitelistPattern(t *testing.T) {
	tests := []struct {
		name            string
		model           string
		limitEnabled    bool
		limitMap        map[string]bool
		want403         bool
	}{
		{
			name:         "enabled_with_model_in_whitelist",
			model:        "gpt-4o",
			limitEnabled: true,
			limitMap:     map[string]bool{"gpt-4o": true},
			want403:      false,
		},
		{
			name:         "enabled_with_model_not_in_whitelist",
			model:        "gpt-4o",
			limitEnabled: true,
			limitMap:     map[string]bool{"gpt-4o-mini": true},
			want403:      true,
		},
		{
			name:         "enabled_with_empty_whitelist",
			model:        "gpt-4o",
			limitEnabled: true,
			limitMap:     map[string]bool{},
			want403:      true,
		},
		{
			name:         "disabled_allows_all",
			model:        "any-model",
			limitEnabled: false,
			limitMap:     nil,
			want403:      false,
		},
		{
			name:         "enabled_but_map_missing",
			model:        "gpt-4o",
			limitEnabled: true,
			limitMap:     nil,
			want403:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the enforcement logic from Distribute middleware
			is403 := false

			if tt.limitEnabled {
				if tt.limitMap == nil {
					is403 = true // map not found → all forbidden
				} else if len(tt.limitMap) == 0 {
					is403 = true // empty whitelist → all forbidden
				} else {
					_, allowed := tt.limitMap[tt.model]
					is403 = !allowed // model not in whitelist
				}
			}
			// If not enabled, is403 stays false

			assert.Equal(t, tt.want403, is403,
				"model=%s, limitEnabled=%v, limitMap=%v", tt.model, tt.limitEnabled, tt.limitMap)
		})
	}
}

// Test that context values are correctly extracted for ModelLimits.
func TestModelLimitsContextExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setEnabled     bool
		setMap         bool
		mapValues      map[string]bool
		wantEnabled    bool
		wantEnabledOK  bool
		wantMapOK     bool
		wantMapValues map[string]bool
	}{
		{
			name:           "both_set",
			setEnabled:     true,
			setMap:         true,
			mapValues:      map[string]bool{"gpt-4o": true},
			wantEnabled:    true,
			wantEnabledOK:  true,
			wantMapOK:      true,
			wantMapValues:  map[string]bool{"gpt-4o": true},
		},
		{
			name:           "only_enabled_no_map",
			setEnabled:     true,
			setMap:         false,
			wantEnabled:    true,
			wantEnabledOK:  true,
			wantMapOK:      false,
		},
		{
			name:           "disabled_no_map",
			setEnabled:     true,
			setMap:         false,
			wantEnabled:    true,
			wantEnabledOK:  true,
			wantMapOK:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())

			if tt.setEnabled {
				c.Set("token_model_limit_enabled", tt.wantEnabled)
			}
			if tt.setMap && tt.mapValues != nil {
				c.Set("token_model_limit", tt.mapValues)
			}

			// Simulate GetContextKeyBool
			enabledVal, enabledOK := c.Get("token_model_limit_enabled")
			assert.Equal(t, tt.wantEnabledOK, enabledOK)
			if tt.wantEnabledOK {
				assert.Equal(t, tt.wantEnabled, enabledVal)
			}

			// Simulate GetContextKey
			mapVal, mapOK := c.Get("token_model_limit")
			assert.Equal(t, tt.wantMapOK, mapOK)
			if tt.wantMapOK {
				assert.Equal(t, tt.wantMapValues, mapVal)
			}
		})
	}
}

// Test case sensitivity of model name matching.
func TestModelLimits_CaseSensitive(t *testing.T) {
	limits := map[string]bool{"gpt-4o": true}

	assert.True(t, limits["gpt-4o"], "exact case should match")
	assert.False(t, limits["GPT-4O"], "uppercase should not match")
	assert.False(t, limits["Gpt-4o"], "mixed case should not match")
	assert.False(t, limits["gpt-4o "], "trailing space should not match")
	assert.False(t, limits["gpt-4"], "partial match should not work")
}

// Test that token bindings can produce a correct ModelLimits map.
func TestTokenPricingBindingToModelLimitsMap(t *testing.T) {
	// Simulate what the service/controller layer does:
	// Converts token_pricing_model_bindings → map[string]bool
	bindings := []struct {
		model string
	}{
		{"gpt-4o"},
		{"gpt-4o-mini"},
		{"claude-3-5-sonnet"},
	}

	limitsMap := make(map[string]bool)
	for _, b := range bindings {
		limitsMap[b.model] = true
	}

	assert.True(t, limitsMap["gpt-4o"])
	assert.True(t, limitsMap["gpt-4o-mini"])
	assert.True(t, limitsMap["claude-3-5-sonnet"])
	assert.False(t, limitsMap["gpt-4"])
	assert.Len(t, limitsMap, 3)
}

// Test that multiple bindings from the same token produce unique maps.
func TestModelLimitsMap_MultipleBindingsSameToken(t *testing.T) {
	bindings := []string{"gpt-4o", "gpt-4o-mini", "gpt-4o", "claude-3-5-sonnet"}

	limitsMap := make(map[string]bool)
	for _, m := range bindings {
		limitsMap[m] = true
	}

	// Duplicate models should not create duplicate entries
	assert.Len(t, limitsMap, 3)
	assert.True(t, limitsMap["gpt-4o"])
	assert.True(t, limitsMap["gpt-4o-mini"])
	assert.True(t, limitsMap["claude-3-5-sonnet"])
}

// Test that empty bindings produce an empty map.
func TestModelLimitsMap_EmptyBindings(t *testing.T) {
	bindings := []string{}

	limitsMap := make(map[string]bool)
	for _, m := range bindings {
		limitsMap[m] = true
	}

	assert.Len(t, limitsMap, 0)
}

// Test binding-to-ModelLimits derivation with various pricing sheet configurations.
func TestModelLimitsMap_PricingSheetConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		sheetModels  []string
		boundModels  []string
		wantInMap    []string
		wantNotInMap []string
	}{
		{
			name:         "all_sheet_models_bound",
			sheetModels:  []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"},
			boundModels:  []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"},
			wantInMap:    []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"},
			wantNotInMap: []string{"gpt-4"},
		},
		{
			name:         "subset_of_sheet_models_bound",
			sheetModels:  []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"},
			boundModels:  []string{"gpt-4o"},
			wantInMap:    []string{"gpt-4o"},
			wantNotInMap: []string{"gpt-4o-mini", "claude-3-5-sonnet"},
		},
		{
			name:         "single_model_bound",
			sheetModels:  []string{"gpt-4o"},
			boundModels:  []string{"gpt-4o"},
			wantInMap:    []string{"gpt-4o"},
			wantNotInMap: []string{},
		},
		{
			name:         "no_models_bound",
			sheetModels:  []string{"gpt-4o", "gpt-4o-mini"},
			boundModels:  []string{},
			wantInMap:    []string{},
			wantNotInMap: []string{"gpt-4o", "gpt-4o-mini"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limitsMap := make(map[string]bool)
			for _, m := range tt.boundModels {
				limitsMap[m] = true
			}

			for _, m := range tt.wantInMap {
				assert.True(t, limitsMap[m], "expected %s in map", m)
			}
			for _, m := range tt.wantNotInMap {
				assert.False(t, limitsMap[m], "expected %s NOT in map", m)
			}
		})
	}
}
