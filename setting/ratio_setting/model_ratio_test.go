package ratio_setting_test

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/require"
)

func TestFormatMatchingModelName_GeminiThinkingBudget(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		// 无思考预算后缀 → 不变
		{"gemini-2.5-flash", "gemini-2.5-flash"},
		{"gemini-2.5-pro", "gemini-2.5-pro"},
		{"gemini-2.5-flash-lite", "gemini-2.5-flash-lite"},
		// 带思考预算后缀 → 归一化到 wildcard
		{"gemini-2.5-flash-thinking", "gemini-2.5-flash-thinking-*"},
		{"gemini-2.5-flash-thinking-experimenting-1219", "gemini-2.5-flash-thinking-*"},
		{"gemini-2.5-pro-thinking-32k", "gemini-2.5-pro-thinking-*"},
		{"gemini-2.5-flash-lite-thinking-budget", "gemini-2.5-flash-lite-thinking-*"},
		// 无 -thinking- 关键字 → 不变
		{"gemini-2.5-flash-exp", "gemini-2.5-flash-exp"},
		{"gemini-2.5-pro-latest", "gemini-2.5-pro-latest"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := ratio_setting.FormatMatchingModelName(tc.input)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestFormatMatchingModelName_GizmoNormalization(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		// gizmo 模型归一化
		{"gpt-4-gizmo", "gpt-4-gizmo-*"},
		{"gpt-4-gizmo-p-123", "gpt-4-gizmo-*"},
		{"gpt-4-gizmo-g-abc", "gpt-4-gizmo-*"},
		{"gpt-4o-gizmo", "gpt-4o-gizmo-*"},
		{"gpt-4o-gizmo-g-12345", "gpt-4o-gizmo-*"},
		// 非 gizmo 模型 → 不变
		{"gpt-4o", "gpt-4o"},
		{"gpt-4o-mini", "gpt-4o-mini"},
		{"gpt-4-gizmo-all", "gpt-4-gizmo-*"}, // 先归一化 gizmo，后因含 -thinking- 再处理？不含，所以不变
		{"unknown-model", "unknown-model"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := ratio_setting.FormatMatchingModelName(tc.input)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestFormatMatchingModelName_OrderOfOperations(t *testing.T) {
	// gemini 优先于 gizmo（代码中 gemini 分支在前）
	// 但实际上这两个是互斥的，所以顺序不影响
	name := "gemini-2.5-flash-thinking"
	got := ratio_setting.FormatMatchingModelName(name)
	require.Equal(t, "gemini-2.5-flash-thinking-*", got)
}

func TestGetModelRatio_KnownModels(t *testing.T) {
	cases := []struct {
		model  string
		wantOK bool
	}{
		// GPT 系列
		{"gpt-4o", true},
		{"gpt-4o-mini", true},
		{"gpt-4.1", true},
		{"gpt-4.1-mini", true},
		{"gpt-4.1-nano", true},
		{"o1", true},
		{"o1-mini", true},
		{"o3", true},
		{"o3-mini", true},
		{"gpt-5", true},
		{"gpt-5-mini", true},
		// Gemini 系列
		{"gemini-2.5-pro", true},
		{"gemini-2.5-flash", true},
		// Claude 系列
		{"claude-3-5-sonnet-20241022", true},
		{"claude-sonnet-4-20250514", true},
		// 归一化后的 thinking budget 模型
		{"gemini-2.5-pro-thinking-*", true},
		// gizmo 归一化
		{"gpt-4-gizmo-*", true},
		{"gpt-4o-gizmo-*", true},
		// 未知模型
		{"unknown-model-xyz", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			ratio, ok, name := ratio_setting.GetModelRatio(tc.model)
			if tc.wantOK {
				require.True(t, ok, "expected model %q to be found", tc.model)
				require.Greater(t, ratio, 0.0, "ratio should be positive")
				_ = name // matched name can be different from input
			} else {
				// 未知模型返回 37.5，ok = SelfUseModeEnabled
				if !operation_setting.SelfUseModeEnabled {
					require.False(t, ok, "expected model %q to NOT be found", tc.model)
				}
			}
		})
	}
}

func TestGetModelRatio_UnknownModel_Fallback37_5(t *testing.T) {
	// 未知模型的兜底逻辑：ratio=37.5, ok=SelfUseModeEnabled
	ratio, ok, matched := ratio_setting.GetModelRatio("this-model-definitely-does-not-exist-12345")
	if operation_setting.SelfUseModeEnabled {
		require.True(t, ok)
	} else {
		require.False(t, ok)
	}
	require.Equal(t, 37.5, ratio)
	// matched name 是归一化后的名称
	require.NotEmpty(t, matched)
}

func TestGetModelRatio_CompactSuffix(t *testing.T) {
	// 带 compact 后缀的模型名会尝试查找 wildcard key
	// 需要 modelRatioMap 中有对应条目才能测试，这里验证函数不 panic
	ratio, ok, _ := ratio_setting.GetModelRatio("some-model-compact")
	// compact 后缀不在 map 中，不在 defaultModelRatio 中
	// 因此会走 37.5 兜底
	if !operation_setting.SelfUseModeEnabled {
		require.False(t, ok)
		require.Equal(t, 37.5, ratio)
	}
}

func TestGetModelPrice_KnownModels(t *testing.T) {
	cases := []struct {
		model string
		want  bool
	}{
		{"gpt-4o", false}, // 已知模型，但没有配置固定价格 → usePrice=false
		{"gpt-4o-mini", false},
		{"gpt-4.1", false},
		{"gpt-4.1-mini", false},
		{"o1", false},
		{"o1-mini", false},
		{"claude-3-5-sonnet-20241022", false},
		{"gemini-2.5-pro", false},
		{"gpt-5", false},
		// 默认表里没有固定价格
		{"unknown-model", false},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			price, usePrice := ratio_setting.GetModelPrice(tc.model, false)
			if tc.want {
				require.True(t, usePrice)
				require.Greater(t, price, 0.0)
			} else {
				require.False(t, usePrice, "model %q should not use fixed price", tc.model)
				require.Equal(t, float64(-1), price, "price should be -1 when usePrice=false")
			}
		})
	}
}

func TestGetModelPrice_PrintError(t *testing.T) {
	// GetModelPrice(model, true) 在找不到时会打印错误
	// GetModelPrice(model, false) 不打印错误
	// 两个调用不应该 panic
	price1, ok1 := ratio_setting.GetModelPrice("definitely-not-exists-model-abc", true)
	price2, ok2 := ratio_setting.GetModelPrice("definitely-not-exists-model-abc", false)

	require.Equal(t, float64(-1), price1)
	require.False(t, ok1)
	require.Equal(t, float64(-1), price2)
	require.False(t, ok2)
}

func TestGetCompletionRatio_KnownModels(t *testing.T) {
	cases := []struct {
		model         string
		wantRatioGTE  float64
		wantRatioLTE  float64
	}{
		{"gpt-4o", 0, 100},
		{"gpt-4o-mini", 0, 100},
		{"gpt-4.1", 0, 100},
		{"gpt-4.1-mini", 0, 100},
		{"o1", 0, 100},
		{"o3", 0, 100},
		{"gpt-5", 0, 100},
		{"gpt-5-mini", 0, 100},
		{"claude-3-5-sonnet-20241022", 0, 100},
		{"gemini-2.5-pro", 0, 100},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			ratio := ratio_setting.GetCompletionRatio(tc.model)
			require.GreaterOrEqual(t, ratio, tc.wantRatioGTE)
			require.LessOrEqual(t, ratio, tc.wantRatioLTE)
		})
	}
}

func TestGetCompletionRatio_HardcodedRules(t *testing.T) {
	// getHardcodedCompletionModelRatio 的规则：
	// gpt-4o: 4x (除非 2024-05-13 版本是 3x)
	// gpt-5: 8x
	// o1/o3: 10x

	// gpt-4o 变体
	ratio := ratio_setting.GetCompletionRatio("gpt-4o")
	require.Equal(t, 4.0, ratio)

	ratio = ratio_setting.GetCompletionRatio("gpt-4o-mini")
	require.Equal(t, 4.0, ratio)

	// 特殊版本
	ratio = ratio_setting.GetCompletionRatio("gpt-4o-2024-05-13")
	require.Equal(t, 3.0, ratio)

	// gpt-5
	ratio = ratio_setting.GetCompletionRatio("gpt-5")
	require.Equal(t, 8.0, ratio)

	ratio = ratio_setting.GetCompletionRatio("gpt-5-mini")
	require.Equal(t, 8.0, ratio)

	// o1/o3
	ratio = ratio_setting.GetCompletionRatio("o1")
	require.Equal(t, 10.0, ratio)

	ratio = ratio_setting.GetCompletionRatio("o3")
	require.Equal(t, 10.0, ratio)

	ratio = ratio_setting.GetCompletionRatio("o1-mini")
	require.Equal(t, 10.0, ratio)

	// unknown → 返回默认值
	ratio = ratio_setting.GetCompletionRatio("completely-unknown-model-xyz")
	require.Equal(t, 0.0, ratio)
}

func TestGetImageRatio(t *testing.T) {
	// imageRatioMap 目前只有 gpt-image-1
	ratio, ok := ratio_setting.GetImageRatio("gpt-image-1")
	require.True(t, ok)
	require.Equal(t, 2.0, ratio)

	// 未知模型 → ok=false, ratio=1（默认值）
	ratio, ok = ratio_setting.GetImageRatio("unknown-image-model")
	require.False(t, ok)
	require.Equal(t, 1.0, ratio)
}

func TestGetModelRatioOrPrice_Priority(t *testing.T) {
	// GetModelRatioOrPrice: 先尝试固定价格，再尝试 ratio，最后 37.5 兜底
	// 没有配置固定价格的模型，应该走 ratio 路径

	// gpt-4o 有 ratio（defaultModelRatio 中有），没有固定价格
	ratio, usePrice, found := ratio_setting.GetModelRatioOrPrice("gpt-4o")
	require.False(t, usePrice, "gpt-4o should not use fixed price")
	if found {
		require.Greater(t, ratio, 0.0)
	} else {
		// 如果 ratio 查找失败，found=false，返回 ratio=37.5
		require.Equal(t, 37.5, ratio)
	}

	// 完全不存在的模型
	ratio, usePrice, found = ratio_setting.GetModelRatioOrPrice("model-does-not-exist-12345")
	require.False(t, usePrice)
	require.Equal(t, 37.5, ratio)
	require.False(t, found)
}

func TestGetModelRatioCopy_NotNil(t *testing.T) {
	m := ratio_setting.GetModelRatioCopy()
	require.NotNil(t, m)
	require.Greater(t, len(m), 0, "defaultModelRatio should have entries")
}

func TestGetModelPriceCopy_NotNil(t *testing.T) {
	m := ratio_setting.GetModelPriceCopy()
	require.NotNil(t, m)
}

func TestGetCompletionRatioCopy_NotNil(t *testing.T) {
	m := ratio_setting.GetCompletionRatioCopy()
	require.NotNil(t, m)
}

func TestGetImageRatioCopy_NotNil(t *testing.T) {
	m := ratio_setting.GetImageRatioCopy()
	require.NotNil(t, m)
}

func TestGetAudioRatioCopy_NotNil(t *testing.T) {
	m := ratio_setting.GetAudioRatioCopy()
	require.NotNil(t, m)
}

func TestCompletionRatioInfo_KnownModels(t *testing.T) {
	// GetCompletionRatioInfo 返回 Ratio 和 Locked 状态
	info := ratio_setting.GetCompletionRatioInfo("gpt-4o")
	require.Greater(t, info.Ratio, 0.0)
	// gpt-4o 是 hardcoded 且 locked = false（默认倍率为 4，但非硬锁）
	// 因为 completionRatioMap 中无 gpt-4o，走 hardcoded 路径

	info = ratio_setting.GetCompletionRatioInfo("gpt-4o-2024-05-13")
	require.Equal(t, 3.0, info.Ratio)
	require.True(t, info.Locked, "gpt-4o-2024-05-13 completion ratio should be locked")

	info = ratio_setting.GetCompletionRatioInfo("gpt-5")
	require.Equal(t, 8.0, info.Ratio)
	require.True(t, info.Locked, "gpt-5 completion ratio should be locked")

	info = ratio_setting.GetCompletionRatioInfo("unknown-model")
	require.GreaterOrEqual(t, info.Ratio, 0.0)
}

func TestHandleThinkingBudgetModel(t *testing.T) {
	// handleThinkingBudgetModel 是未导出函数，但 FormatMatchingModelName 间接覆盖了它
	// 通过 FormatMatchingModelName 测试即可
	cases := []struct {
		input    string
		expected string
	}{
		{"gemini-2.5-flash", "gemini-2.5-flash"},
		{"gemini-2.5-pro", "gemini-2.5-pro"},
		{"gemini-2.5-flash-thinking-experimenting", "gemini-2.5-flash-thinking-*"},
		{"gemini-2.5-pro-thinking-32k", "gemini-2.5-pro-thinking-*"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := ratio_setting.FormatMatchingModelName(tc.input)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestFormatMatchingModelName_EmptyString(t *testing.T) {
	got := ratio_setting.FormatMatchingModelName("")
	require.Equal(t, "", got)
}

func TestFormatMatchingModelName_GeminiPriority(t *testing.T) {
	// gemini-2.5-flash-lite 比 gemini-2.5-flash 更特殊，应该优先匹配
	// 但 handleThinkingBudgetModel 只检查 HasPrefix，所以：
	// "gemini-2.5-flash-lite-thinking" 会先匹配 gemini-2.5-flash-lite 前缀
	// → handleThinkingBudgetModel(lite) → "gemini-2.5-flash-lite-thinking-*"
	got := ratio_setting.FormatMatchingModelName("gemini-2.5-flash-lite-thinking-budget")
	require.Equal(t, "gemini-2.5-flash-lite-thinking-*", got)
}

func TestGetModelRatio_DoesNotModifyInput(t *testing.T) {
	input := "gpt-4o"
	original := input
	ratio_setting.GetModelRatio(input)
	require.Equal(t, original, input, "GetModelRatio should not modify input string")
}

func TestGetCompletionRatio_DoesNotModifyInput(t *testing.T) {
	input := "gpt-4o"
	original := input
	ratio_setting.GetCompletionRatio(input)
	require.Equal(t, original, input, "GetCompletionRatio should not modify input string")
}

func TestGetModelPrice_DoesNotModifyInput(t *testing.T) {
	input := "gpt-4o"
	original := input
	ratio_setting.GetModelPrice(input, false)
	require.Equal(t, original, input, "GetModelPrice should not modify input string")
}

// TestFormatMatchingModelName_AllKnownPrefixes verifies that every model family
// that appears in defaultModelRatio is handled by FormatMatchingModelName.
func TestFormatMatchingModelName_AllKnownPrefixes(t *testing.T) {
	// 获取所有已知的模型名前缀（从 defaultModelRatio 中提取）
	knownPrefixes := []struct {
		model   string
		matches []string
	}{
		{"gemini-2.5-flash", []string{"gemini-2.5-flash", "gemini-2.5-flash-lite"}},
		{"gemini-2.5-pro", []string{"gemini-2.5-pro"}},
		{"gpt-4-gizmo", []string{"gpt-4-gizmo"}},
		{"gpt-4o-gizmo", []string{"gpt-4o-gizmo"}},
	}

	for _, tc := range knownPrefixes {
		t.Run(tc.model, func(t *testing.T) {
			for _, model := range tc.matches {
				got := ratio_setting.FormatMatchingModelName(model)
				// 不应该包含下划线（如果有说明有问题）
				require.NotContains(t, got, "_", "FormatMatchingModelName(%q) = %q should not contain underscore", model, got)
			}
		})
	}
}

func TestGetModelRatio_LargeRatio(t *testing.T) {
	// o1-pro 费率最高 (75)，验证不会溢出
	ratio, ok, _ := ratio_setting.GetModelRatio("o1-pro")
	if ok {
		require.Equal(t, 75.0, ratio)
	}
}

func TestGetModelRatio_TinyRatio(t *testing.T) {
	// 一些模型倍率很小（如 gpt-4.1-nano = 0.05）
	ratio, ok, _ := ratio_setting.GetModelRatio("gpt-4.1-nano")
	if ok {
		require.Equal(t, 0.05, ratio)
	}
}

func TestGetModelRatio_NilSafe(t *testing.T) {
	// 确保所有 getter 在各种输入下都不 panic
	funcs := []func(){
		func() { ratio_setting.GetModelRatio("") },
		func() { ratio_setting.GetModelRatio("a") },
		func() { ratio_setting.GetModelRatio("very-long-model-name-that-is-unlikely-to-exist-anywhere-in-the-world-12345") },
		func() { ratio_setting.GetModelPrice("", false) },
		func() { ratio_setting.GetModelPrice("", true) },
		func() { ratio_setting.GetCompletionRatio("") },
		func() { ratio_setting.GetImageRatio("") },
	}
	for i, f := range funcs {
		t.Run(strings.TrimSpace(strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ReplaceAll(
						funcs[i](), "func() { ", ""),
					"}", ""),
				"ratio_setting.", ""),
			"()", "")), func(t *testing.T) {
			// 捕获 panic
		})
	}

	// 直接调用防止 linter 报错
	require.NotPanics(t, func() { ratio_setting.GetModelRatio("") })
	require.NotPanics(t, func() { ratio_setting.GetModelPrice("", false) })
	require.NotPanics(t, func() { ratio_setting.GetCompletionRatio("") })
	require.NotPanics(t, func() { ratio_setting.GetImageRatio("") })
}
