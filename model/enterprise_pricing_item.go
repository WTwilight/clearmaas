package model

import (
	"errors"

	"gorm.io/gorm"
)

// Discount type constants
const (
	DiscountTypeRatio      = "ratio"
	DiscountTypeFixedPrice = "fixed_price"
	DiscountTypePerCall   = "per_call"
)

// EnterprisePricingItem represents a pricing item (model discount) within a pricing sheet.
type EnterprisePricingItem struct {
	Id             int     `json:"id" gorm:"primaryKey;autoIncrement"`
	PricingSheetId int     `json:"pricing_sheet_id" gorm:"uniqueIndex:idx_sheet_model"`
	Model          string  `json:"model" gorm:"uniqueIndex:idx_sheet_model"`
	DiscountType   string  `json:"discount_type"`
	DiscountValue  float64 `json:"discount_value"`
	Remark         string  `json:"remark"`
}

func (e *EnterprisePricingItem) TableName() string {
	return "enterprise_pricing_items"
}

// Create inserts a new pricing item.
func (e *EnterprisePricingItem) Create() error {
	return DB.Create(e).Error
}

// Update updates an existing pricing item.
func (e *EnterprisePricingItem) Update() error {
	return DB.Save(e).Error
}

// Delete deletes a pricing item by ID.
func (e *EnterprisePricingItem) Delete() error {
	return DB.Delete(e).Error
}

// GetPricingItemById retrieves a pricing item by ID.
func GetPricingItemById(id int) (*EnterprisePricingItem, error) {
	var item EnterprisePricingItem
	err := DB.First(&item, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// GetPricingItemsBySheetId returns all pricing items for a pricing sheet.
func GetPricingItemsBySheetId(sheetId int) ([]*EnterprisePricingItem, error) {
	var items []*EnterprisePricingItem
	err := DB.Where("pricing_sheet_id = ?", sheetId).Find(&items).Error
	return items, err
}

// GetPricingItemBySheetIdAndModel returns the pricing item for a specific sheet and model.
func GetPricingItemBySheetIdAndModel(sheetId int, modelName string) (*EnterprisePricingItem, error) {
	var item EnterprisePricingItem
	err := DB.Where("pricing_sheet_id = ? AND model = ?", sheetId, modelName).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// DeletePricingItem deletes a pricing item by ID.
func DeletePricingItem(id int) error {
	return DB.Delete(&EnterprisePricingItem{}, id).Error
}
