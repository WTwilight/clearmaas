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

	// Determine pricing context and ratio_source for each model
	var ratioSource string
	if exists {
		uid := userId.(int)
		sheet, err := service.GetUserActivePricingSheet(uid)
		if err == nil && sheet != nil {
			// User has an active enterprise pricing sheet
			for i := range pricing {
				ratio, found := service.GetModelDiscount(sheet.Id, pricing[i].ModelName)
				if found {
					pricing[i].DiscountRatio = ratio
					pricing[i].RatioSource = "enterprise_pricing_sheet"
				}
			}
		ratioSource = "enterprise_pricing_sheet"
		} else {
		// No enterprise pricing sheet → check all active platform pricing sheets
		// When multiple sheets configure the same model, use the lowest discount_value (best discount for user).
		platformSheets, err := model.GetAllActivePricingSheetsByEnterpriseIdByType(model.EnterpriseTypePlatform)
		if err == nil && len(platformSheets) > 0 {
			for _, platformSheet := range platformSheets {
				for i := range pricing {
					// Only set if not already set by enterprise sheet
					if pricing[i].RatioSource == "" {
						ratio, found := service.GetModelDiscount(platformSheet.Id, pricing[i].ModelName)
						if found {
							// Only update if this sheet has a better (lower) discount
							if pricing[i].DiscountRatio == 0 || ratio < pricing[i].DiscountRatio {
								pricing[i].DiscountRatio = ratio
								pricing[i].RatioSource = "platform_pricing_sheet"
							}
						}
					}
				}
			}
			ratioSource = "platform_pricing_sheet"
		}
	}
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
