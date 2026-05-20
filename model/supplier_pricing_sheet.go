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

// timeFilterExpr returns the raw SQL time validity expression for pricing sheets.
// start_time = 0 means "no start restriction"
// end_time = 0 means "no end restriction" (permanently valid)
func timeFilterExpr() string {
	return "(start_time = 0 OR start_time <= ?) AND (end_time = 0 OR end_time >= ?)"
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

// GetPricingSheetsBySupplierId returns all pricing sheets for a supplier with channel name.
func GetPricingSheetsBySupplierId(supplierId int) ([]*SupplierPricingSheetWithSupplier, error) {
	var sheets []*SupplierPricingSheetWithSupplier
	err := DB.Table("supplier_pricing_sheets").
		Select("supplier_pricing_sheets.*, channels.name as channel_name").
		Joins("LEFT JOIN channels ON channels.id = supplier_pricing_sheets.channel_id").
		Where("supplier_pricing_sheets.supplier_id = ?", supplierId).
		Order("supplier_pricing_sheets.id desc").
		Find(&sheets).Error
	if err != nil {
		return nil, err
	}
	PopulateChannelBindings(sheets)
	return sheets, nil
}

// GetActivePricingSheetsBySupplierId returns active (status=1, within time range, supplier enabled) sheets.
// For multiple active sheets, only the latest one (by id desc) is returned.
func GetActivePricingSheetsBySupplierId(supplierId int) ([]*SupplierPricingSheet, error) {
	if !IsSupplierEnabled(supplierId) {
		return []*SupplierPricingSheet{}, nil
	}
	var sheets []*SupplierPricingSheet
	now := time.Now().Unix()
	err := DB.Where("supplier_id = ? AND status = ? AND "+timeFilterExpr(),
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

	query := DB.Where("status = ? AND "+timeFilterExpr()+" AND supplier_id > 0",
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

// GetFirstActivePricingSheetByChannelIdOnly returns the active sheet matching channelId without requiring supplierId.
// Uses the new supplier_pricing_sheet_channels binding table.
// Priority: channel binding in supplier_pricing_sheet_channels > universal (channel_id = 0, no explicit bindings).
// Requires: supplier enabled, status=active, current time within [start_time, end_time]
func GetFirstActivePricingSheetByChannelIdOnly(channelId int) (*SupplierPricingSheet, error) {
	return GetActivePricingSheetByChannelIdBinding(channelId)
}

// GetFirstActivePricingSheetByChannelId returns the first active sheet matching channelId for a supplier.
// Uses the new supplier_pricing_sheet_channels binding table.
// Priority: channel binding in supplier_pricing_sheet_channels > universal (channel_id = 0, no explicit bindings).
// Requires: supplier enabled, status=active, current time within [start_time, end_time]
func GetFirstActivePricingSheetByChannelId(supplierId int, channelId int) (*SupplierPricingSheet, error) {
	return GetActivePricingSheetBySupplierIdBinding(supplierId, channelId)
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

// SupplierPricingSheetWithSupplier represents a pricing sheet with supplier and channel info.
type SupplierPricingSheetWithSupplier struct {
	SupplierPricingSheet
	SupplierName string   `json:"supplier_name"`
	ChannelName  string   `json:"channel_name"`
	ChannelIds   []int    `json:"channel_ids" gorm:"-"`
	ChannelNames []string `json:"channel_names" gorm:"-"`
}

// PopulateChannelBindings populates ChannelIds and ChannelNames for a list of sheets.
func PopulateChannelBindings(sheets []*SupplierPricingSheetWithSupplier) {
	for _, sheet := range sheets {
		ids, err := GetSupplierPricingSheetChannelIds(sheet.Id)
		if err != nil || len(ids) == 0 {
			sheet.ChannelIds = []int{}
			sheet.ChannelNames = []string{}
			continue
		}
		sheet.ChannelIds = ids
		channels, err := GetSupplierPricingSheetChannels(sheet.Id)
		if err != nil || len(channels) == 0 {
			sheet.ChannelNames = []string{}
			continue
		}
		names := make([]string, 0, len(channels))
		for _, ch := range channels {
			if ch != nil {
				names = append(names, ch.Name)
			}
		}
		sheet.ChannelNames = names
	}
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
		Select("supplier_pricing_sheets.*, suppliers.name as supplier_name, channels.name as channel_name").
		Joins("LEFT JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
		Joins("LEFT JOIN channels ON channels.id = supplier_pricing_sheets.channel_id").
		Order("supplier_pricing_sheets.id desc").
		Offset(offset).
		Limit(pageSize).
		Find(&sheets).Error
	if err != nil {
		return nil, 0, err
	}
	PopulateChannelBindings(sheets)
	return sheets, total, nil
}

// GetPricingSheetByIdWithSupplier returns a pricing sheet with supplier and channel info by ID.
func GetPricingSheetByIdWithSupplier(id int) (*SupplierPricingSheetWithSupplier, error) {
	var sheet SupplierPricingSheetWithSupplier
	err := DB.Table("supplier_pricing_sheets").
		Select("supplier_pricing_sheets.*, suppliers.name as supplier_name, channels.name as channel_name").
		Joins("LEFT JOIN suppliers ON suppliers.id = supplier_pricing_sheets.supplier_id").
		Joins("LEFT JOIN channels ON channels.id = supplier_pricing_sheets.channel_id").
		Where("supplier_pricing_sheets.id = ?", id).
		First(&sheet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	PopulateChannelBindings([]*SupplierPricingSheetWithSupplier{&sheet})
	return &sheet, nil
}

// DeleteSupplierPricingSheet deletes a pricing sheet by ID.
func DeleteSupplierPricingSheet(id int) error {
	return DB.Delete(&SupplierPricingSheet{}, id).Error
}
