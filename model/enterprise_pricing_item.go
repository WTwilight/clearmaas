package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// JSON models field — stores []string as JSON in DB
type ModelsSlice []string

func (m ModelsSlice) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

func (m *ModelsSlice) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan ModelsSlice: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, m)
}

// Discount type constants
const (
	DiscountTypeRatio      = "ratio"
	DiscountTypeFixedPrice = "fixed_price"
	DiscountTypePerCall   = "per_call"
)

// EnterprisePricingItem represents a pricing item (batch of models with same discount) within a pricing sheet.
type EnterprisePricingItem struct {
	Id             int          `json:"id" gorm:"primaryKey;autoIncrement"`
	PricingSheetId int          `json:"pricing_sheet_id"`
	VendorType     string       `json:"vendor_type"`
	Models         ModelsSlice  `json:"models" gorm:"type:json"`
	DiscountType   string       `json:"discount_type"`
	DiscountValue  float64      `json:"discount_value"`
	Remark         string       `json:"remark"`
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
	err := DB.Take(&item, id).Error
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

// GetPricingItemsBySheetIdTx returns all pricing items for a pricing sheet within a transaction.
func GetPricingItemsBySheetIdTx(sheetId int, tx *gorm.DB) ([]*EnterprisePricingItem, error) {
	var items []*EnterprisePricingItem
	err := tx.Where("pricing_sheet_id = ?", sheetId).Find(&items).Error
	return items, err
}

// GetPricingItemBySheetIdAndModelName returns the pricing item whose models JSON array contains the given model name.
func GetPricingItemBySheetIdAndModelName(sheetId int, modelName string) (*EnterprisePricingItem, error) {
	var items []*EnterprisePricingItem
	err := DB.Where("pricing_sheet_id = ?", sheetId).Find(&items).Error
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

// DeletePricingItem deletes a pricing item by ID.
func DeletePricingItem(id int) error {
	return DB.Delete(&EnterprisePricingItem{}, id).Error
}
