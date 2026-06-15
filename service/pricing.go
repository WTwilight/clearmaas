package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

type EffectivePricingData struct {
	Pricing     []model.Pricing
	GroupRatio  map[string]float64
	UsableGroup map[string]string
	AutoGroups  []string
	RatioSource string
	UserGroup   string
}

func GetEffectivePricingForUser(userID int, hasUser bool) EffectivePricingData {
	pricing := model.GetPricing()
	groupRatio := make(map[string]float64)
	for group, ratio := range ratio_setting.GetGroupRatioCopy() {
		groupRatio[group] = ratio
	}

	userGroup := ""
	if hasUser {
		if user, err := model.GetUserCache(userID); err == nil {
			userGroup = user.Group
			for group := range groupRatio {
				if ratio, ok := ratio_setting.GetGroupGroupRatio(userGroup, group); ok {
					groupRatio[group] = ratio
				}
			}
		}
	}

	usableGroup := GetUserUsableGroups(userGroup)
	pricing = filterEffectivePricingByUsableGroups(pricing, usableGroup)
	for group := range ratio_setting.GetGroupRatioCopy() {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}

	platformModelSheetMap := make(map[string]model.SheetInfo)
	platformSheets, err := model.GetAllActivePricingSheetsByEnterpriseIdByType(model.EnterpriseTypePlatform)
	if err == nil && len(platformSheets) > 0 {
		for _, platformSheet := range platformSheets {
			items, itemErr := model.GetPricingItemsBySheetId(platformSheet.Id)
			if itemErr != nil {
				continue
			}
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

	var enterpriseSheet *model.EnterprisePricingSheet
	hasEnterprise := false
	if hasUser {
		enterpriseSheet, _ = GetUserActivePricingSheet(userID)
		hasEnterprise = enterpriseSheet != nil
	}

	enterpriseModelSheetMap := make(map[string]model.SheetInfo)
	if hasEnterprise {
		items, itemErr := model.GetPricingItemsBySheetId(enterpriseSheet.Id)
		if itemErr == nil {
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
		sheetInfo, ratioSource, hasSheet := effectivePricingSheet(item.ModelName, platformModelSheetMap, enterpriseModelSheetMap, hasEnterprise)
		if !hasSheet {
			continue
		}
		item.RatioSource = ratioSource
		if ratio, found := GetModelDiscount(sheetInfo.SheetId, item.ModelName); found {
			item.DiscountRatio = ratio
		}
		filtered = append(filtered, item)
	}

	ratioSource := ""
	if hasEnterprise {
		ratioSource = "enterprise_pricing_sheet"
	} else if len(platformSheets) > 0 {
		ratioSource = "platform_pricing_sheet"
	}

	return EffectivePricingData{
		Pricing:     filtered,
		GroupRatio:  groupRatio,
		UsableGroup: usableGroup,
		AutoGroups:  GetUserAutoGroup(userGroup),
		RatioSource: ratioSource,
		UserGroup:   userGroup,
	}
}

func filterEffectivePricingByUsableGroups(pricing []model.Pricing, usableGroup map[string]string) []model.Pricing {
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

func effectivePricingSheet(modelName string, platformSheets map[string]model.SheetInfo, enterpriseSheets map[string]model.SheetInfo, hasEnterprise bool) (model.SheetInfo, string, bool) {
	if hasEnterprise {
		if sheetInfo, ok := enterpriseSheets[modelName]; ok {
			return sheetInfo, "enterprise_pricing_sheet", true
		}
		if sheetInfo, ok := platformSheets[modelName]; ok {
			return sheetInfo, "platform_pricing_sheet", true
		}
		return model.SheetInfo{}, "", false
	}
	if sheetInfo, ok := platformSheets[modelName]; ok {
		return sheetInfo, "platform_pricing_sheet", true
	}
	return model.SheetInfo{}, "", false
}
