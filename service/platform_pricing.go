package service

import (
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// SelectableModelInfo represents a model available for binding in the key creation/edit form.
type SelectableModelInfo struct {
	Model               string  `json:"model"`
	QuotaType           int     `json:"quota_type"` // 0=ratio(per 1M tokens), 1=fixed(per request)
	InputOriginalPrice  float64 `json:"input_original_price"`  // ratio:type=model_ratio*2($/1M), price:type=model_price($/req)
	OutputOriginalPrice float64 `json:"output_original_price"` // ratio:type=model_ratio*2*completion_ratio, price:type=0
	DiscountRatio       float64 `json:"discount_ratio"`
	InputDiscountedPrice  float64 `json:"input_discounted_price"`  // final price after discount
	OutputDiscountedPrice float64 `json:"output_discounted_price"` // final price after discount
	VendorType          string  `json:"vendor_type"`
	Source              string  `json:"source"` // "enterprise" | "platform"
	SheetId             int     `json:"sheet_id"`
	SheetName           string  `json:"sheet_name"`
}

// GetSelectableModelsForUser returns the list of models available in the user's current effective pricing sheet
// (used for key creation/edit form).
// Rules:
//   - User belongs to an enterprise with an active sheet → return models from enterprise sheet
//   - Otherwise → return models from platform sheet (enterprise type='platform')
//
// Data consistency constraint:
// Since the model list source is consistent with the binding source, there is no risk of
// "Token binds to a platform sheet but user belongs to an enterprise" conflict.
// That is: during auto-binding and manual binding, a Token's sheet_id can only come from
// the user's current effective pricing sheet.
func GetSelectableModelsForUser(userId int) []SelectableModelInfo {
	modelsMap := make(map[string]SelectableModelInfo)

	// Helper: compute price fields for a single model.
	// Uses the same calculation as model/pricing.go: ratio type uses model_ratio*2 as input unit price.
	addModelInfo := func(modelName, vendorType, source string, sheetId int, sheetName string, discountValue float64) {
		modelPrice, usePrice := ratio_setting.GetModelPrice(modelName, false)
		var inputOriginal, outputOriginal, inputDiscounted, outputDiscounted float64
		var quotaType int

		if usePrice {
			// Fixed price per request
			quotaType = 1
			inputOriginal = modelPrice
			outputOriginal = 0
			inputDiscounted = modelPrice * discountValue
			outputDiscounted = 0
		} else {
			// Ratio type: price per 1M tokens = model_ratio * 2
			quotaType = 0
			modelRatio, _, _ := ratio_setting.GetModelRatio(modelName)
			completionRatio := ratio_setting.GetCompletionRatio(modelName)
			inputOriginal = modelRatio * 2
			outputOriginal = inputOriginal * completionRatio
			inputDiscounted = inputOriginal * discountValue
			outputDiscounted = outputOriginal * discountValue
		}

		modelsMap[modelName] = SelectableModelInfo{
			Model:                  modelName,
			QuotaType:              quotaType,
			InputOriginalPrice:     inputOriginal,
			OutputOriginalPrice:    outputOriginal,
			DiscountRatio:          discountValue,
			InputDiscountedPrice:   inputDiscounted,
			OutputDiscountedPrice:  outputDiscounted,
			VendorType:             vendorType,
			Source:                source,
			SheetId:               sheetId,
			SheetName:             sheetName,
		}
	}

	// Priority 1: user's enterprise sheet
	enterpriseId, found := model.IsUserInEnterprise(userId)
	if found && enterpriseId > 0 {
		if enterpriseSheet, err := model.GetFirstActivePricingSheetByEnterpriseId(enterpriseId); err == nil && enterpriseSheet != nil {
			if items, err := model.GetPricingItemsBySheetId(enterpriseSheet.Id); err == nil && len(items) > 0 {
				for _, item := range items {
					for _, modelName := range item.Models {
						addModelInfo(modelName, item.VendorType, "enterprise", enterpriseSheet.Id, enterpriseSheet.Name, item.DiscountValue)
					}
				}
				return mapToSlice(modelsMap)
			}
		}
	}

	// Priority 2: platform sheet fallback — aggregate all active platform sheets
	// When multiple sheets configure the same model, use the lowest discount_value (best discount for user).
	platformSheets, err := model.GetAllActivePricingSheetsByEnterpriseIdByType(model.EnterpriseTypePlatform)
	if err == nil && len(platformSheets) > 0 {
		for _, platformSheet := range platformSheets {
			if items, err := model.GetPricingItemsBySheetId(platformSheet.Id); err == nil {
				for _, item := range items {
					for _, modelName := range item.Models {
						existing, exists := modelsMap[modelName]
						if !exists || item.DiscountValue < existing.DiscountRatio {
							addModelInfo(modelName, item.VendorType, "platform", platformSheet.Id, platformSheet.Name, item.DiscountValue)
						}
					}
				}
			}
		}
	}

	return mapToSlice(modelsMap)
}

func mapToSlice(m map[string]SelectableModelInfo) []SelectableModelInfo {
	result := make([]SelectableModelInfo, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}
