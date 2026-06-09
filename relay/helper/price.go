package helper

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func modelPriceNotConfiguredError(modelName string, userId int) error {
	if model.IsAdmin(userId) {
		return fmt.Errorf(
			"模型 %s 的价格未配置。请前往「系统设置 → 运营设置」开启自用模式，或在「系统设置 → 分组与模型定价设置」中为该模型配置价格；"+
				"Model %s price not configured. Go to System Settings → Operation Settings to enable self-use mode, or configure the model price in System Settings → Group & Model Pricing.",
			modelName, modelName,
		)
	}
	return fmt.Errorf(
		"模型 %s 的价格尚未由管理员配置，暂时无法使用，请联系站点管理员开启该模型；"+
			"Model %s has not been priced by the administrator yet. Please contact the site administrator to enable this model.",
		modelName, modelName,
	)
}

// https://docs.claude.com/en/docs/build-with-claude/prompt-caching#1-hour-cache-duration
const claudeCacheCreation1hMultiplier = 6 / 3.75

// HandleGroupRatio returns the billing ratio info and any error.
// When ModelLimitsEnabled=true and the model is in ModelLimits but not in the binding table,
// it returns a 403 error indicating a data consistency violation.
// The relay layer will catch this error and return an HTTP 403 response.
func HandleGroupRatio(ctx *gin.Context, relayInfo *relaycommon.RelayInfo) (types.GroupRatioInfo, error) {
	groupRatioInfo := types.GroupRatioInfo{
		GroupRatio:        1.0, // default ratio
		GroupSpecialRatio: -1,
	}

	// check auto group
	autoGroup, exists := ctx.Get("auto_group")
	if exists {
		logger.LogInfo(ctx, fmt.Sprintf("final group: %s", autoGroup))
		relayInfo.UsingGroup = autoGroup.(string)
	}

	// check user group special ratio
	userGroupRatio, ok := ratio_setting.GetGroupGroupRatio(relayInfo.UserGroup, relayInfo.UsingGroup)
	if ok {
		groupRatioInfo.GroupSpecialRatio = userGroupRatio
		groupRatioInfo.GroupRatio = userGroupRatio
		groupRatioInfo.HasSpecialRatio = true
		groupRatioInfo.RatioSource = "group_group_ratio"
	} else {
		groupRatioInfo.GroupRatio = ratio_setting.GetGroupRatio(relayInfo.UsingGroup)
		groupRatioInfo.RatioSource = "group_ratio"
	}

	// Priority 1: Token Pricing Binding (highest)
	// Only applies when token.ModelLimitsEnabled=true AND model is in ModelLimits whitelist.
	// Core invariant: when ModelLimitsEnabled=true, ModelLimits whitelist and binding table must stay in sync.
	//   - If model is in ModelLimits but NOT in binding table → 403 (data inconsistency)
	//   - If model is NOT in ModelLimits → skip Token binding, fall through to enterprise/platform
	//   - When ModelLimitsEnabled=false, binding table data is preserved but not used for billing
	// Read from gin context (set by auth middleware).
	modelLimitsEnabled := common.GetContextKeyBool(ctx, constant.ContextKeyTokenModelLimitEnabled)
	modelLimitsAllowed := false
	if modelLimitsEnabled {
		modelLimitsMapRaw, _ := ctx.Get(string(constant.ContextKeyTokenModelLimit))
		modelLimitsMap, _ := modelLimitsMapRaw.(map[string]bool)
		modelLimitsAllowed = modelLimitsMap != nil && modelLimitsMap[relayInfo.OriginModelName]
	}

	if modelLimitsEnabled && modelLimitsAllowed {
		// Model is in both ModelLimits whitelist and binding table → apply Token binding discount
		// OR model is in ModelLimits but NOT in binding table → 403
		modelBinding, err := model.GetTokenPricingModelBindingByTokenAndModel(
			relayInfo.TokenId,
			relayInfo.OriginModelName,
		)
		if err != nil {
			// Database query error (not "not found"). Log and fall back conservatively.
			logger.LogError(ctx, fmt.Sprintf("token binding query error: %v", err))
			var fallbackErr error
			groupRatioInfo, fallbackErr = handleEnterpriseAndGroupRatio(ctx, relayInfo, groupRatioInfo)
			if fallbackErr != nil {
				return groupRatioInfo, fallbackErr
			}
		} else if modelBinding != nil {
			pricingItem, err := model.GetPricingItemBySheetIdAndModelName(
				modelBinding.PricingSheetId,
				relayInfo.OriginModelName,
			)
			if err != nil {
				logger.LogError(ctx, fmt.Sprintf("pricing item query error: %v", err))
				var fallbackErr error
				groupRatioInfo, fallbackErr = handleEnterpriseAndGroupRatio(ctx, relayInfo, groupRatioInfo)
				if fallbackErr != nil {
					return groupRatioInfo, fallbackErr
				}
			} else if pricingItem != nil {
				groupRatioInfo.EnterpriseSheetId = modelBinding.PricingSheetId
				if pricingItem.DiscountType == model.DiscountTypePerCall {
					groupRatioInfo.PerCallPriceSheet = pricingItem.DiscountValue
				} else {
					groupRatioInfo.GroupRatio = pricingItem.DiscountValue
				}
				groupRatioInfo.RatioSource = "token_pricing_binding"
			}
			// Whether or not pricingItem was found, do not fall through to enterprise/platform chain
		} else {
			// Model is in ModelLimits but NOT in binding table → 403 (data inconsistency)
			return groupRatioInfo, types.NewErrorWithStatusCode(
				fmt.Errorf("model '%s' is in token model limits but not found in pricing binding", relayInfo.OriginModelName),
				types.ErrorCodeTokenModelLimitInconsistent,
				http.StatusForbidden,
				types.ErrOptionWithSkipRetry(),
			)
		}
	} else {
		// ModelLimitsEnabled=false OR model not in ModelLimits whitelist → skip Token binding chain
		var fallbackErr error
		groupRatioInfo, fallbackErr = handleEnterpriseAndGroupRatio(ctx, relayInfo, groupRatioInfo)
		if fallbackErr != nil {
			return groupRatioInfo, fallbackErr
		}
	}

	// Priority 5: Supplier pricing sheet cost tracking (does not affect customer billing)
	groupRatioInfo = HandleSupplierPricingSheet(ctx, relayInfo, groupRatioInfo)

	return groupRatioInfo, nil
}

// handleEnterpriseAndGroupRatio handles billing priorities 2-4:
//   - Priority 2: User's enterprise pricing sheet
//   - Priority 3: Platform pricing sheet (fallback when enterprise sheet doesn't have the model)
//   - Priority 4: Group ratio (fallback when neither sheet has the model)
//
// Returns (groupRatioInfo, error). When error is non-nil, the caller should return it immediately.
func handleEnterpriseAndGroupRatio(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, groupRatioInfo types.GroupRatioInfo) (types.GroupRatioInfo, error) {
	// Priority 2: User's enterprise pricing sheet
	sheet, err := getUserActivePricingSheetForBilling(relayInfo.UserId)
	if err != nil {
		return groupRatioInfo, types.NewErrorWithStatusCode(
			err, types.ErrorCodeModelPriceError, http.StatusInternalServerError,
			types.ErrOptionWithSkipRetry(),
		)
	}
	if sheet != nil {
		pricingItemResult := getPricingItemResult(sheet.Id, relayInfo.OriginModelName)
		if pricingItemResult.Found && pricingItemResult.Item != nil {
			applyPricingItem(pricingItemResult.Item, sheet.Id, sheet.Name, &groupRatioInfo, "enterprise_pricing_sheet")
			return groupRatioInfo, nil
		}
	}

	// Priority 3: Platform pricing sheet (fallback when enterprise sheet doesn't have this model)
	// Aggregate all active platform sheets and use the lowest discount_value for this model.
	platformSheets, err := model.GetAllActivePricingSheetsByEnterpriseIdByType(model.EnterpriseTypePlatform)
	if err != nil {
		return groupRatioInfo, types.NewErrorWithStatusCode(
			err, types.ErrorCodeModelPriceError, http.StatusInternalServerError,
			types.ErrOptionWithSkipRetry(),
		)
	}
	bestItem := (*model.EnterprisePricingItem)(nil)
	bestSheet := (*model.EnterprisePricingSheet)(nil)
	for _, platformSheet := range platformSheets {
		pricingItemResult := getPricingItemResult(platformSheet.Id, relayInfo.OriginModelName)
		if pricingItemResult.Found && pricingItemResult.Item != nil {
			if bestItem == nil || pricingItemResult.Item.DiscountValue < bestItem.DiscountValue {
				bestItem = pricingItemResult.Item
				bestSheet = platformSheet
			}
		}
	}
	if bestItem != nil && bestSheet != nil {
		applyPricingItem(bestItem, bestSheet.Id, bestSheet.Name, &groupRatioInfo, "platform_pricing_sheet")
		return groupRatioInfo, nil
	}

	// Priority 4: Fall back to group ratio (set by initial HandleGroupRatio, no change needed)
	groupRatioInfo.RatioSource = "group_ratio"
	return groupRatioInfo, nil
}

// applyPricingItem applies a pricing item's discount to the groupRatioInfo.
func applyPricingItem(item *model.EnterprisePricingItem, sheetId int, sheetName string, info *types.GroupRatioInfo, source string) {
	info.EnterpriseSheetId = sheetId
	info.EnterpriseSheetName = sheetName
	if item.DiscountType == model.DiscountTypePerCall {
		info.PerCallPriceSheet = item.DiscountValue
		info.RatioSource = source
	} else {
		info.GroupRatio = item.DiscountValue
		info.RatioSource = source
	}
}

// HandleEnterprisePricingSheet is kept for backward compatibility.
// New billing code should use handleEnterpriseAndGroupRatio directly.
// It delegates to handleEnterpriseAndGroupRatio but discards the error
// (for cases where the caller cannot handle errors).
func HandleEnterprisePricingSheet(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, baseRatioInfo types.GroupRatioInfo) types.GroupRatioInfo {
	info, _ := handleEnterpriseAndGroupRatio(ctx, relayInfo, baseRatioInfo)
	return info
}

// getUserActivePricingSheetForBilling is the internal helper for billing.
// It wraps the service layer to avoid circular imports.
func getUserActivePricingSheetForBilling(userId int) (*model.EnterprisePricingSheet, error) {
	enterpriseId, found := model.IsUserInEnterprise(userId)
	if !found {
		return nil, nil
	}
	if !model.IsEnterpriseEnabled(enterpriseId) {
		return nil, nil
	}
	return model.GetFirstActivePricingSheetByEnterpriseId(enterpriseId)
}

// PricingItemResult holds the resolved discount from a pricing sheet.
type PricingItemResult struct {
	Item  *model.EnterprisePricingItem
	Found bool
}

// getPricingItemResult returns the pricing item for a specific sheet and model.
// Returns only an exact model match; no vendor-type fallback (group_ratio handles fallback).
func getPricingItemResult(sheetId int, modelName string) PricingItemResult {
	item, err := model.GetPricingItemBySheetIdAndModelName(sheetId, modelName)
	if err == nil && item != nil {
		return PricingItemResult{Item: item, Found: true}
	}
	return PricingItemResult{Found: false}
}

func ModelPriceHelper(c *gin.Context, info *relaycommon.RelayInfo, promptTokens int, meta *types.TokenCountMeta) (types.PriceData, error) {
	modelPrice, usePrice := ratio_setting.GetModelPrice(info.OriginModelName, false)

	groupRatioInfo, billingErr := HandleGroupRatio(c, info)
	if billingErr != nil {
		return types.PriceData{}, billingErr
	}

	// per_call: enterprise pricing sheet sets a fixed per-call price for this model.
	// The billing behaves like per-call (MJ/Task): charge a fixed amount per request.
	if groupRatioInfo.PerCallPriceSheet > 0 {
		quota := int(groupRatioInfo.PerCallPriceSheet * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
		freeModel := false
		if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
			if groupRatioInfo.GroupRatio == 0 {
				quota = 0
				freeModel = true
			}
		}
		priceData := types.PriceData{
			FreeModel:         freeModel,
			ModelPrice:        groupRatioInfo.PerCallPriceSheet,
			GroupRatioInfo:    groupRatioInfo,
			UsePrice:         true,
			Quota:            quota,
			PerCallPriceSheet: groupRatioInfo.PerCallPriceSheet,
		}
		if common.DebugEnabled {
			println(fmt.Sprintf("model_price_helper (per_call sheet): model=%s price=%.4f quota=%d", info.OriginModelName, groupRatioInfo.PerCallPriceSheet, quota))
		}
		info.PriceData = priceData
		return priceData, nil
	}

	// Check if this model uses tiered_expr billing
	if billing_setting.GetBillingMode(info.OriginModelName) == billing_setting.BillingModeTieredExpr {
		return modelPriceHelperTiered(c, info, promptTokens, meta, groupRatioInfo)
	}

	var preConsumedQuota int
	var modelRatio float64
	var completionRatio float64
	var cacheRatio float64
	var imageRatio float64
	var cacheCreationRatio float64
	var cacheCreationRatio5m float64
	var cacheCreationRatio1h float64
	var audioRatio float64
	var audioCompletionRatio float64
	var freeModel bool
	if !usePrice {
		preConsumedTokens := common.Max(promptTokens, common.PreConsumedQuota)
		if meta != nil && meta.MaxTokens != 0 {
			preConsumedTokens += meta.MaxTokens
		}
		var success bool
		var matchName string
		modelRatio, success, matchName = ratio_setting.GetModelRatio(info.OriginModelName)
		if !success {
			acceptUnsetRatio := false
			if info.UserSetting.AcceptUnsetRatioModel {
				acceptUnsetRatio = true
			}
			// If enterprise pricing sheet overrides the ratio, accept the model even without group ratio config
			if groupRatioInfo.RatioSource == "enterprise_pricing_sheet" {
				acceptUnsetRatio = true
			}
			if !acceptUnsetRatio {
				return types.PriceData{}, modelPriceNotConfiguredError(matchName, info.UserId)
			}
		}
		completionRatio = ratio_setting.GetCompletionRatio(info.OriginModelName)
		cacheRatio, _ = ratio_setting.GetCacheRatio(info.OriginModelName)
		cacheCreationRatio, _ = ratio_setting.GetCreateCacheRatio(info.OriginModelName)
		cacheCreationRatio5m = cacheCreationRatio
		// 固定1h和5min缓存写入价格的比例
		cacheCreationRatio1h = cacheCreationRatio * claudeCacheCreation1hMultiplier
		imageRatio, _ = ratio_setting.GetImageRatio(info.OriginModelName)
		audioRatio = ratio_setting.GetAudioRatio(info.OriginModelName)
		audioCompletionRatio = ratio_setting.GetAudioCompletionRatio(info.OriginModelName)
		ratio := modelRatio * groupRatioInfo.GroupRatio
		preConsumedQuota = int(float64(preConsumedTokens) * ratio)
	} else {
		if meta.ImagePriceRatio != 0 {
			modelPrice = modelPrice * meta.ImagePriceRatio
		}
		preConsumedQuota = int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
	}

	// check if free model pre-consume is disabled
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		// if model price or ratio is 0, do not pre-consume quota
		if groupRatioInfo.GroupRatio == 0 {
			preConsumedQuota = 0
			freeModel = true
		} else if usePrice {
			if modelPrice == 0 {
				preConsumedQuota = 0
				freeModel = true
			}
		} else {
			if modelRatio == 0 {
				preConsumedQuota = 0
				freeModel = true
			}
		}
	}

	priceData := types.PriceData{
		FreeModel:            freeModel,
		ModelPrice:           modelPrice,
		ModelRatio:           modelRatio,
		CompletionRatio:      completionRatio,
		GroupRatioInfo:       groupRatioInfo,
		UsePrice:             usePrice,
		CacheRatio:           cacheRatio,
		ImageRatio:           imageRatio,
		AudioRatio:           audioRatio,
		AudioCompletionRatio: audioCompletionRatio,
		CacheCreationRatio:   cacheCreationRatio,
		CacheCreation5mRatio: cacheCreationRatio5m,
		CacheCreation1hRatio: cacheCreationRatio1h,
		QuotaToPreConsume:    preConsumedQuota,
	}

	if common.DebugEnabled {
		println(fmt.Sprintf("model_price_helper result: %s", priceData.ToSetting()))
	}
	info.PriceData = priceData
	return priceData, nil
}

// ModelPriceHelperPerCall 按次/按量计费的 PriceHelper (MJ、Task)
// 支持企业报价单的 per_call 类型：报价单中配置了 per_call 时，discount_value 作为绝对价格使用。
func ModelPriceHelperPerCall(c *gin.Context, info *relaycommon.RelayInfo) (types.PriceData, error) {
	groupRatioInfo, billingErr := HandleGroupRatio(c, info)
	if billingErr != nil {
		return types.PriceData{}, billingErr
	}

	// per_call 优先级最高：使用企业报价单的绝对价格作为每次调用费用。
	if groupRatioInfo.PerCallPriceSheet > 0 {
		quota := int(groupRatioInfo.PerCallPriceSheet * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
		freeModel := false
		if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
			if groupRatioInfo.GroupRatio == 0 {
				quota = 0
				freeModel = true
			}
		}
		priceData := types.PriceData{
			FreeModel:         freeModel,
			ModelPrice:        groupRatioInfo.PerCallPriceSheet,
			GroupRatioInfo:    groupRatioInfo,
			UsePrice:         true,
			Quota:            quota,
			PerCallPriceSheet: groupRatioInfo.PerCallPriceSheet,
		}
		if common.DebugEnabled {
			println(fmt.Sprintf("model_price_helper_percall (per_call sheet): model=%s price=%.4f quota=%d", info.OriginModelName, groupRatioInfo.PerCallPriceSheet, quota))
		}
		info.PriceData = priceData
		return priceData, nil
	}

	modelPrice, success := ratio_setting.GetModelPrice(info.OriginModelName, true)
	usePrice := success
	var modelRatio float64

	if !success {
		defaultPrice, ok := ratio_setting.GetDefaultModelPriceMap()[info.OriginModelName]
		if ok {
			modelPrice = defaultPrice
			usePrice = true
		} else {
			var ratioSuccess bool
			var matchName string
			modelRatio, ratioSuccess, matchName = ratio_setting.GetModelRatio(info.OriginModelName)
			acceptUnsetRatio := false
			if info.UserSetting.AcceptUnsetRatioModel {
				acceptUnsetRatio = true
			}
			if !ratioSuccess && !acceptUnsetRatio {
				return types.PriceData{}, modelPriceNotConfiguredError(matchName, info.UserId)
			}
		}
	}

	var quota int
	freeModel := false

	if usePrice {
		quota = int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
		if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
			if groupRatioInfo.GroupRatio == 0 || modelPrice == 0 {
				quota = 0
				freeModel = true
			}
		}
	} else {
		// 按量计费：以模型倍率的一半作为预扣额度
		quota = int(modelRatio / 2 * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
		modelPrice = -1
		if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
			if groupRatioInfo.GroupRatio == 0 || modelRatio == 0 {
				quota = 0
				freeModel = true
			}
		}
	}

	priceData := types.PriceData{
		FreeModel:      freeModel,
		ModelPrice:     modelPrice,
		ModelRatio:     modelRatio,
		UsePrice:       usePrice,
		Quota:          quota,
		GroupRatioInfo: groupRatioInfo,
	}
	return priceData, nil
}

func HasModelBillingConfig(modelName string) bool {
	if _, ok := ratio_setting.GetModelPrice(modelName, false); ok {
		return true
	}
	if _, ok, _ := ratio_setting.GetModelRatio(modelName); ok {
		return true
	}
	if billing_setting.GetBillingMode(modelName) != billing_setting.BillingModeTieredExpr {
		return false
	}
	expr, ok := billing_setting.GetBillingExpr(modelName)
	return ok && strings.TrimSpace(expr) != ""
}

func modelPriceHelperTiered(c *gin.Context, info *relaycommon.RelayInfo, promptTokens int, meta *types.TokenCountMeta, groupRatioInfo types.GroupRatioInfo) (types.PriceData, error) {
	exprStr, ok := billing_setting.GetBillingExpr(info.OriginModelName)
	if !ok {
		return types.PriceData{}, fmt.Errorf("model %s is configured as tiered_expr but has no billing expression", info.OriginModelName)
	}

	estimatedCompletionTokens := 0
	if meta.MaxTokens != 0 {
		estimatedCompletionTokens = meta.MaxTokens
	}

	requestInput, err := ResolveIncomingBillingExprRequestInput(c, info)
	if err != nil {
		return types.PriceData{}, err
	}

	rawCost, trace, err := billingexpr.RunExprWithRequest(exprStr, billingexpr.TokenParams{
		P:   float64(promptTokens),
		C:   float64(estimatedCompletionTokens),
		Len: float64(promptTokens),
	}, requestInput)
	if err != nil {
		return types.PriceData{}, fmt.Errorf("model %s tiered expr run failed: %w", info.OriginModelName, err)
	}

	// Expression coefficients are $/1M tokens prices; convert to quota the same way per-call billing does.
	quotaBeforeGroup := rawCost / 1_000_000 * common.QuotaPerUnit
	preConsumedQuota := billingexpr.QuotaRound(quotaBeforeGroup * groupRatioInfo.GroupRatio)

	freeModel := false
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		if groupRatioInfo.GroupRatio == 0 {
			preConsumedQuota = 0
			freeModel = true
		}
	}

	exprHash := billingexpr.ExprHashString(exprStr)
	snapshot := &billingexpr.BillingSnapshot{
		BillingMode:               billing_setting.BillingModeTieredExpr,
		ModelName:                 info.OriginModelName,
		ExprString:                exprStr,
		ExprHash:                  exprHash,
		GroupRatio:                groupRatioInfo.GroupRatio,
		EstimatedPromptTokens:     promptTokens,
		EstimatedCompletionTokens: estimatedCompletionTokens,
		EstimatedQuotaBeforeGroup: quotaBeforeGroup,
		EstimatedQuotaAfterGroup:  preConsumedQuota,
		EstimatedTier:             trace.MatchedTier,
		QuotaPerUnit:              common.QuotaPerUnit,
		ExprVersion:               billingexpr.ExprVersion(exprStr),
	}
	info.TieredBillingSnapshot = snapshot
	info.BillingRequestInput = &requestInput

	priceData := types.PriceData{
		FreeModel:         freeModel,
		GroupRatioInfo:    groupRatioInfo,
		QuotaToPreConsume: preConsumedQuota,
	}

	if common.DebugEnabled {
		println(fmt.Sprintf("model_price_helper_tiered result: model=%s preConsume=%d quotaBeforeGroup=%.2f groupRatio=%.2f tier=%s", info.OriginModelName, preConsumedQuota, quotaBeforeGroup, groupRatioInfo.GroupRatio, trace.MatchedTier))
	}

	info.PriceData = priceData
	return priceData, nil
}

// HandleSupplierPricingSheet looks up the supplier pricing sheet for the current channel
// and records the supplier cost in the GroupRatioInfo.
// This does NOT affect the customer's billing ratio - it is purely for cost tracking.
func HandleSupplierPricingSheet(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, baseInfo types.GroupRatioInfo) types.GroupRatioInfo {
	channelId := 0
	if relayInfo.ChannelMeta != nil {
		channelId = relayInfo.ChannelMeta.ChannelId
	}
	// Fallback: when ChannelMeta is nil (before InitChannelMeta is called),
	// read channel_id directly from the gin context.
	if channelId <= 0 {
		channelId = common.GetContextKeyInt(ctx, constant.ContextKeyChannelId)
	}

	logger.LogInfo(ctx, fmt.Sprintf("[DEBUG_BILLING] HandleSupplierPricingSheet: channelId=%d model=%s userId=%d", channelId, relayInfo.OriginModelName, relayInfo.UserId))
	if channelId <= 0 {
		logger.LogInfo(ctx, fmt.Sprintf("[DEBUG_BILLING] HandleSupplierPricingSheet: channelId=%d <= 0, skipping supplier pricing", channelId))
		return baseInfo
	}

	sheetId, found := getSupplierActivePricingSheetForBilling(channelId)
	if !found {
		logger.LogInfo(ctx, fmt.Sprintf("[DEBUG_BILLING] HandleSupplierPricingSheet: no active supplier sheet found for channelId=%d", channelId))
		return baseInfo
	}
	logger.LogInfo(ctx, fmt.Sprintf("[DEBUG_BILLING] HandleSupplierPricingSheet: found supplier sheet id=%d for channelId=%d", sheetId, channelId))

	cost, costFound := getSupplierModelCostForBilling(sheetId, relayInfo.OriginModelName)
	if !costFound {
		logger.LogInfo(ctx, fmt.Sprintf("[DEBUG_BILLING] HandleSupplierPricingSheet: model=%s NOT found in supplier sheet id=%d", relayInfo.OriginModelName, sheetId))
		return baseInfo
	}
	logger.LogInfo(ctx, fmt.Sprintf("[DEBUG_BILLING] HandleSupplierPricingSheet: model=%s matched in supplier sheet id=%d, cost=%.6f", relayInfo.OriginModelName, sheetId, cost))

	discountType := getSupplierModelCostTypeForBilling(sheetId, relayInfo.OriginModelName)
	sheet := getSupplierPricingSheetById(sheetId)

	baseInfo.SupplierCost = cost
	baseInfo.SupplierCostType = discountType
	if sheet != nil {
		baseInfo.SupplierSheetId = sheet.Id
		baseInfo.SupplierSheetName = sheet.Name
	}

	logger.LogInfo(ctx, fmt.Sprintf("[BILLING] HandleSupplierPricingSheet: supplier pricing sheet applied: channelId=%d sheetId=%d sheet=%s cost=%.6f type=%s", channelId, sheetId, baseInfo.SupplierSheetName, cost, discountType))
	return baseInfo
}

// getSupplierActivePricingSheetForBilling is the internal helper for billing.
func getSupplierActivePricingSheetForBilling(channelId int) (int, bool) {
	sheet, err := model.GetFirstActivePricingSheetByChannelIdOnly(channelId)
	if err != nil || sheet == nil {
		return 0, false
	}
	return sheet.Id, true
}

// getSupplierModelCostForBilling returns the supplier cost for a given pricing sheet and model.
func getSupplierModelCostForBilling(sheetId int, modelName string) (float64, bool) {
	item, err := model.GetSupplierPricingItemBySheetIdAndModel(sheetId, modelName)
	if err != nil || item == nil {
		return 0, false
	}
	return item.DiscountValue, true
}

// getSupplierModelCostTypeForBilling returns the supplier cost type for a given pricing sheet and model.
func getSupplierModelCostTypeForBilling(sheetId int, modelName string) string {
	item, err := model.GetSupplierPricingItemBySheetIdAndModel(sheetId, modelName)
	if err != nil || item == nil {
		return ""
	}
	return item.DiscountType
}

// getSupplierPricingSheetById returns the pricing sheet name for a given sheet id.
func getSupplierPricingSheetById(sheetId int) *model.SupplierPricingSheet {
	sheet, err := model.GetSupplierPricingSheetById(sheetId)
	if err != nil || sheet == nil {
		return nil
	}
	return sheet
}
