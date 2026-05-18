package service

import (
	"github.com/QuantumNous/new-api/model"
)

// IsUserInEnterprise checks if a user is bound to any enterprise.
// This wraps the model function for service layer access.
func IsUserInEnterprise(userId int) (int, bool) {
	return model.IsUserInEnterprise(userId)
}

// GetUserActivePricingSheet returns the currently active pricing sheet for a user.
// Returns nil if the user is not bound to any enabled enterprise, or if no active sheet exists.
func GetUserActivePricingSheet(userId int) (*model.EnterprisePricingSheet, error) {
	enterpriseId, found := IsUserInEnterprise(userId)
	if !found {
		return nil, nil
	}

	// Check if enterprise is enabled
	if !model.IsEnterpriseEnabled(enterpriseId) {
		return nil, nil
	}

	// Get active pricing sheets
	sheet, err := model.GetFirstActivePricingSheetByEnterpriseId(enterpriseId)
	if err != nil {
		return nil, err
	}
	return sheet, nil
}

// GetPricingSheetItems returns all pricing items for a given pricing sheet.
func GetPricingSheetItems(sheetId int) ([]*model.EnterprisePricingItem, error) {
	return model.GetPricingItemsBySheetId(sheetId)
}

// GetModelDiscount returns the discount value (ratio) for a given pricing sheet and model.
// Returns (ratio, found). The ratio is the discount value from the pricing item.
func GetModelDiscount(sheetId int, modelName string) (float64, bool) {
	item, err := model.GetPricingItemBySheetIdAndModel(sheetId, modelName)
	if err != nil || item == nil {
		return 0, false
	}
	return item.DiscountValue, true
}

// GetPricingItemDiscountType returns the discount type for a given pricing sheet and model.
func GetPricingItemDiscountType(sheetId int, modelName string) (string, bool) {
	item, err := model.GetPricingItemBySheetIdAndModel(sheetId, modelName)
	if err != nil || item == nil {
		return "", false
	}
	return item.DiscountType, true
}
