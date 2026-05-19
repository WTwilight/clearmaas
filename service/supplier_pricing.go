package service

import (
	"github.com/QuantumNous/new-api/model"
)

// IsSupplierEnabled checks if a supplier is enabled.
func IsSupplierEnabled(supplierId int) bool {
	return model.IsSupplierEnabled(supplierId)
}

// GetChannelActivePricingSheet returns the currently active pricing sheet for a channel.
// Priority: ChannelId exact match > SupplierId universal (channel_id = 0).
// Returns nil if no active sheet is found.
func GetChannelActivePricingSheet(channelId int, supplierId int) (*model.SupplierPricingSheet, error) {
	return model.GetFirstActivePricingSheetByChannelId(supplierId, channelId)
}

// GetChannelActivePricingSheetByChannelId returns the currently active pricing sheet for a channel.
// Does not require supplierId; finds any supplier's sheet matching the channel.
// Priority: ChannelId exact match > universal (channel_id = 0).
// Returns nil if channelId <= 0 or no active sheet is found.
func GetChannelActivePricingSheetByChannelId(channelId int) (*model.SupplierPricingSheet, error) {
	if channelId <= 0 {
		return nil, nil
	}
	return model.GetFirstActivePricingSheetByChannelIdOnly(channelId)
}

// GetSupplierPricingSheetItems returns all pricing items for a given pricing sheet.
func GetSupplierPricingSheetItems(sheetId int) ([]*model.SupplierPricingItem, error) {
	return model.GetSupplierPricingItemsBySheetId(sheetId)
}

// GetModelCost returns the discount value for a given pricing sheet and model.
// Returns (discountValue, found). The discount value represents the cost based on discount type.
func GetModelCost(sheetId int, modelName string) (float64, bool) {
	item, err := model.GetSupplierPricingItemBySheetIdAndModel(sheetId, modelName)
	if err != nil || item == nil {
		return 0, false
	}
	return item.DiscountValue, true
}

// GetModelCostDiscountType returns the discount type for a given pricing sheet and model.
func GetModelCostDiscountType(sheetId int, modelName string) (string, bool) {
	item, err := model.GetSupplierPricingItemBySheetIdAndModel(sheetId, modelName)
	if err != nil || item == nil {
		return "", false
	}
	return item.DiscountType, true
}

// GetChannelActivePricingSheetForBilling returns the active pricing sheet ID for a channel.
// This is the billing chain variant that only returns the sheet ID.
// Returns (sheetId, found).
func GetChannelActivePricingSheetForBilling(channelId int) (int, bool) {
	sheet, err := model.GetFirstActivePricingSheetByChannelIdOnly(channelId)
	if err != nil || sheet == nil {
		return 0, false
	}
	return sheet.Id, true
}

// GetModelCostForBilling returns the discount value for a given pricing sheet and model.
// This is the billing chain variant with optimized return signature.
func GetModelCostForBilling(sheetId int, modelName string) (float64, bool) {
	return GetModelCost(sheetId, modelName)
}
