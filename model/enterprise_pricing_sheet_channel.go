package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// EnterprisePricingSheetChannel represents a channel bound to an enterprise pricing sheet.
type EnterprisePricingSheetChannel struct {
	Id             int   `json:"id" gorm:"primaryKey;autoIncrement"`
	PricingSheetId int   `json:"pricing_sheet_id" gorm:"column:pricing_sheet_id"`
	ChannelId      int   `json:"channel_id" gorm:"column:channel_id"`
	CreatedAt      int64 `json:"created_at"`
}

func (e *EnterprisePricingSheetChannel) TableName() string {
	return "enterprise_pricing_sheet_channels"
}

// Create inserts a new sheet-channel binding.
func (e *EnterprisePricingSheetChannel) Create() error {
	if e.CreatedAt == 0 {
		e.CreatedAt = time.Now().Unix()
	}
	return DB.Create(e).Error
}

// Delete deletes a sheet-channel binding by pricing_sheet_id and channel_id.
func (e *EnterprisePricingSheetChannel) Delete() error {
	return DB.Where("pricing_sheet_id = ? AND channel_id = ?", e.PricingSheetId, e.ChannelId).
		Delete(&EnterprisePricingSheetChannel{}).Error
}

// DeleteBySheetId deletes all bindings for a given pricing sheet.
func DeletePricingSheetChannelBySheetId(sheetId int) error {
	return DB.Where("pricing_sheet_id = ?", sheetId).Delete(&EnterprisePricingSheetChannel{}).Error
}

// GetChannelIdsBySheetId returns all channel IDs bound to a pricing sheet.
func GetChannelIdsBySheetId(sheetId int) ([]int, error) {
	var bindings []EnterprisePricingSheetChannel
	err := DB.Where("pricing_sheet_id = ?", sheetId).Find(&bindings).Error
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(bindings))
	for _, b := range bindings {
		ids = append(ids, b.ChannelId)
	}
	return ids, nil
}

// GetChannelsBySheetId returns all channel objects bound to a pricing sheet.
func GetChannelsBySheetId(sheetId int) ([]*Channel, error) {
	ids, err := GetChannelIdsBySheetId(sheetId)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var channels []*Channel
	err = DB.Where("id IN ?", ids).Find(&channels).Error
	return channels, err
}

// GetFirstMatchedChannelForModel returns the first channel bound to the sheet
// that supports the given model in the given group.
// Returns (channel, true) if found; (nil, false) otherwise.
func GetFirstMatchedChannelForModel(sheetId int, modelName string, group string) (*Channel, bool) {
	ids, err := GetChannelIdsBySheetId(sheetId)
	if err != nil || len(ids) == 0 {
		return nil, false
	}
	for _, channelId := range ids {
		if IsChannelEnabledForGroupModel(group, modelName, channelId) {
			ch, err := CacheGetChannel(channelId)
			if err == nil && ch != nil && ch.Status == common.ChannelStatusEnabled {
				return ch, true
			}
		}
	}
	return nil, false
}

// BindChannelsToSheet replaces all channel bindings for a given pricing sheet.
func BindChannelsToSheet(sheetId int, channelIds []int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("pricing_sheet_id = ?", sheetId).Delete(&EnterprisePricingSheetChannel{}).Error; err != nil {
			return err
		}
		now := time.Now().Unix()
		for _, chId := range channelIds {
			binding := &EnterprisePricingSheetChannel{
				PricingSheetId: sheetId,
				ChannelId:      chId,
				CreatedAt:      now,
			}
			if err := tx.Create(binding).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
