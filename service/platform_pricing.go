package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// SelectableModelInfo represents a model available for binding in the key creation/edit form.
type SelectableModelInfo struct {
	Model                 string  `json:"model"`
	QuotaType             int     `json:"quota_type"`            // 0=ratio(per 1M tokens), 1=fixed(per request)
	InputOriginalPrice    float64 `json:"input_original_price"`  // ratio:type=model_ratio*2($/1M), price:type=model_price($/req)
	OutputOriginalPrice   float64 `json:"output_original_price"` // ratio:type=model_ratio*2*completion_ratio, price:type=0
	DiscountRatio         float64 `json:"discount_ratio"`
	InputDiscountedPrice  float64 `json:"input_discounted_price"`  // final price after discount
	OutputDiscountedPrice float64 `json:"output_discounted_price"` // final price after discount
	VendorType            string  `json:"vendor_type"`
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
			Model:                 modelName,
			QuotaType:             quotaType,
			InputOriginalPrice:    inputOriginal,
			OutputOriginalPrice:   outputOriginal,
			DiscountRatio:         discountValue,
			InputDiscountedPrice:  inputDiscounted,
			OutputDiscountedPrice: outputDiscounted,
			VendorType:            vendorType,
			Source:                source,
			SheetId:               sheetId,
			SheetName:             sheetName,
		})
	}

	return result
}

// ValidateTokenPricingModelBindings ensures a token's model whitelist and pricing
// bindings match the pricing sheets the user is allowed to use.
func ValidateTokenPricingModelBindings(userId int, modelLimits string, bindings []model.TokenPricingModelBindingInput) error {
	limitSet := parseModelLimitSet(modelLimits)
	if len(bindings) != len(limitSet) {
		return fmt.Errorf("model limits and pricing bindings are inconsistent")
	}

	selectableModels := GetSelectableModelsForUser(userId)
	selectableByModel := make(map[string]SelectableModelInfo, len(selectableModels))
	for _, selectable := range selectableModels {
		selectableByModel[selectable.Model] = selectable
	}

	seen := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		modelName := strings.TrimSpace(binding.Model)
		if modelName == "" || binding.PricingSheetId <= 0 {
			return fmt.Errorf("invalid pricing binding")
		}
		if _, ok := limitSet[modelName]; !ok {
			return fmt.Errorf("pricing binding model %q is not in model limits", modelName)
		}
		if _, ok := seen[modelName]; ok {
			return fmt.Errorf("duplicate pricing binding for model %q", modelName)
		}
		seen[modelName] = struct{}{}

		selectable, ok := selectableByModel[modelName]
		if !ok {
			return fmt.Errorf("model %q is not available for current user pricing", modelName)
		}
		if selectable.SheetId != binding.PricingSheetId {
			return fmt.Errorf("model %q is not available in pricing sheet %d", modelName, binding.PricingSheetId)
		}
		if item, err := model.GetPricingItemBySheetIdAndModelName(binding.PricingSheetId, modelName); err != nil {
			return err
		} else if item == nil {
			return fmt.Errorf("model %q is not configured in pricing sheet %d", modelName, binding.PricingSheetId)
		}
	}

	for modelName := range limitSet {
		if _, ok := seen[modelName]; !ok {
			return fmt.Errorf("model %q is missing pricing binding", modelName)
		}
	}
	return nil
}

func parseModelLimitSet(modelLimits string) map[string]struct{} {
	limitSet := make(map[string]struct{})
	for _, modelName := range strings.Split(modelLimits, ",") {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" {
			continue
		}
		limitSet[modelName] = struct{}{}
	}
	return limitSet
}

// GetGroupedSelectableModelsForUser returns selectable pricing models grouped by model-square parents.
func GetGroupedSelectableModelsForUser(userId int) (dto.AvailablePricingModelGroupsData, error) {
	selectableModels := GetSelectableModelsForUser(userId)
	if len(selectableModels) == 0 {
		return dto.AvailablePricingModelGroupsData{
			Models: []dto.AvailablePricingModelGroup{},
			Total:  0,
		}, nil
	}

	selectableByModel := make(map[string]SelectableModelInfo, len(selectableModels))
	for _, item := range selectableModels {
		selectableByModel[item.Model] = item
	}

	squareData, err := GetModelSquareData(ModelSquareQuery{
		UserID:  userId,
		HasUser: true,
	})
	if err != nil {
		groups := buildFallbackAvailablePricingModelGroups(selectableModels)
		return dto.AvailablePricingModelGroupsData{
			Models: groups,
			Total:  len(groups),
		}, nil
	}

	groups := make([]dto.AvailablePricingModelGroup, 0, len(squareData.Models))
	groupIndexByName := make(map[string]int, len(squareData.Models))
	usedSelectableModels := make(map[string]bool, len(selectableModels))
	for _, parent := range squareData.Models {
		versions := make([]dto.AvailablePricingModelVersion, 0, len(parent.Versions))
		for _, version := range parent.Versions {
			selectable, ok := selectableByModel[version.ModelName]
			if !ok {
				continue
			}
			usedSelectableModels[selectable.Model] = true
			versions = upsertAvailablePricingModelVersion(versions, buildAvailablePricingModelVersion(version, selectable))
		}
		if len(versions) == 0 {
			continue
		}
		groups = append(groups, dto.AvailablePricingModelGroup{
			Name:              parent.Name,
			DisplayName:       parent.DisplayName,
			Icon:              parent.Icon,
			VendorID:          parent.VendorID,
			Vendor:            parent.VendorName,
			VendorName:        parent.VendorName,
			VendorIcon:        parent.VendorIcon,
			Tags:              parent.Tags,
			Context:           parent.ContextTokens,
			ContextTokens:     parent.ContextTokens,
			MaxOutput:         parent.MaxOutput,
			BestDiscountRatio: bestSelectableVersionDiscountRatio(versions),
			Versions:          versions,
		})
		groupIndexByName[parent.Name] = len(groups) - 1
	}

	leftoverSelectableModels := make([]SelectableModelInfo, 0)
	for _, selectable := range selectableModels {
		if usedSelectableModels[selectable.Model] {
			continue
		}
		leftoverSelectableModels = append(leftoverSelectableModels, selectable)
	}
	for _, fallbackGroup := range buildFallbackAvailablePricingModelGroups(leftoverSelectableModels) {
		if index, ok := groupIndexByName[fallbackGroup.Name]; ok {
			for _, version := range fallbackGroup.Versions {
				groups[index].Versions = upsertAvailablePricingModelVersion(groups[index].Versions, version)
			}
			groups[index].BestDiscountRatio = bestSelectableVersionDiscountRatio(groups[index].Versions)
			continue
		}
		groups = append(groups, fallbackGroup)
		groupIndexByName[fallbackGroup.Name] = len(groups) - 1
	}

	return dto.AvailablePricingModelGroupsData{
		Models: groups,
		Total:  len(groups),
	}, nil
}

func buildAvailablePricingModelVersion(version dto.ModelVersion, selectable SelectableModelInfo) dto.AvailablePricingModelVersion {
	return dto.AvailablePricingModelVersion{
		Model:          selectable.Model,
		UpstreamKey:    version.UpstreamKey,
		ChannelID:      version.ChannelID,
		ChannelName:    version.ChannelName,
		ChannelTags:    version.ChannelTags,
		PricingSheetID: selectable.SheetId,
		SheetID:        selectable.SheetId,
		SheetName:      selectable.SheetName,
		Source:         selectable.Source,
		VendorType:     selectable.VendorType,
		QuotaType:      selectable.QuotaType,
		Original: dto.PricePair{
			Input:  selectable.InputOriginalPrice,
			Output: selectable.OutputOriginalPrice,
		},
		InputOriginalPrice:  selectable.InputOriginalPrice,
		OutputOriginalPrice: selectable.OutputOriginalPrice,
		Discounted: dto.PricePair{
			Input:  selectable.InputDiscountedPrice,
			Output: selectable.OutputDiscountedPrice,
		},
		DiscountRatio:         selectable.DiscountRatio,
		InputDiscountedPrice:  selectable.InputDiscountedPrice,
		OutputDiscountedPrice: selectable.OutputDiscountedPrice,
	}
}

func upsertAvailablePricingModelVersion(versions []dto.AvailablePricingModelVersion, next dto.AvailablePricingModelVersion) []dto.AvailablePricingModelVersion {
	for index, current := range versions {
		if current.Model != next.Model {
			continue
		}
		merged := current
		merged.ChannelTags = mergeAvailablePricingChannelTags(current.ChannelTags, next.ChannelTags)
		if merged.ChannelName == "" {
			merged.ChannelName = next.ChannelName
		} else if next.ChannelName != "" && next.ChannelName != merged.ChannelName {
			merged.ChannelTags = mergeAvailablePricingChannelTags(merged.ChannelTags, []string{next.ChannelName})
		}
		if shouldReplaceAvailablePricingVersion(current, next) {
			merged.UpstreamKey = next.UpstreamKey
			merged.ChannelID = next.ChannelID
			merged.ChannelName = next.ChannelName
			merged.PricingSheetID = next.PricingSheetID
			merged.SheetID = next.SheetID
			merged.SheetName = next.SheetName
			merged.Source = next.Source
			merged.VendorType = next.VendorType
			merged.QuotaType = next.QuotaType
			merged.Original = next.Original
			merged.InputOriginalPrice = next.InputOriginalPrice
			merged.OutputOriginalPrice = next.OutputOriginalPrice
			merged.Discounted = next.Discounted
			merged.DiscountRatio = next.DiscountRatio
			merged.InputDiscountedPrice = next.InputDiscountedPrice
			merged.OutputDiscountedPrice = next.OutputDiscountedPrice
		}
		versions[index] = merged
		return versions
	}
	return append(versions, next)
}

func mergeAvailablePricingChannelTags(current []string, next []string) []string {
	seen := make(map[string]struct{}, len(current)+len(next))
	result := make([]string, 0, len(current)+len(next))
	for _, tag := range append(current, next...) {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
	}
	return result
}

func shouldReplaceAvailablePricingVersion(current, next dto.AvailablePricingModelVersion) bool {
	if current.PricingSheetID != next.PricingSheetID {
		return next.Source == "enterprise"
	}
	if current.DiscountRatio > 0 && next.DiscountRatio > 0 && current.DiscountRatio != next.DiscountRatio {
		return next.DiscountRatio < current.DiscountRatio
	}
	if current.ChannelID <= 0 {
		return next.ChannelID > 0
	}
	return false
}

type availablePricingFallbackRoute struct {
	UpstreamKey  string
	ChannelID    int
	ChannelName  string
	ChannelTags  []string
	Priority     int64
	Status       int
	HasRouteInfo bool
}

func buildFallbackAvailablePricingModelGroups(selectableModels []SelectableModelInfo) []dto.AvailablePricingModelGroup {
	if len(selectableModels) == 0 {
		return []dto.AvailablePricingModelGroup{}
	}
	routeByModel := availablePricingFallbackRoutes(selectableModels)
	metaByName, _ := getModelSquareMetadata()
	vendors := model.GetVendors()
	vendorByID := make(map[int]model.PricingVendor, len(vendors))
	for _, vendor := range vendors {
		vendorByID[vendor.ID] = vendor
	}

	groups := make([]dto.AvailablePricingModelGroup, 0, len(selectableModels))
	groupIndexByName := make(map[string]int, len(selectableModels))
	for _, selectable := range selectableModels {
		route := routeByModel[selectable.Model]
		groupName := route.UpstreamKey
		if groupName == "" {
			groupName = inferModelSquareParentName(selectable.Model)
		} else {
			groupName = inferModelSquareParentName(groupName)
		}
		version := buildAvailablePricingModelFallbackVersion(selectable, route)
		if index, ok := groupIndexByName[groupName]; ok {
			groups[index].Versions = upsertAvailablePricingModelVersion(groups[index].Versions, version)
			groups[index].BestDiscountRatio = bestSelectableVersionDiscountRatio(groups[index].Versions)
			continue
		}
		group := buildFallbackAvailablePricingModelGroup(groupName, selectable, version, metaByName, vendorByID)
		groups = append(groups, group)
		groupIndexByName[groupName] = len(groups) - 1
	}
	return groups
}

func availablePricingFallbackRoutes(selectableModels []SelectableModelInfo) map[string]availablePricingFallbackRoute {
	selectableNames := make(map[string]struct{}, len(selectableModels))
	for _, selectable := range selectableModels {
		selectableNames[selectable.Model] = struct{}{}
	}

	var channels []model.Channel
	err := model.DB.
		Select("id", "name", "status", "priority", "models", "model_mapping", "tag").
		Where("models <> ''").
		Find(&channels).Error
	if err != nil {
		return map[string]availablePricingFallbackRoute{}
	}

	routeByModel := make(map[string]availablePricingFallbackRoute, len(selectableModels))
	for _, channel := range channels {
		var mapping map[string]string
		if err := common.UnmarshalJsonStr(channel.GetModelMapping(), &mapping); err != nil {
			mapping = map[string]string{}
		}
		normalized := normalizeModelSquareMapping(mapping)
		for _, channelModel := range splitChannelModels(channel.Models) {
			if _, ok := selectableNames[channelModel]; !ok {
				continue
			}
			upstreamKey := resolveModelSquareMappingTarget(channelModel, normalized)
			if upstreamKey == "" {
				upstreamKey = channelModel
			}
			upstreamKey = inferModelSquareParentName(upstreamKey)
			next := availablePricingFallbackRoute{
				UpstreamKey:  upstreamKey,
				ChannelID:    channel.Id,
				ChannelName:  channel.Name,
				ChannelTags:  splitModelTags(ptrString(channel.Tag)),
				Priority:     channel.GetPriority(),
				Status:       channel.Status,
				HasRouteInfo: true,
			}
			if shouldReplaceAvailablePricingFallbackRoute(routeByModel[channelModel], next) {
				routeByModel[channelModel] = next
			}
		}
	}
	return routeByModel
}

func shouldReplaceAvailablePricingFallbackRoute(current, next availablePricingFallbackRoute) bool {
	if !current.HasRouteInfo {
		return true
	}
	if current.Status != next.Status {
		return next.Status == common.ChannelStatusEnabled
	}
	if current.Priority != next.Priority {
		return next.Priority > current.Priority
	}
	return next.ChannelID < current.ChannelID
}

func buildFallbackAvailablePricingModelGroup(
	groupName string,
	selectable SelectableModelInfo,
	version dto.AvailablePricingModelVersion,
	metaByName map[string]model.Model,
	vendorByID map[int]model.PricingVendor,
) dto.AvailablePricingModelGroup {
	meta := metaByName[groupName]
	tags := splitModelTags(meta.Tags)
	vendorName := selectable.VendorType
	vendorIcon := ""
	vendorID := meta.VendorID
	if vendor, ok := vendorByID[vendorID]; ok {
		vendorName = vendor.Name
		vendorIcon = vendor.Icon
	}
	displayName := groupName
	if groupName != selectable.Model || meta.Id != 0 {
		displayName = displayModelName(groupName)
	}
	return dto.AvailablePricingModelGroup{
		Name:              groupName,
		DisplayName:       displayName,
		Icon:              meta.Icon,
		VendorID:          vendorID,
		Vendor:            vendorName,
		VendorName:        vendorName,
		VendorIcon:        vendorIcon,
		Tags:              tags,
		Context:           modelSquareContextTokens(groupName, tags),
		ContextTokens:     modelSquareContextTokens(groupName, tags),
		MaxOutput:         modelSquareMaxOutputTokens(groupName),
		BestDiscountRatio: bestSelectableVersionDiscountRatio([]dto.AvailablePricingModelVersion{version}),
		Versions:          []dto.AvailablePricingModelVersion{version},
	}
}

func buildAvailablePricingModelFallbackVersion(selectable SelectableModelInfo, route availablePricingFallbackRoute) dto.AvailablePricingModelVersion {
	return dto.AvailablePricingModelVersion{
		Model:          selectable.Model,
		UpstreamKey:    route.UpstreamKey,
		ChannelID:      route.ChannelID,
		ChannelName:    route.ChannelName,
		ChannelTags:    route.ChannelTags,
		PricingSheetID: selectable.SheetId,
		SheetID:        selectable.SheetId,
		SheetName:      selectable.SheetName,
		Source:         selectable.Source,
		VendorType:     selectable.VendorType,
		QuotaType:      selectable.QuotaType,
		Original: dto.PricePair{
			Input:  selectable.InputOriginalPrice,
			Output: selectable.OutputOriginalPrice,
		},
		InputOriginalPrice:  selectable.InputOriginalPrice,
		OutputOriginalPrice: selectable.OutputOriginalPrice,
		Discounted: dto.PricePair{
			Input:  selectable.InputDiscountedPrice,
			Output: selectable.OutputDiscountedPrice,
		},
		DiscountRatio:         selectable.DiscountRatio,
		InputDiscountedPrice:  selectable.InputDiscountedPrice,
		OutputDiscountedPrice: selectable.OutputDiscountedPrice,
	}
}

func bestSelectableVersionDiscountRatio(versions []dto.AvailablePricingModelVersion) float64 {
	var best float64
	for _, version := range versions {
		if version.DiscountRatio <= 0 {
			continue
		}
		if best == 0 || version.DiscountRatio < best {
			best = version.DiscountRatio
		}
	}
	return best
}

// platformModelEntry holds model entry info for platform/enterprise pricing aggregation.
type platformModelEntry struct {
	VendorType    string
	DiscountValue float64
	SheetId       int
	SheetName     string
}
