package types

import "fmt"

type GroupRatioInfo struct {
	GroupRatio          float64
	GroupSpecialRatio   float64
	HasSpecialRatio     bool
	RatioSource         string
	EnterpriseSheetId   int
	EnterpriseSheetName string
	// PerCallPriceSheet is the fixed per-call price from the enterprise pricing sheet (美元/次).
	// A value > 0 indicates the pricing sheet item is of type per_call.
	PerCallPriceSheet float64
}

type PriceData struct {
	FreeModel            bool
	ModelPrice           float64
	ModelRatio           float64
	CompletionRatio      float64
	CacheRatio           float64
	CacheCreationRatio   float64
	CacheCreation5mRatio float64
	CacheCreation1hRatio float64
	ImageRatio           float64
	AudioRatio           float64
	AudioCompletionRatio float64
	OtherRatios          map[string]float64
	UsePrice             bool
	Quota                int // 按次计费的最终额度（MJ / Task）
	QuotaToPreConsume    int // 按量计费的预消耗额度
	GroupRatioInfo       GroupRatioInfo
	// PerCallPriceSheet is the fixed per-call price from the enterprise pricing sheet (美元/次).
	// When set ( > 0), the billing uses this as the absolute price per call instead of
	// computing quota via modelRatio. A value of -1 signals "per-call billing" without a
	// sheet price (falls back to model settings / TaskPricePatches).
	PerCallPriceSheet float64
}

func (p *PriceData) AddOtherRatio(key string, ratio float64) {
	if p.OtherRatios == nil {
		p.OtherRatios = make(map[string]float64)
	}
	if ratio <= 0 {
		return
	}
	p.OtherRatios[key] = ratio
}

func (p *PriceData) ToSetting() string {
	return fmt.Sprintf("ModelPrice: %f, ModelRatio: %f, CompletionRatio: %f, CacheRatio: %f, GroupRatio: %f (source: %s, sheetId: %d, sheet: %s), UsePrice: %t, CacheCreationRatio: %f, CacheCreation5mRatio: %f, CacheCreation1hRatio: %f, QuotaToPreConsume: %d, ImageRatio: %f, AudioRatio: %f, AudioCompletionRatio: %f, PerCallPriceSheet: %f",
		p.ModelPrice, p.ModelRatio, p.CompletionRatio, p.CacheRatio, p.GroupRatioInfo.GroupRatio, p.GroupRatioInfo.RatioSource, p.GroupRatioInfo.EnterpriseSheetId, p.GroupRatioInfo.EnterpriseSheetName, p.UsePrice, p.CacheCreationRatio, p.CacheCreation5mRatio, p.CacheCreation1hRatio, p.QuotaToPreConsume, p.ImageRatio, p.AudioRatio, p.AudioCompletionRatio, p.PerCallPriceSheet)
}
