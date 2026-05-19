package model

import (
	"errors"

	"gorm.io/gorm"
)

// SupplierPricingItem represents a pricing item (batch of models with same discount) within a supplier pricing sheet.
type SupplierPricingItem struct {
	Id             int            `json:"id" gorm:"primaryKey;autoIncrement"`
	PricingSheetId int            `json:"pricing_sheet_id"`
	VendorType     string         `json:"vendor_type"`
	Models         ModelsSlice    `json:"models" gorm:"type:json"`
	DiscountType   string         `json:"discount_type"`
	DiscountValue  float64        `json:"discount_value"`
	Remark         string         `json:"remark"`
}

func (s *SupplierPricingItem) TableName() string {
	return "supplier_pricing_items"
}

// Create inserts a new pricing item.
func (s *SupplierPricingItem) Create() error {
	return DB.Create(s).Error
}

// Update updates an existing pricing item.
func (s *SupplierPricingItem) Update() error {
	return DB.Save(s).Error
}

// Delete deletes a pricing item by ID.
func (s *SupplierPricingItem) Delete() error {
	return DB.Delete(s).Error
}

// GetSupplierPricingItemById retrieves a supplier pricing item by ID.
// Returns nil, nil if not found.
func GetSupplierPricingItemById(id int) (*SupplierPricingItem, error) {
	var item SupplierPricingItem
	err := DB.First(&item, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// GetSupplierPricingItemsBySheetId returns all supplier pricing items for a pricing sheet.
func GetSupplierPricingItemsBySheetId(sheetId int) ([]*SupplierPricingItem, error) {
	var items []*SupplierPricingItem
	err := DB.Where("pricing_sheet_id = ?", sheetId).Find(&items).Error
	return items, err
}

// GetSupplierPricingItemBySheetIdAndModel returns the supplier pricing item whose models JSON array contains the given model name.
// Returns nil, nil if not found.
func GetSupplierPricingItemBySheetIdAndModel(sheetId int, modelName string) (*SupplierPricingItem, error) {
	items, err := GetSupplierPricingItemsBySheetId(sheetId)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		for _, m := range item.Models {
			if m == modelName {
				return item, nil
			}
		}
	}
	return nil, nil
}

// DeleteSupplierPricingItem deletes a pricing item by ID.
func DeleteSupplierPricingItem(id int) error {
	return DB.Delete(&SupplierPricingItem{}, id).Error
}
