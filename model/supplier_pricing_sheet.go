package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// SupplierPricingSheet status constants
const (
	SupplierPricingSheetStatusActive   = 1
	SupplierPricingSheetStatusInactive = 0
)

// SupplierPricingSheet represents a pricing sheet for a supplier.
type SupplierPricingSheet struct {
	Id         int   `json:"id" gorm:"primaryKey;autoIncrement"`
	SupplierId int   `json:"supplier_id"`
	ChannelId  int   `json:"channel_id"`
	Name       string `json:"name"`
	Status     int    `json:"status"`
	StartTime  int64  `json:"start_time"`
	EndTime    int64  `json:"end_time"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

func (s *SupplierPricingSheet) TableName() string {
	return "supplier_pricing_sheets"
}

// Create inserts a new pricing sheet.
func (s *SupplierPricingSheet) Create() error {
	return DB.Create(s).Error
}

// Update updates an existing pricing sheet.
func (s *SupplierPricingSheet) Update() error {
	s.UpdatedAt = time.Now().Unix()
	return DB.Save(s).Error
}

// Delete deletes a pricing sheet by ID.
func (s *SupplierPricingSheet) Delete() error {
	return DB.Delete(s).Error
}

// GetSupplierPricingSheetById retrieves a supplier pricing sheet by ID.
// Returns nil, nil if not found.
func GetSupplierPricingSheetById(id int) (*SupplierPricingSheet, error) {
	var sheet SupplierPricingSheet
	err := DB.First(&sheet, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sheet, nil
}

// GetPricingSheetsBySupplierId returns all pricing sheets for a supplier.
func GetPricingSheetsBySupplierId(supplierId int) ([]*SupplierPricingSheet, error) {
	var sheets []*SupplierPricingSheet
	err := DB.Where("supplier_id = ?", supplierId).Order("id desc").Find(&sheets).Error
	return sheets, err
}

// GetActivePricingSheetsBySupplierId returns active (status=1, within time range, supplier enabled) sheets.
// For multiple active sheets, only the latest one (by id desc) is returned.
func GetActivePricingSheetsBySupplierId(supplierId int) ([]*SupplierPricingSheet, error) {
	if !IsSupplierEnabled(supplierId) {
		return []*SupplierPricingSheet{}, nil
	}
	var sheets []*SupplierPricingSheet
	now := time.Now().Unix()
	err := DB.Where("supplier_id = ? AND status = ? AND start_time <= ? AND end_time >= ?",
		supplierId, SupplierPricingSheetStatusActive, now, now).
		Order("id desc").
		Limit(1).
		Find(&sheets).Error
	return sheets, err
}

// GetActivePricingSheetsByChannelId returns active pricing sheets that match by channel_id or supplier_id.
// channelId = 0 means universal sheet (matches any channel).
func GetActivePricingSheetsByChannelId(channelId int) ([]*SupplierPricingSheet, error) {
	var sheets []*SupplierPricingSheet
	now := time.Now().Unix()

	query := DB.Where("status = ? AND start_time <= ? AND end_time >= ? AND supplier_id > 0",
		SupplierPricingSheetStatusActive, now, now)

	if channelId == 0 {
		query = query.Where("channel_id = ?", 0)
	} else {
		query = query.Where("(channel_id = ? OR channel_id = 0)", channelId)
	}

	err := query.
		Joins("JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
		Where("suppliers.status = ?", SupplierStatusEnabled).
		Order("supplier_pricing_sheets.id desc").
		Find(&sheets).Error
	return sheets, err
}

// GetFirstActivePricingSheetByChannelId returns the first active sheet matching channelId for a supplier.
// Priority: channel_id = channelId > channel_id = 0 (universal)
// Requires: supplier enabled, status=active, current time within [start_time, end_time]
func GetFirstActivePricingSheetByChannelId(supplierId int, channelId int) (*SupplierPricingSheet, error) {
	now := time.Now().Unix()

	// First try exact channel match (channel_id > 0)
	if channelId > 0 {
		var sheet SupplierPricingSheet
		err := DB.
			Joins("JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
			Where("supplier_pricing_sheets.supplier_id = ?", supplierId).
			Where("supplier_pricing_sheets.channel_id = ?", channelId).
			Where("supplier_pricing_sheets.status = ?", SupplierPricingSheetStatusActive).
			Where("supplier_pricing_sheets.start_time <= ?", now).
			Where("supplier_pricing_sheets.end_time >= ?", now).
			Where("suppliers.status = ?", SupplierStatusEnabled).
			Order("supplier_pricing_sheets.id desc").
			First(&sheet).Error
		if err == nil {
			return &sheet, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Fall back to universal sheet (channel_id = 0)
	var sheet SupplierPricingSheet
	err := DB.
		Joins("JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
		Where("supplier_pricing_sheets.supplier_id = ?", supplierId).
		Where("supplier_pricing_sheets.channel_id = ?", 0).
		Where("supplier_pricing_sheets.status = ?", SupplierPricingSheetStatusActive).
		Where("supplier_pricing_sheets.start_time <= ?", now).
		Where("supplier_pricing_sheets.end_time >= ?", now).
		Where("suppliers.status = ?", SupplierStatusEnabled).
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

// GetFirstActivePricingSheetBySupplierId returns the first active sheet for a supplier.
// Requires: supplier enabled, status=active, current time within [start_time, end_time]
func GetFirstActivePricingSheetBySupplierId(supplierId int) (*SupplierPricingSheet, error) {
	sheets, err := GetActivePricingSheetsBySupplierId(supplierId)
	if err != nil {
		return nil, err
	}
	if len(sheets) == 0 {
		return nil, nil
	}
	return sheets[0], nil
}

// GetPricingSheetsByChannelId returns all pricing sheets for a specific channel (including universal).
func GetPricingSheetsByChannelId(channelId int) ([]*SupplierPricingSheet, error) {
	var sheets []*SupplierPricingSheet
	err := DB.Where("channel_id = ?", channelId).Order("id desc").Find(&sheets).Error
	return sheets, err
}

// SupplierPricingSheetWithSupplier represents a pricing sheet with supplier info.
type SupplierPricingSheetWithSupplier struct {
	SupplierPricingSheet
	SupplierName string `json:"supplier_name"`
}

// GetAllSupplierPricingSheets returns all supplier pricing sheets across all suppliers with pagination.
func GetAllSupplierPricingSheets(page, pageSize int) ([]*SupplierPricingSheetWithSupplier, int64, error) {
	var sheets []*SupplierPricingSheetWithSupplier
	var total int64

	if err := DB.Model(&SupplierPricingSheet{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := DB.Table("supplier_pricing_sheets").
		Select("supplier_pricing_sheets.*, suppliers.name as supplier_name").
		Joins("LEFT JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
		Order("supplier_pricing_sheets.id desc").
		Offset(offset).
		Limit(pageSize).
		Find(&sheets).Error
	if err != nil {
		return nil, 0, err
	}

	return sheets, total, nil
}

// GetPricingSheetByIdWithSupplier returns a pricing sheet with supplier info by ID.
func GetPricingSheetByIdWithSupplier(id int) (*SupplierPricingSheetWithSupplier, error) {
	var sheet SupplierPricingSheetWithSupplier
	err := DB.Table("supplier_pricing_sheets").
		Select("supplier_pricing_sheets.*, suppliers.name as supplier_name").
		Joins("LEFT JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
		Where("supplier_pricing_sheets.id = ?", id).
		First(&sheet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sheet, nil
}

// DeleteSupplierPricingSheet deletes a pricing sheet by ID.
func DeleteSupplierPricingSheet(id int) error {
	return DB.Delete(&SupplierPricingSheet{}, id).Error
}

// GetFirstActivePricingSheetByChannelIdOnly returns the active sheet matching channelId without requiring supplierId.
// Priority: channel_id = channelId > channel_id = 0 (universal)
// Requires: supplier enabled, status=active, current time within [start_time, end_time]
func GetFirstActivePricingSheetByChannelIdOnly(channelId int) (*SupplierPricingSheet, error) {
	now := time.Now().Unix()

	// 1. Try exact channel match (channel_id > 0)
	if channelId > 0 {
		var sheet SupplierPricingSheet
		err := DB.
			Joins("JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
			Where("supplier_pricing_sheets.channel_id = ?", channelId).
			Where("supplier_pricing_sheets.status = ?", SupplierPricingSheetStatusActive).
			Where("supplier_pricing_sheets.start_time <= ?", now).
			Where("supplier_pricing_sheets.end_time >= ?", now).
			Where("suppliers.status = ?", SupplierStatusEnabled).
			Order("supplier_pricing_sheets.id desc").
			First(&sheet).Error
		if err == nil {
			return &sheet, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// 2. Fall back to universal sheet (channel_id = 0)
	var sheet SupplierPricingSheet
	err := DB.
		Joins("JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
		Where("supplier_pricing_sheets.channel_id = ?", 0).
		Where("supplier_pricing_sheets.status = ?", SupplierPricingSheetStatusActive).
		Where("supplier_pricing_sheets.start_time <= ?", now).
		Where("supplier_pricing_sheets.end_time >= ?", now).
		Where("suppliers.status = ?", SupplierStatusEnabled).
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
