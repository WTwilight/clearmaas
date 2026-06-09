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
//   - Models must be set in a platform pricing sheet (platform sheet is the source of truth).
//   - Enterprise sheet has priority: if a model exists in both enterprise and platform sheets,
//     the enterprise discount is used.
//   - Platform sheet is the fallback when a model is not in the enterprise sheet.
//
// Data consistency constraint:
// Since the model list source is consistent with the binding source, there is no risk of
// "Token binds to a platform sheet but user belongs to an enterprise" conflict.
// That is: during auto-binding and manual binding, a Token's sheet_id can only come from
// the user's current effective pricing sheet.
func GetSelectableModelsForUser(userId int) []SelectableModelInfo {
	// Step 1: Collect all models from all active platform pricing sheets.
	// When the same model appears in multiple platform sheets, keep the lowest discount (best for user).
	platformModelMap := make(map[string]platformModelEntry)
	platformSheets, err := model.GetAllActivePricingSheetsByEnterpriseIdByType(model.EnterpriseTypePlatform)
	if err == nil && len(platformSheets) > 0 {
		for _, platformSheet := range platformSheets {
			items, err := model.GetPricingItemsBySheetId(platformSheet.Id)
			if err == nil {
				for _, item := range items {
					for _, modelName := range item.Models {
						existing, exists := platformModelMap[modelName]
						if !exists || item.DiscountValue < existing.DiscountValue {
							platformModelMap[modelName] = platformModelEntry{
								VendorType:    item.VendorType,
								DiscountValue: item.DiscountValue,
								SheetId:       platformSheet.Id,
								SheetName:     platformSheet.Name,
							}
						}
					}
				}
			}
		}
	}

	// If no platform pricing sheets are configured, return nothing.
	if len(platformModelMap) == 0 {
		return nil
	}

	// Step 2: Collect models from the user's enterprise pricing sheet (if any).
	// Enterprise models take priority over platform models for the same model.
	enterpriseModelMap := make(map[string]platformModelEntry)
	enterpriseSheet, err := model.GetFirstActivePricingSheetByEnterpriseIdByType(model.EnterpriseTypePlatform)
	enterpriseId, hasEnterprise := model.IsUserInEnterprise(userId)
	if hasEnterprise && enterpriseId > 0 {
		enterpriseSheet, err = model.GetFirstActivePricingSheetByEnterpriseId(enterpriseId)
		if err == nil && enterpriseSheet != nil {
			items, err := model.GetPricingItemsBySheetId(enterpriseSheet.Id)
			if err == nil {
				for _, item := range items {
					for _, modelName := range item.Models {
						enterpriseModelMap[modelName] = platformModelEntry{
							VendorType:    item.VendorType,
							DiscountValue: item.DiscountValue,
							SheetId:       enterpriseSheet.Id,
							SheetName:     enterpriseSheet.Name,
						}
					}
				}
			}
		}
	}

	// Step 3: Build result — platform models as base, with enterprise discount applied when available.
	// If a model is in both maps, enterprise takes priority.
	result := make([]SelectableModelInfo, 0, len(platformModelMap))
	for modelName, platformEntry := range platformModelMap {
		vendorType := platformEntry.VendorType
		discountValue := platformEntry.DiscountValue
		sheetId := platformEntry.SheetId
		sheetName := platformEntry.SheetName
		source := "platform"

		if enterpriseEntry, inEnterprise := enterpriseModelMap[modelName]; inEnterprise {
			vendorType = enterpriseEntry.VendorType
			discountValue = enterpriseEntry.DiscountValue
			sheetId = enterpriseEntry.SheetId
			sheetName = enterpriseEntry.SheetName
			source = "enterprise"
		}

		modelPrice, usePrice := ratio_setting.GetModelPrice(modelName, false)
		var inputOriginal, outputOriginal, inputDiscounted, outputDiscounted float64
		var quotaType int

		if usePrice {
			quotaType = 1
			inputOriginal = modelPrice
			outputOriginal = 0
			inputDiscounted = modelPrice * discountValue
			outputDiscounted = 0
		} else {
			quotaType = 0
			modelRatio, _, _ := ratio_setting.GetModelRatio(modelName)
			completionRatio := ratio_setting.GetCompletionRatio(modelName)
			inputOriginal = modelRatio * 2
			outputOriginal = inputOriginal * completionRatio
			inputDiscounted = inputOriginal * discountValue
			outputDiscounted = outputOriginal * discountValue
		}

		result = append(result, SelectableModelInfo{
			Model:                  modelName,
			QuotaType:              quotaType,
			InputOriginalPrice:     inputOriginal,
			OutputOriginalPrice:    outputOriginal,
			DiscountRatio:          discountValue,
			InputDiscountedPrice:   inputDiscounted,
			OutputDiscountedPrice:  outputDiscounted,
			VendorType:             vendorType,
			Source:                 source,
			SheetId:                sheetId,
			SheetName:              sheetName,
		})
	}

	return result
}

// platformModelEntry holds model entry info for platform/enterprise pricing aggregation.
type platformModelEntry struct {
	VendorType    string
	DiscountValue float64
	SheetId       int
	SheetName     string
}

func mapToSlice(m map[string]SelectableModelInfo) []SelectableModelInfo {
	result := make([]SelectableModelInfo, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}
