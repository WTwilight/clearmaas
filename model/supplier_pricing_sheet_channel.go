package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// SupplierPricingSheetChannel represents a channel bound to a supplier pricing sheet.
type SupplierPricingSheetChannel struct {
	Id            int   `json:"id" gorm:"primaryKey;autoIncrement"`
	PricingSheetId int  `json:"pricing_sheet_id" gorm:"column:pricing_sheet_id"`
	ChannelId     int   `json:"channel_id" gorm:"column:channel_id"`
	CreatedAt     int64 `json:"created_at"`
}

func (SupplierPricingSheetChannel) TableName() string {
	return "supplier_pricing_sheet_channels"
}

func (s *SupplierPricingSheetChannel) Create() error {
	if s.CreatedAt == 0 {
		s.CreatedAt = time.Now().Unix()
	}
	return DB.Create(s).Error
}

func (s *SupplierPricingSheetChannel) Delete() error {
	return DB.Where("pricing_sheet_id = ? AND channel_id = ?", s.PricingSheetId, s.ChannelId).
		Delete(&SupplierPricingSheetChannel{}).Error
}

func DeleteSupplierPricingSheetChannelsBySheetId(sheetId int) error {
	return DB.Where("pricing_sheet_id = ?", sheetId).Delete(&SupplierPricingSheetChannel{}).Error
}

func GetSupplierPricingSheetChannelIds(sheetId int) ([]int, error) {
	var bindings []SupplierPricingSheetChannel
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

func GetSupplierPricingSheetChannels(sheetId int) ([]*Channel, error) {
	ids, err := GetSupplierPricingSheetChannelIds(sheetId)
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

func BindChannelsToSupplierPricingSheet(sheetId int, channelIds []int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("pricing_sheet_id = ?", sheetId).Delete(&SupplierPricingSheetChannel{}).Error; err != nil {
			return err
		}
		now := time.Now().Unix()
		for _, chId := range channelIds {
			binding := &SupplierPricingSheetChannel{
				PricingSheetId: sheetId,
				ChannelId:     chId,
				CreatedAt:     now,
			}
			if err := tx.Create(binding).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetActivePricingSheetByChannelIdBinding returns the first active supplier pricing sheet
// that is bound to the given channel via the supplier_pricing_sheet_channels table.
// Priority: exact channel binding > universal (no explicit bindings).
// Requires: supplier enabled, status=active, current time within [start_time, end_time].
func GetActivePricingSheetByChannelIdBinding(channelId int) (*SupplierPricingSheet, error) {
	if channelId <= 0 {
		return nil, nil
	}
	now := time.Now().Unix()
	tf := timeFilterExpr()

	// 1. Try exact channel binding match via the channels table
	var sheet SupplierPricingSheet
	err := DB.
		Joins("JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
		Joins("JOIN supplier_pricing_sheet_channels ch_bind ON ch_bind.pricing_sheet_id = supplier_pricing_sheets.id").
		Where("ch_bind.channel_id = ?", channelId).
		Where("supplier_pricing_sheets.status = ?", SupplierPricingSheetStatusActive).
		Where(tf, now, now).
		Where("suppliers.status = ?", SupplierStatusEnabled).
		Order("supplier_pricing_sheets.id desc").
		First(&sheet).Error
	if err == nil {
		return &sheet, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 2. Fall back to universal sheet (no explicit channel bindings: channel_id = 0)
	err = DB.
		Joins("JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
		Where("supplier_pricing_sheets.channel_id = ?", 0).
		Where("supplier_pricing_sheets.status = ?", SupplierPricingSheetStatusActive).
		Where(tf, now, now).
		Where("suppliers.status = ?", SupplierStatusEnabled).
		Where(`NOT EXISTS (
			SELECT 1 FROM supplier_pricing_sheet_channels ch_bind2
			WHERE ch_bind2.pricing_sheet_id = supplier_pricing_sheets.id
		)`).
		Order("supplier_pricing_sheets.id desc").
		First(&sheet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sheet, nil
}

// GetActivePricingSheetBySupplierIdBinding returns the first active supplier pricing sheet
// bound to the given channel for a specific supplier.
// Priority: exact channel binding > universal (no explicit bindings).
// Requires: supplier enabled, status=active, current time within [start_time, end_time].
func GetActivePricingSheetBySupplierIdBinding(supplierId int, channelId int) (*SupplierPricingSheet, error) {
	if channelId <= 0 {
		return nil, nil
	}
	now := time.Now().Unix()
	tf := timeFilterExpr()

	// 1. Try exact channel binding match
	var sheet SupplierPricingSheet
	err := DB.
		Joins("JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
		Joins("JOIN supplier_pricing_sheet_channels ch_bind ON ch_bind.pricing_sheet_id = supplier_pricing_sheets.id").
		Where("supplier_pricing_sheets.supplier_id = ?", supplierId).
		Where("ch_bind.channel_id = ?", channelId).
		Where("supplier_pricing_sheets.status = ?", SupplierPricingSheetStatusActive).
		Where(tf, now, now).
		Where("suppliers.status = ?", SupplierStatusEnabled).
		Order("supplier_pricing_sheets.id desc").
		First(&sheet).Error
	if err == nil {
		return &sheet, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 2. Fall back to universal sheet (channel_id = 0)
	err = DB.
		Joins("JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
		Where("supplier_pricing_sheets.supplier_id = ?", supplierId).
		Where("supplier_pricing_sheets.channel_id = ?", 0).
		Where("supplier_pricing_sheets.status = ?", SupplierPricingSheetStatusActive).
		Where(tf, now, now).
		Where("suppliers.status = ?", SupplierStatusEnabled).
		Where(`NOT EXISTS (
			SELECT 1 FROM supplier_pricing_sheet_channels ch_bind2
			WHERE ch_bind2.pricing_sheet_id = supplier_pricing_sheets.id
		)`).
		Order("supplier_pricing_sheets.id desc").
		First(&sheet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sheet, nil
}
