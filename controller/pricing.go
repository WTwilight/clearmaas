package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func filterPricingByUsableGroups(pricing []model.Pricing, usableGroup map[string]string) []model.Pricing {
	if len(pricing) == 0 {
		return pricing
	}
	if len(usableGroup) == 0 {
		return []model.Pricing{}
	}

	filtered := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		if common.StringsContains(item.EnableGroup, "all") {
			filtered = append(filtered, item)
			continue
		}
		for _, group := range item.EnableGroup {
			if _, ok := usableGroup[group]; ok {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

func GetPricing(c *gin.Context) {
	pricing := model.GetPricing()
	userId, exists := c.Get("id")
	usableGroup := map[string]string{}
	groupRatio := map[string]float64{}
	var err error
	for s, f := range ratio_setting.GetGroupRatioCopy() {
		groupRatio[s] = f
	}
	var group string
	if exists {
		user, err := model.GetUserCache(userId.(int))
		if err == nil {
			group = user.Group
			for g := range groupRatio {
				ratio, ok := ratio_setting.GetGroupGroupRatio(group, g)
				if ok {
					groupRatio[g] = ratio
				}
			}
		}
	}

	usableGroup = service.GetUserUsableGroups(group)
	pricing = filterPricingByUsableGroups(pricing, usableGroup)
	// check groupRatio contains usableGroup
	for g := range ratio_setting.GetGroupRatioCopy() {
		if _, ok := usableGroup[g]; !ok {
			delete(groupRatio, g)
		}
	}

	// Step 1: Only show models that are set in platform pricing sheets.
	// Build a map of models that exist in any platform pricing sheet.
	platformModelSheetMap := make(map[string]model.SheetInfo)
	var platformSheets []*model.EnterprisePricingSheet
	platformSheets, err = model.GetAllActivePricingSheetsByEnterpriseIdByType(model.EnterpriseTypePlatform)
	if err == nil && len(platformSheets) > 0 {
		for _, platformSheet := range platformSheets {
			var items []*model.EnterprisePricingItem
			items, err = model.GetPricingItemsBySheetId(platformSheet.Id)
			if err == nil {
				for _, item := range items {
					for _, modelName := range item.Models {
						platformModelSheetMap[modelName] = model.SheetInfo{
							SheetId:   platformSheet.Id,
							SheetName: platformSheet.Name,
						}
					}
				}
			}
		}
	}

	// Step 2: Filter pricing to only include models in platform sheets, and
	// determine enterprise sheet info for each user-visible model.
	enterpriseSheet := (*model.EnterprisePricingSheet)(nil)
	hasEnterprise := false
	if exists {
		enterpriseSheet, _ = service.GetUserActivePricingSheet(userId.(int))
		hasEnterprise = enterpriseSheet != nil
	}

	enterpriseModelSheetMap := make(map[string]model.SheetInfo)
	if hasEnterprise {
		items, err := model.GetPricingItemsBySheetId(enterpriseSheet.Id)
		if err == nil {
			for _, item := range items {
				for _, modelName := range item.Models {
					enterpriseModelSheetMap[modelName] = model.SheetInfo{
						SheetId:   enterpriseSheet.Id,
						SheetName: enterpriseSheet.Name,
					}
				}
			}
		}
	}

	filtered := make([]model.Pricing, 0, len(pricing))
	for _, item := range pricing {
		var sheetInfo model.SheetInfo
		var hasSheet bool

		// Enterprise sheet has priority when both enterprise and platform have this model.
		if hasEnterprise {
			if esi, ok := enterpriseModelSheetMap[item.ModelName]; ok {
				sheetInfo = esi
				hasSheet = true
			} else if psi, ok := platformModelSheetMap[item.ModelName]; ok {
				sheetInfo = psi
				hasSheet = true
			}
		} else {
			if psi, ok := platformModelSheetMap[item.ModelName]; ok {
				sheetInfo = psi
				hasSheet = true
			}
		}

		if !hasSheet {
			continue
		}

		item.RatioSource = "platform_pricing_sheet"
		if hasEnterprise {
			item.RatioSource = "enterprise_pricing_sheet"
		}

		// Fill DiscountRatio from the effective sheet (enterprise takes priority).
		if hasEnterprise {
			if ratio, found := service.GetModelDiscount(enterpriseSheet.Id, item.ModelName); found {
				item.DiscountRatio = ratio
			} else if ratio, found := service.GetModelDiscount(platformModelSheetMap[item.ModelName].SheetId, item.ModelName); found {
				item.DiscountRatio = ratio
			}
		} else {
			if ratio, found := service.GetModelDiscount(sheetInfo.SheetId, item.ModelName); found {
				item.DiscountRatio = ratio
			}
		}

		filtered = append(filtered, item)
	}
	pricing = filtered

	ratioSource := ""
	if hasEnterprise {
		ratioSource = "enterprise_pricing_sheet"
	} else if len(platformSheets) > 0 {
		ratioSource = "platform_pricing_sheet"
	}

	c.JSON(200, gin.H{
		"success":            true,
		"data":              pricing,
		"vendors":           model.GetVendors(),
		"group_ratio":       groupRatio,
		"usable_group":      usableGroup,
		"supported_endpoint": model.GetSupportedEndpointMap(),
		"auto_groups":       service.GetUserAutoGroup(group),
		"pricing_version":   "a42d372ccf0b5dd13ecf71203521f9d2",
		"ratio_source":      ratioSource,
	})
}

func ResetModelRatio(c *gin.Context) {
	defaultStr := ratio_setting.DefaultModelRatio2JSONString()
	err := model.UpdateOption("ModelRatio", defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = ratio_setting.UpdateModelRatioByJSONString(defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "重置模型倍率成功",
	})
}
