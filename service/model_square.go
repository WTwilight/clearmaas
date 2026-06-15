package service

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

type ModelSquareQuery struct {
	Keyword  string
	VendorID string
	Tag      string
	UserID   int
	HasUser  bool
}

type modelSquareDraft struct {
	dto.ModelSquareItem
	endpointSet map[string]struct{}
	updatedTime int64
}

func GetModelSquareData(query ModelSquareQuery) (dto.ModelSquareData, error) {
	effectivePricing := GetEffectivePricingForUser(query.UserID, query.HasUser).Pricing
	pricingByModel := make(map[string]model.Pricing, len(effectivePricing))
	for _, item := range effectivePricing {
		pricingByModel[item.ModelName] = item
	}
	effectivePriceByModel := buildModelSquarePricingData(effectivePricing)

	vendors := model.GetVendors()
	vendorByID := make(map[int]model.PricingVendor, len(vendors))
	dtoVendors := make([]dto.ModelVendor, 0, len(vendors))
	for _, vendor := range vendors {
		vendorByID[vendor.ID] = vendor
		dtoVendors = append(dtoVendors, dto.ModelVendor{
			ID:   vendor.ID,
			Name: vendor.Name,
			Icon: vendor.Icon,
		})
	}

	metaByName, err := getModelSquareMetadata()
	if err != nil {
		return dto.ModelSquareData{}, err
	}

	drafts := make(map[string]*modelSquareDraft)
	for _, item := range effectivePricing {
		parentName := inferModelSquareParentName(item.ModelName)
		if parentName == item.ModelName {
			ensureModelSquareDraft(drafts, item.ModelName, item, effectivePriceByModel, metaByName, vendorByID)
			continue
		}
		draft := ensureModelSquareDraftWithOfficialRoute(drafts, parentName, item, effectivePriceByModel, metaByName, vendorByID, false)
		version := dto.ModelVersion{
			ModelName:       item.ModelName,
			UpstreamKey:     parentName,
			Status:          "enabled",
			ModeDescription: modelVersionDescription(item.ModelName, parentName),
			PriceInfo:       priceInfoForVersion(item.ModelName, parentName, item, effectivePriceByModel),
		}
		applyPriceInfoToModelVersion(&version)
		draft.Versions = upsertModelVersion(draft.Versions, version)
		addEndpoints(draft.endpointSet, item)
	}

	mappedAliases, err := attachMappingVersions(drafts, pricingByModel, effectivePriceByModel, metaByName, vendorByID)
	if err != nil {
		return dto.ModelSquareData{}, err
	}
	metricByModel, metricByVersion, _ := getModelSquareMetricSummaries()

	models := make([]dto.ModelSquareItem, 0, len(drafts))
	tagSet := make(map[string]struct{})
	for _, draft := range drafts {
		if _, ok := mappedAliases[draft.Name]; ok {
			continue
		}
		if len(draft.Versions) == 0 {
			continue
		}
		draft.Versions = normalizeModelSquareVersions(draft.Versions)
		applyModelSquareVersionMetrics(draft, metricByModel, metricByVersion)
		sort.SliceStable(draft.Versions, func(i, j int) bool {
			if draft.Versions[i].Priority == draft.Versions[j].Priority {
				return draft.Versions[i].ModelName < draft.Versions[j].ModelName
			}
			return draft.Versions[i].Priority > draft.Versions[j].Priority
		})
		draft.VersionCount = len(draft.Versions)
		draft.Capabilities = buildCapabilities(draft.endpointSet)
		finalizeModelSquareMetadata(draft)
		for _, tag := range draft.Tags {
			if tag != "" {
				tagSet[tag] = struct{}{}
			}
		}
		item := draft.ModelSquareItem
		if matchesModelSquareQuery(item, query) {
			models = append(models, item)
		}
	}

	sort.SliceStable(models, func(i, j int) bool {
		return compareModelSquareItemsByVendorLatest(models[i], models[j]) < 0
	})

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	return dto.ModelSquareData{
		Models:  models,
		Vendors: dtoVendors,
		Tags:    tags,
		Total:   len(models),
	}, nil
}

func GetModelSquareDetail(modelName string, userID int, hasUser bool) (dto.ModelSquareItem, error) {
	modelName = strings.TrimPrefix(modelName, "/")
	data, err := GetModelSquareData(ModelSquareQuery{UserID: userID, HasUser: hasUser})
	if err != nil {
		return dto.ModelSquareItem{}, err
	}
	for _, item := range data.Models {
		if item.Name == modelName {
			return item, nil
		}
	}
	return dto.ModelSquareItem{}, fmt.Errorf("model %q not found", modelName)
}

func getModelSquareMetadata() (map[string]model.Model, error) {
	var models []model.Model
	if err := model.DB.Find(&models).Error; err != nil {
		return nil, err
	}
	result := make(map[string]model.Model, len(models))
	for _, item := range models {
		if item.Status == 1 {
			result[item.ModelName] = item
		}
	}
	return result, nil
}

func attachMappingVersions(
	drafts map[string]*modelSquareDraft,
	pricingByModel map[string]model.Pricing,
	effectivePriceByModel map[string]*dto.PriceInfo,
	metaByName map[string]model.Model,
	vendorByID map[int]model.PricingVendor,
) (map[string]struct{}, error) {
	var channels []model.Channel
	if err := model.DB.
		Select("id", "name", "type", "status", "priority", "response_time", "models", "model_mapping", "tag").
		Where("models <> ''").
		Find(&channels).Error; err != nil {
		return nil, err
	}

	mappedAliases := make(map[string]struct{})
	for _, channel := range channels {
		var mapping map[string]string
		if err := common.UnmarshalJsonStr(channel.GetModelMapping(), &mapping); err != nil {
			mapping = map[string]string{}
		}
		normalized := normalizeModelSquareMapping(mapping)
		for _, channelModel := range splitChannelModels(channel.Models) {
			modelName := strings.TrimSpace(channelModel)
			if modelName == "" {
				continue
			}
			if _, ok := pricingByModel[modelName]; !ok {
				continue
			}
			modelName = strings.TrimSpace(modelName)
			upstreamKey := resolveModelSquareMappingTarget(modelName, normalized)
			if modelName == "" || upstreamKey == "" {
				continue
			}
			upstreamKey = inferModelSquareParentName(upstreamKey)
			if modelName != upstreamKey {
				mappedAliases[modelName] = struct{}{}
			}
			sourcePricing, hasSourcePricing := pricingByModel[modelName]
			upstreamPricing, hasUpstreamPricing := pricingByModel[upstreamKey]
			pricing := upstreamPricing
			if !hasUpstreamPricing {
				if hasSourcePricing {
					pricing = sourcePricing
				} else {
					pricing = model.Pricing{ModelName: upstreamKey}
				}
			}
			versionPricing := pricing
			if hasSourcePricing {
				versionPricing = sourcePricing
			}
			draft := ensureModelSquareDraftWithOfficialRoute(drafts, upstreamKey, pricing, effectivePriceByModel, metaByName, vendorByID, hasUpstreamPricing)
			versionPriceInfo := priceInfoForVersion(modelName, upstreamKey, versionPricing, effectivePriceByModel)
			version := dto.ModelVersion{
				ModelName:       modelName,
				UpstreamKey:     upstreamKey,
				ChannelID:       channel.Id,
				ChannelName:     channel.Name,
				ChannelTags:     splitModelTags(ptrString(channel.Tag)),
				ChannelType:     channel.Type,
				Status:          channelStatusLabel(channel.Status),
				Priority:        int(channel.GetPriority()),
				ModeDescription: modelVersionDescription(modelName, upstreamKey),
				PriceInfo:       versionPriceInfo,
			}
			applyPriceInfoToModelVersion(&version)
			if channel.ResponseTime > 0 {
				version.LatencySeconds = roundFloat(float64(channel.ResponseTime)/1000, 2)
			}
			if channel.Status == common.ChannelStatusEnabled {
				version.SuccessRate = 100
			}
			draft.Versions = upsertModelVersion(draft.Versions, version)
			addEndpoints(draft.endpointSet, versionPricing)
		}
	}
	return mappedAliases, nil
}

func ensureModelSquareDraft(
	drafts map[string]*modelSquareDraft,
	name string,
	pricing model.Pricing,
	effectivePriceByModel map[string]*dto.PriceInfo,
	metaByName map[string]model.Model,
	vendorByID map[int]model.PricingVendor,
) *modelSquareDraft {
	return ensureModelSquareDraftWithOfficialRoute(drafts, name, pricing, effectivePriceByModel, metaByName, vendorByID, true)
}

func ensureModelSquareDraftWithOfficialRoute(
	drafts map[string]*modelSquareDraft,
	name string,
	pricing model.Pricing,
	effectivePriceByModel map[string]*dto.PriceInfo,
	metaByName map[string]model.Model,
	vendorByID map[int]model.PricingVendor,
	includeOfficialRoute bool,
) *modelSquareDraft {
	if draft, ok := drafts[name]; ok {
		if draft.PriceInfo == nil {
			draft.PriceInfo = priceInfoForModel(name, pricing, effectivePriceByModel)
		}
		if includeOfficialRoute {
			priceInfo := priceInfoForModel(name, pricing, effectivePriceByModel)
			version := dto.ModelVersion{
				ModelName:       name,
				UpstreamKey:     name,
				Status:          "enabled",
				ModeDescription: "Official route",
				PriceInfo:       priceInfo,
			}
			applyPriceInfoToModelVersion(&version)
			draft.Versions = upsertModelVersion(draft.Versions, version)
		}
		addEndpoints(draft.endpointSet, pricing)
		return draft
	}

	meta, hasMeta := metaByName[name]
	vendorID := pricing.VendorID
	description := pricing.Description
	icon := pricing.Icon
	tags := splitModelTags(pricing.Tags)
	id := 0
	if hasMeta {
		id = meta.Id
		if meta.Description != "" {
			description = meta.Description
		}
		if meta.Icon != "" {
			icon = meta.Icon
		}
		if meta.Tags != "" {
			tags = splitModelTags(meta.Tags)
		}
		if meta.VendorID != 0 {
			vendorID = meta.VendorID
		}
	}

	vendor := vendorByID[vendorID]
	versions := make([]dto.ModelVersion, 0, 1)
	if includeOfficialRoute {
		priceInfo := priceInfoForModel(name, pricing, effectivePriceByModel)
		version := dto.ModelVersion{
			ModelName:       name,
			UpstreamKey:     name,
			Status:          "enabled",
			ModeDescription: "Official route",
			PriceInfo:       priceInfo,
		}
		applyPriceInfoToModelVersion(&version)
		versions = append(versions, version)
	}

	draft := &modelSquareDraft{
		ModelSquareItem: dto.ModelSquareItem{
			ID:            id,
			Name:          name,
			DisplayName:   displayModelName(name),
			Description:   description,
			Icon:          icon,
			VendorID:      vendorID,
			VendorName:    vendor.Name,
			VendorIcon:    vendor.Icon,
			Tags:          tags,
			ContextTokens: modelSquareContextTokens(name, tags),
			MaxOutput:     modelSquareMaxOutputTokens(name),
			PriceInfo:     priceInfoForModel(name, pricing, effectivePriceByModel),
			Versions:      versions,
			UpdatedTime:   updatedTimeFromMeta(meta, hasMeta),
		},
		endpointSet: make(map[string]struct{}),
		updatedTime: updatedTimeFromMeta(meta, hasMeta),
	}
	addEndpoints(draft.endpointSet, pricing)
	drafts[name] = draft
	return draft
}

func normalizeModelSquareVersions(versions []dto.ModelVersion) []dto.ModelVersion {
	routedByName := make(map[string]struct{})
	for _, version := range versions {
		if version.ChannelID != 0 {
			routedByName[version.ModelName] = struct{}{}
		}
	}
	if len(routedByName) == 0 {
		return versions
	}
	result := make([]dto.ModelVersion, 0, len(versions))
	for _, version := range versions {
		if version.ChannelID == 0 {
			if _, ok := routedByName[version.ModelName]; ok {
				continue
			}
		}
		result = append(result, version)
	}
	return result
}

type modelSquareMetricSummary struct {
	AvgLatencyMs int64
	SuccessRate  float64
	AvgTps       float64
	RequestCount int64
	HasMetrics   bool
}

type modelSquareVersionMetricKey struct {
	ChannelID int
	ModelName string
}

type modelSquareMetricTotals struct {
	requestCount   int64
	successCount   int64
	totalLatencyMs int64
	outputTokens   int64
	generationMs   int64
}

func getModelSquareMetricSummaries() (map[string]modelSquareMetricSummary, map[modelSquareVersionMetricKey]modelSquareMetricSummary, error) {
	summaries := make(map[string]modelSquareMetricSummary)
	result, err := perfmetrics.QuerySummaryAll(24)
	if err != nil {
		common.SysError("failed to query model square perf metrics: " + err.Error())
	} else {
		for _, item := range result.Models {
			summaries[item.ModelName] = modelSquareMetricSummary{
				AvgLatencyMs: item.AvgLatencyMs,
				SuccessRate:  item.SuccessRate,
				AvgTps:       item.AvgTps,
				RequestCount: item.RequestCount,
				HasMetrics:   true,
			}
		}
	}
	versionSummaries := mergeModelSquareLogSummaries(summaries)
	return summaries, versionSummaries, nil
}

func mergeModelSquareLogSummaries(summaries map[string]modelSquareMetricSummary) map[modelSquareVersionMetricKey]modelSquareMetricSummary {
	startTs := common.GetTimestamp() - 24*3600
	var rows []struct {
		ChannelID        int
		ModelName        string
		Type             int
		RequestCount     int64
		CompletionTokens int64
		TotalUseTime     int64
	}
	err := model.LOG_DB.
		Model(&model.Log{}).
		Select("channel_id, model_name, type, COUNT(*) as request_count, SUM(completion_tokens) as completion_tokens, SUM(use_time) as total_use_time").
		Where("created_at >= ? AND type IN ?", startTs, []int{model.LogTypeConsume, model.LogTypeError}).
		Group("channel_id, model_name, type").
		Scan(&rows).Error
	if err != nil {
		common.SysError("failed to query model square log metrics: " + err.Error())
		return nil
	}
	byModel := make(map[string]modelSquareMetricTotals)
	byVersion := make(map[modelSquareVersionMetricKey]modelSquareMetricTotals)
	for _, row := range rows {
		modelName := strings.TrimSpace(row.ModelName)
		if modelName == "" {
			continue
		}
		total := byModel[modelName]
		total.requestCount += row.RequestCount
		if row.Type == model.LogTypeConsume {
			total.successCount += row.RequestCount
		}
		if row.TotalUseTime > 0 {
			latencyMs := row.TotalUseTime * 1000
			total.totalLatencyMs += latencyMs
			total.generationMs += latencyMs
		}
		if row.CompletionTokens > 0 {
			total.outputTokens += row.CompletionTokens
		}
		byModel[modelName] = total

		if row.ChannelID > 0 {
			key := modelSquareVersionMetricKey{
				ChannelID: row.ChannelID,
				ModelName: modelName,
			}
			versionTotal := byVersion[key]
			versionTotal.requestCount += row.RequestCount
			if row.Type == model.LogTypeConsume {
				versionTotal.successCount += row.RequestCount
			}
			if row.TotalUseTime > 0 {
				latencyMs := row.TotalUseTime * 1000
				versionTotal.totalLatencyMs += latencyMs
				versionTotal.generationMs += latencyMs
			}
			if row.CompletionTokens > 0 {
				versionTotal.outputTokens += row.CompletionTokens
			}
			byVersion[key] = versionTotal
		}
	}
	for modelName, total := range byModel {
		if total.requestCount == 0 {
			continue
		}
		existing := summaries[modelName]
		if existing.RequestCount == 0 {
			existing.RequestCount = total.requestCount
		}
		if !existing.HasMetrics {
			existing.SuccessRate = roundFloat(float64(total.successCount)/float64(total.requestCount)*100, 2)
		}
		if existing.AvgLatencyMs == 0 && total.totalLatencyMs > 0 {
			existing.AvgLatencyMs = total.totalLatencyMs / total.requestCount
		}
		if existing.AvgTps == 0 && total.generationMs > 0 {
			existing.AvgTps = roundFloat(float64(total.outputTokens)/(float64(total.generationMs)/1000), 2)
		}
		existing.HasMetrics = true
		summaries[modelName] = existing
	}
	versionSummaries := make(map[modelSquareVersionMetricKey]modelSquareMetricSummary, len(byVersion))
	for key, total := range byVersion {
		summary := buildModelSquareMetricSummary(total)
		if summary.HasMetrics {
			versionSummaries[key] = summary
		}
	}
	return versionSummaries
}

func applyModelSquareVersionMetrics(draft *modelSquareDraft, summaries map[string]modelSquareMetricSummary, versionSummaries map[modelSquareVersionMetricKey]modelSquareMetricSummary) {
	if draft == nil {
		return
	}
	var requestCount int64
	for index := range draft.Versions {
		version := &draft.Versions[index]
		if version.Status == "enabled" && version.SuccessRate == 0 {
			version.SuccessRate = 100
		}
		summary, ok := modelSquareVersionMetric(version, summaries, versionSummaries)
		if !ok {
			continue
		}
		version.RequestCount = summary.RequestCount
		requestCount += summary.RequestCount
		if summary.AvgTps > 0 {
			version.TPS = summary.AvgTps
		}
		if summary.AvgLatencyMs > 0 {
			version.LatencySeconds = roundFloat(float64(summary.AvgLatencyMs)/1000, 2)
		}
		if summary.HasMetrics {
			version.SuccessRate = summary.SuccessRate
		}
	}
	draft.RequestCount = requestCount
}

func buildModelSquareMetricSummary(total modelSquareMetricTotals) modelSquareMetricSummary {
	if total.requestCount == 0 {
		return modelSquareMetricSummary{}
	}
	summary := modelSquareMetricSummary{
		RequestCount: total.requestCount,
		SuccessRate:  roundFloat(float64(total.successCount)/float64(total.requestCount)*100, 2),
		HasMetrics:   true,
	}
	if total.totalLatencyMs > 0 {
		summary.AvgLatencyMs = total.totalLatencyMs / total.requestCount
	}
	if total.outputTokens > 0 && total.generationMs > 0 {
		summary.AvgTps = roundFloat(float64(total.outputTokens)/(float64(total.generationMs)/1000), 2)
	}
	return summary
}

func modelSquareVersionMetric(version *dto.ModelVersion, summaries map[string]modelSquareMetricSummary, versionSummaries map[modelSquareVersionMetricKey]modelSquareMetricSummary) (modelSquareMetricSummary, bool) {
	if version == nil {
		return modelSquareMetricSummary{}, false
	}
	if version.ChannelID > 0 {
		if summary, ok := versionSummaries[modelSquareVersionMetricKey{ChannelID: version.ChannelID, ModelName: version.ModelName}]; ok {
			return summary, true
		}
		if version.UpstreamKey != "" && version.UpstreamKey != version.ModelName {
			if summary, ok := versionSummaries[modelSquareVersionMetricKey{ChannelID: version.ChannelID, ModelName: version.UpstreamKey}]; ok {
				return summary, true
			}
		}
	}
	if summary, ok := summaries[version.ModelName]; ok {
		return summary, true
	}
	if version.UpstreamKey != "" {
		if summary, ok := summaries[version.UpstreamKey]; ok {
			return summary, true
		}
	}
	return modelSquareMetricSummary{}, false
}

func normalizeModelSquareMapping(mapping map[string]string) map[string]string {
	normalized := make(map[string]string, len(mapping))
	for source, target := range mapping {
		source = strings.TrimSpace(source)
		target = strings.TrimSpace(target)
		if source == "" || target == "" {
			continue
		}
		normalized[source] = target
	}
	return normalized
}

func splitChannelModels(models string) []string {
	values := strings.Split(models, ",")
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func resolveModelSquareMappingTarget(source string, mapping map[string]string) string {
	current := source
	visited := map[string]struct{}{current: {}}
	for {
		next := strings.TrimSpace(mapping[current])
		if next == "" {
			return inferModelSquareParentName(current)
		}
		if _, ok := visited[next]; ok {
			if next == current {
				return inferModelSquareParentName(current)
			}
			return ""
		}
		visited[next] = struct{}{}
		current = next
	}
}

var modelSquareChannelSuffixPattern = regexp.MustCompile(`(?i)^(.+)-[abcd]\d+$`)

func inferModelSquareParentName(modelName string) string {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return ""
	}
	match := modelSquareChannelSuffixPattern.FindStringSubmatch(modelName)
	if len(match) != 2 {
		return modelName
	}
	parent := strings.TrimSpace(match[1])
	if parent == "" {
		return modelName
	}
	return parent
}

func upsertModelVersion(versions []dto.ModelVersion, next dto.ModelVersion) []dto.ModelVersion {
	applyPriceInfoToModelVersion(&next)
	for i, version := range versions {
		if version.ModelName == next.ModelName && version.ChannelID == next.ChannelID {
			versions[i] = next
			return versions
		}
	}
	return append(versions, next)
}

func applyPriceInfoToModelVersion(version *dto.ModelVersion) {
	if version == nil || version.PriceInfo == nil {
		return
	}
	version.DiscountRatio = version.PriceInfo.DiscountRatio
	version.RatioSource = version.PriceInfo.RatioSource
	version.DiscountPercent = discountPercentFromPriceInfo(version.PriceInfo)
}

func buildModelSquarePricingData(pricing []model.Pricing) map[string]*dto.PriceInfo {
	priceByModel := make(map[string]*dto.PriceInfo, len(pricing))
	for _, item := range pricing {
		priceByModel[item.ModelName] = priceInfoFromPricing(item)
	}
	return priceByModel
}

func priceInfoForModel(modelName string, pricing model.Pricing, effectivePriceByModel map[string]*dto.PriceInfo) *dto.PriceInfo {
	if priceInfo, ok := effectivePriceByModel[modelName]; ok && priceInfo != nil {
		copyValue := *priceInfo
		return &copyValue
	}
	return priceInfoFromPricing(pricing)
}

func priceInfoForVersion(modelName string, upstreamKey string, pricing model.Pricing, effectivePriceByModel map[string]*dto.PriceInfo) *dto.PriceInfo {
	if priceInfo, ok := effectivePriceByModel[modelName]; ok && priceInfo != nil {
		copyValue := *priceInfo
		return &copyValue
	}
	return priceInfoFromPricing(pricing)
}

func priceInfoFromPricing(pricing model.Pricing) *dto.PriceInfo {
	discountRatio := normalizeDiscountRatio(pricing.DiscountRatio)
	modelPrice, usePrice := ratio_setting.GetModelPrice(pricing.ModelName, false)
	quotaType := pricing.QuotaType
	if usePrice {
		quotaType = 1
	}
	priceInfo := &dto.PriceInfo{
		InputRatio:    pricing.ModelRatio,
		OutputRatio:   pricing.CompletionRatio,
		QuotaType:     quotaType,
		ModelPrice:    pricing.ModelPrice,
		DiscountRatio: discountRatio,
		RatioSource:   pricing.RatioSource,
	}
	if usePrice {
		priceInfo.ModelPrice = modelPrice
		priceInfo.InputOriginalPrice = modelPrice
		priceInfo.InputDiscountedPrice = modelPrice * discountRatio
		return priceInfo
	}
	if quotaType == 1 {
		priceInfo.InputOriginalPrice = pricing.ModelPrice
		priceInfo.InputDiscountedPrice = pricing.ModelPrice * discountRatio
		return priceInfo
	}
	inputOriginal := pricing.ModelRatio * 2
	outputOriginal := inputOriginal * pricing.CompletionRatio
	priceInfo.InputOriginalPrice = inputOriginal
	priceInfo.OutputOriginalPrice = outputOriginal
	priceInfo.InputDiscountedPrice = inputOriginal * discountRatio
	priceInfo.OutputDiscountedPrice = outputOriginal * discountRatio
	return priceInfo
}

func normalizeDiscountRatio(value float64) float64 {
	if value <= 0 {
		return 1
	}
	return value
}

func addEndpoints(endpointSet map[string]struct{}, pricing model.Pricing) {
	for _, endpoint := range pricing.SupportedEndpointTypes {
		endpointSet[fmt.Sprint(endpoint)] = struct{}{}
	}
}

func buildCapabilities(endpointSet map[string]struct{}) *dto.ModelCapabilities {
	endpoints := make([]string, 0, len(endpointSet))
	for endpoint := range endpointSet {
		endpoints = append(endpoints, endpoint)
	}
	sort.Strings(endpoints)
	streaming := true
	vision := false
	for _, endpoint := range endpoints {
		if endpoint == string(constant.EndpointTypeImageGeneration) || endpoint == "image" || endpoint == "2" {
			vision = true
			break
		}
	}
	return &dto.ModelCapabilities{
		SupportedEndpoints: endpoints,
		Streaming:          &streaming,
		Vision:             &vision,
	}
}

func finalizeModelSquareMetadata(draft *modelSquareDraft) {
	endpoints := sortedEndpointTypes(draft.endpointSet)
	draft.InputTypes = modelSquareInputTypes(draft.ModelSquareItem, endpoints)
	draft.OutputTypes = modelSquareOutputTypes(draft.ModelSquareItem, endpoints)
	draft.Parameters = modelSquareParameters(draft.ModelSquareItem, endpoints)
	draft.Protocols = modelSquareProtocols(draft.ModelSquareItem, endpoints)
	draft.Reasoning = modelSquareReasoning(draft.ModelSquareItem)
	draft.UpdatedTime = draft.updatedTime
	if bestVersionPriceInfo := bestVersionPriceInfo(draft.Versions); bestVersionPriceInfo != nil {
		if !hasDisplayPriceInfo(draft.PriceInfo) || betterDiscountPriceInfo(bestVersionPriceInfo, draft.PriceInfo) {
			draft.PriceInfo = bestVersionPriceInfo
		}
	}
	draft.DiscountPercent = maxVersionDiscount(draft.PriceInfo, draft.Versions)
	for index := range draft.Versions {
		applyPriceInfoToModelVersion(&draft.Versions[index])
		if draft.Versions[index].DiscountPercent == 0 {
			draft.Versions[index].DiscountPercent = discountPercentFromPriceInfo(draft.Versions[index].PriceInfo)
			if draft.Versions[index].DiscountPercent == 0 {
				draft.Versions[index].DiscountPercent = discountPercent(draft.PriceInfo, draft.Versions[index].PriceInfo)
			}
		}
		if draft.Versions[index].UpdatedTime == 0 {
			draft.Versions[index].UpdatedTime = draft.updatedTime
		}
	}
}

func sortedEndpointTypes(endpointSet map[string]struct{}) []string {
	endpoints := make([]string, 0, len(endpointSet))
	for endpoint := range endpointSet {
		endpoints = append(endpoints, endpoint)
	}
	sort.Strings(endpoints)
	return endpoints
}

func modelSquareInputTypes(item dto.ModelSquareItem, endpoints []string) []string {
	values := []string{"Text"}
	if endpointHasAny(endpoints, constant.EndpointTypeImageGeneration, "image", "2") || hasAnyTag(item.Tags, "image", "vision", "图片", "视觉") {
		values = append(values, "Image")
	}
	if endpointHasAny(endpoints, constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAIResponseCompact) || hasAnyTag(item.Tags, "file", "文件") {
		values = append(values, "File")
	}
	if hasAnyTag(item.Tags, "audio", "音频") {
		values = append(values, "Audio")
	}
	if endpointHasAny(endpoints, constant.EndpointTypeOpenAIVideo) || hasAnyTag(item.Tags, "video", "视频") {
		values = append(values, "Video")
	}
	return uniqueStrings(values)
}

func modelSquareOutputTypes(item dto.ModelSquareItem, endpoints []string) []string {
	values := []string{"Text"}
	if endpointHasAny(endpoints, constant.EndpointTypeImageGeneration, "image", "2") || hasAnyTag(item.Tags, "image", "vision", "图片", "视觉") {
		values = append(values, "Image")
	}
	if hasAnyTag(item.Tags, "audio", "音频") {
		values = append(values, "Audio")
	}
	if endpointHasAny(endpoints, constant.EndpointTypeOpenAIVideo) || hasAnyTag(item.Tags, "video", "视频") {
		values = append(values, "Video")
	}
	return uniqueStrings(values)
}

func modelSquareParameters(item dto.ModelSquareItem, endpoints []string) []string {
	values := []string{"max_completion_tokens", "temperature", "top_p"}
	if endpointHasAny(endpoints, constant.EndpointTypeOpenAI, constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAIResponseCompact) {
		values = append(values, "presence_penalty", "frequency_penalty")
	}
	if hasAnyTag(item.Tags, "reason", "推理") {
		values = append(values, "reasoning_effort")
	}
	return uniqueStrings(values)
}

func modelSquareProtocols(item dto.ModelSquareItem, endpoints []string) []string {
	values := make([]string, 0, 6)
	if endpointHasAny(endpoints, constant.EndpointTypeOpenAI) || len(endpoints) == 0 {
		values = append(values, "OpenAI Chat Completions")
	}
	if endpointHasAny(endpoints, constant.EndpointTypeOpenAIResponse, constant.EndpointTypeOpenAIResponseCompact) {
		values = append(values, "OpenAI Responses")
	}
	if endpointHasAny(endpoints, constant.EndpointTypeImageGeneration) {
		values = append(values, "OpenAI Images")
	}
	if endpointHasAny(endpoints, constant.EndpointTypeAnthropic) || strings.Contains(strings.ToLower(item.Name+" "+item.VendorName), "claude") {
		values = append(values, "Anthropic Messages")
	}
	if endpointHasAny(endpoints, constant.EndpointTypeGemini) {
		values = append(values, "Google Gemini")
	}
	if endpointHasAny(endpoints, constant.EndpointTypeOpenAIVideo) {
		values = append(values, "Google Video")
	}
	return uniqueStrings(values)
}

func modelSquareReasoning(item dto.ModelSquareItem) string {
	if !hasAnyTag(item.Tags, "reason", "推理", "thinking", "思考") {
		return "No reasoning"
	}
	if item.Capabilities != nil && item.Capabilities.Streaming != nil && *item.Capabilities.Streaming {
		return "Switchable reasoning"
	}
	return "Always-on reasoning"
}

func endpointHasAny(endpoints []string, targets ...interface{}) bool {
	targetSet := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		targetSet[fmt.Sprint(target)] = struct{}{}
	}
	for _, endpoint := range endpoints {
		if _, ok := targetSet[endpoint]; ok {
			return true
		}
	}
	return false
}

func hasAnyTag(tags []string, needles ...string) bool {
	for _, tag := range tags {
		normalized := strings.ToLower(tag)
		for _, needle := range needles {
			if strings.Contains(normalized, strings.ToLower(needle)) {
				return true
			}
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func discountPercent(base *dto.PriceInfo, current *dto.PriceInfo) int {
	if discount := discountPercentFromPriceInfo(current); discount > 0 {
		return discount
	}
	if base == nil || current == nil || base.InputRatio <= 0 || current.InputRatio <= 0 {
		return 0
	}
	discount := int((1 - current.InputRatio/base.InputRatio) * 100)
	if discount < 0 {
		return 0
	}
	if discount > 100 {
		return 100
	}
	return discount
}

func discountPercentFromPriceInfo(priceInfo *dto.PriceInfo) int {
	if priceInfo == nil || priceInfo.DiscountRatio <= 0 || priceInfo.DiscountRatio >= 1 {
		return 0
	}
	discount := int(math.Round((1 - priceInfo.DiscountRatio) * 100))
	if discount < 0 {
		return 0
	}
	if discount > 100 {
		return 100
	}
	return discount
}

func maxVersionDiscount(base *dto.PriceInfo, versions []dto.ModelVersion) int {
	maxDiscount := 0
	for _, version := range versions {
		discount := version.DiscountPercent
		if discount == 0 {
			discount = discountPercent(base, version.PriceInfo)
		}
		if discount > maxDiscount {
			maxDiscount = discount
		}
	}
	return maxDiscount
}

func bestVersionPriceInfo(versions []dto.ModelVersion) *dto.PriceInfo {
	var best *dto.PriceInfo
	for _, version := range versions {
		if !hasDisplayPriceInfo(version.PriceInfo) {
			continue
		}
		if best == nil || betterDiscountPriceInfo(version.PriceInfo, best) {
			copyValue := *version.PriceInfo
			best = &copyValue
		}
	}
	return best
}

func hasDisplayPriceInfo(priceInfo *dto.PriceInfo) bool {
	if priceInfo == nil {
		return false
	}
	return priceInfo.ModelPrice > 0 ||
		priceInfo.InputOriginalPrice > 0 ||
		priceInfo.OutputOriginalPrice > 0 ||
		priceInfo.InputDiscountedPrice > 0 ||
		priceInfo.OutputDiscountedPrice > 0 ||
		priceInfo.InputRatio > 0 ||
		priceInfo.OutputRatio > 0
}

func betterDiscountPriceInfo(candidate *dto.PriceInfo, current *dto.PriceInfo) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	candidateDiscount := normalizeDiscountRatio(candidate.DiscountRatio)
	currentDiscount := normalizeDiscountRatio(current.DiscountRatio)
	if candidateDiscount != currentDiscount {
		return candidateDiscount < currentDiscount
	}
	return priceInfoSortValue(candidate) < priceInfoSortValue(current)
}

func priceInfoSortValue(priceInfo *dto.PriceInfo) float64 {
	if priceInfo == nil {
		return math.MaxFloat64
	}
	if priceInfo.InputDiscountedPrice > 0 {
		return priceInfo.InputDiscountedPrice
	}
	if priceInfo.ModelPrice > 0 {
		return priceInfo.ModelPrice
	}
	if priceInfo.InputRatio > 0 {
		return priceInfo.InputRatio
	}
	return math.MaxFloat64
}

var (
	modelContextTagPattern = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)(K|M)$`)
	modelDatePattern       = regexp.MustCompile(`(?:^|[^0-9])((?:20\d{2})(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01]))(?:[^0-9]|$)`)
	modelVersionPattern    = regexp.MustCompile(`(?:^|[^0-9])(\d+)(?:[-_.](\d+))?(?:[-_.](\d+))?(?:[-_.](\d+))?(?:[^0-9]|$)`)
)

func modelReleaseSortTime(item dto.ModelSquareItem) int64 {
	if timestamp := modelReleaseDateFromName(item.Name); timestamp > 0 {
		return timestamp
	}
	for _, version := range item.Versions {
		if timestamp := modelReleaseDateFromName(version.ModelName); timestamp > 0 {
			return timestamp
		}
	}
	if versionValue := modelSemanticVersionSortValue(item.Name); versionValue > 0 {
		return versionValue
	}
	for _, version := range item.Versions {
		if versionValue := modelSemanticVersionSortValue(version.ModelName); versionValue > 0 {
			return versionValue
		}
	}
	return item.UpdatedTime
}

func compareModelSquareItemsByVendorLatest(a dto.ModelSquareItem, b dto.ModelSquareItem) int {
	if aVendor, bVendor := modelSquareVendorSortKey(a), modelSquareVendorSortKey(b); aVendor != bVendor {
		if aVendor < bVendor {
			return -1
		}
		return 1
	}
	if aRelease, bRelease := modelReleaseSortTime(a), modelReleaseSortTime(b); aRelease != bRelease {
		if aRelease > bRelease {
			return -1
		}
		return 1
	}
	if a.UpdatedTime != b.UpdatedTime {
		if a.UpdatedTime > b.UpdatedTime {
			return -1
		}
		return 1
	}
	if a.VersionCount != b.VersionCount {
		if a.VersionCount > b.VersionCount {
			return -1
		}
		return 1
	}
	if a.Name < b.Name {
		return -1
	}
	if a.Name > b.Name {
		return 1
	}
	return 0
}

func modelSquareVendorSortKey(item dto.ModelSquareItem) string {
	if item.VendorName != "" {
		return strings.ToLower(strings.TrimSpace(item.VendorName))
	}
	if item.VendorID > 0 {
		return fmt.Sprintf("%08d", item.VendorID)
	}
	return strings.ToLower(strings.TrimSpace(item.Name))
}

func modelReleaseDateFromName(name string) int64 {
	match := modelDatePattern.FindStringSubmatch(name)
	if len(match) < 2 {
		return 0
	}
	dateValue := match[1]
	year, _ := strconv.Atoi(dateValue[0:4])
	month, _ := strconv.Atoi(dateValue[4:6])
	day, _ := strconv.Atoi(dateValue[6:8])
	return int64(year*10000 + month*100 + day)
}

func modelSemanticVersionSortValue(name string) int64 {
	matches := modelVersionPattern.FindAllStringSubmatch(strings.ToLower(name), -1)
	var best int64
	for _, match := range matches {
		var value int64
		for index := 1; index <= 4; index++ {
			part := 0
			if index < len(match) && match[index] != "" {
				if index > 1 && len(match[index]) > 2 {
					break
				}
				part, _ = strconv.Atoi(match[index])
			}
			if index == 1 && part >= 2000 {
				value = 0
				break
			}
			value = value*1000 + int64(part)
		}
		if value > best {
			best = value
		}
	}
	if best == 0 {
		return 0
	}
	return 10_000_000_000 + best
}

func modelSquareContextTokens(name string, tags []string) int64 {
	for _, tag := range tags {
		match := modelContextTagPattern.FindStringSubmatch(strings.TrimSpace(tag))
		if len(match) != 3 {
			continue
		}
		value, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			continue
		}
		if strings.EqualFold(match[2], "M") {
			value *= 1_000_000
		} else {
			value *= 1_000
		}
		return int64(math.Round(value))
	}
	normalized := strings.ToLower(name)
	switch {
	case strings.Contains(normalized, "gemini"):
		return 1_000_000
	case strings.Contains(normalized, "claude"):
		return 200_000
	case strings.Contains(normalized, "grok"):
		return 256_000
	case strings.Contains(normalized, "deepseek"):
		return 128_000
	case strings.Contains(normalized, "gpt-5"):
		return 400_000
	case strings.Contains(normalized, "gpt-4"):
		return 128_000
	}
	return 0
}

func modelSquareMaxOutputTokens(name string) int64 {
	normalized := strings.ToLower(name)
	switch {
	case strings.Contains(normalized, "embedding"):
		return 0
	case strings.Contains(normalized, "gpt-5"),
		strings.Contains(normalized, "grok"):
		return 32_000
	case strings.Contains(normalized, "claude"):
		return 16_000
	case strings.Contains(normalized, "gemini"),
		strings.Contains(normalized, "deepseek"):
		return 8_000
	default:
		return 8_000
	}
}

func ptrString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func roundFloat(value float64, precision int) float64 {
	if precision < 0 {
		precision = 0
	}
	factor := math.Pow(10, float64(precision))
	return math.Round(value*factor) / factor
}

func updatedTimeFromMeta(meta model.Model, hasMeta bool) int64 {
	if !hasMeta {
		return 0
	}
	if meta.UpdatedTime > 0 {
		return meta.UpdatedTime
	}
	return meta.CreatedTime
}

func splitModelTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；'
	})
	tags := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		tag := strings.TrimSpace(strings.Trim(part, `"`))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	return tags
}

func displayModelName(name string) string {
	if name == "" {
		return ""
	}
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == '/'
	})
	for i, part := range parts {
		if len(part) <= 3 {
			parts[i] = strings.ToUpper(part)
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func channelStatusLabel(status int) string {
	if status == common.ChannelStatusEnabled {
		return "enabled"
	}
	return "disabled"
}

func modelVersionDescription(modelName string, upstreamKey string) string {
	if modelName == upstreamKey {
		return "Official route"
	}
	return "Mapped upstream: " + upstreamKey
}

func matchesModelSquareQuery(item dto.ModelSquareItem, query ModelSquareQuery) bool {
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	if keyword != "" {
		haystack := strings.ToLower(item.Name + " " + item.DisplayName + " " + item.Description + " " + item.VendorName)
		if !strings.Contains(haystack, keyword) {
			return false
		}
	}

	if vendorID := strings.TrimSpace(query.VendorID); vendorID != "" {
		id, err := strconv.Atoi(vendorID)
		if err == nil && id != 0 && item.VendorID != id {
			return false
		}
	}

	tag := strings.TrimSpace(query.Tag)
	if tag != "" && tag != "all" {
		found := false
		for _, itemTag := range item.Tags {
			if itemTag == tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
