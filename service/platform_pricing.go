package service

import (
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// SelectableModelInfo represents a model available for binding in the key creation/edit form.
type SelectableModelInfo struct {
	Model                  string  `json:"model"`
	QuotaType              int     `json:"quota_type"` // 0=ratio(per 1M tokens), 1=fixed(per request)
	InputOriginalPrice     float64 `json:"input_original_price"`  // ratio:type=model_ratio*2($/1M), price:type=model_price($/req)
	OutputOriginalPrice    float64 `json:"output_original_price"` // ratio:type=model_ratio*2*completion_ratio, price:type=0
	DiscountRatio          float64 `json:"discount_ratio"`
	InputDiscountedPrice  float64 `json:"input_discounted_price"`  // final price after discount
	OutputDiscountedPrice  float64 `json:"output_discounted_price"` // final price after discount
	VendorType             string  `json:"vendor_type"`
	Source                string  `json:"source"` // "enterprise" | "platform"
	SheetId               int     `json:"sheet_id"`
	SheetName             string  `json:"sheet_name"`
}

// GetSelectableModelsForUser returns the list of models available for binding in the key creation/edit form.
// This is the UNION of all platform pricing sheet models AND the user's enterprise pricing sheet models,
// with enterprise discount taking priority over platform for the same model.
// This is consistent with the model marketplace (/api/pricing) behavior.
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

	// Step 2: Collect models from the user's enterprise pricing sheet (if any).
	// Models ONLY in the enterprise sheet (not in any platform sheet) are also included.
	// This ensures the model list matches the model marketplace behavior.
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
						// Enterprise always takes priority for the same model
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

	// Step 3: Build union of platform + enterprise models.
	// If neither platform nor enterprise sheet has any models, return nothing.
	allModelMap := make(map[string]platformModelEntry)
	for modelName, entry := range platformModelMap {
		allModelMap[modelName] = entry
	}
	for modelName, entry := range enterpriseModelMap {
		allModelMap[modelName] = entry
	}
	if len(allModelMap) == 0 {
		return nil
	}

	// Step 4: Build result — enterprise discount takes priority for overlapping models.
	result := make([]SelectableModelInfo, 0, len(allModelMap))
	for modelName, entry := range allModelMap {
		vendorType := entry.VendorType
		discountValue := entry.DiscountValue
		sheetId := entry.SheetId
		sheetName := entry.SheetName
		source := "platform"

		// Enterprise takes priority over platform for the same model
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
