package service

import (
	"github.com/QuantumNous/new-api/model"
)

// GetEnterpriseChannelForModel checks if the user is bound to an enterprise with an active
// pricing sheet whose associated channels contain one that supports the given model.
// Returns (channel, true) if found; (nil, false) otherwise (falls through to default routing).
func GetEnterpriseChannelForModel(userId int, modelName string, usingGroup string) (*model.Channel, bool) {
	enterpriseId, found := model.IsUserInEnterprise(userId)
	if !found {
		return nil, false
	}
	if !model.IsEnterpriseEnabled(enterpriseId) {
		return nil, false
	}
	sheet, err := model.GetFirstActivePricingSheetByEnterpriseId(enterpriseId)
	if err != nil || sheet == nil {
		return nil, false
	}
	channel, found := model.GetFirstMatchedChannelForModel(sheet.Id, modelName, usingGroup)
	return channel, found
}
