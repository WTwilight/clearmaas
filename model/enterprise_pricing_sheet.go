package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// PricingSheet status constants
const (
	PricingSheetStatusActive   = 1
	PricingSheetStatusInactive = 0
)

// EnterprisePricingSheet represents a pricing sheet for an enterprise.
type EnterprisePricingSheet struct {
	Id           int       `json:"id" gorm:"primaryKey;autoIncrement"`
	EnterpriseId int       `json:"enterprise_id"`
	Name         string    `json:"name"`
	Status       int       `json:"status"`
	StartTime    int64     `json:"start_time"`
	EndTime      int64     `json:"end_time"`
	CreatedAt    int64     `json:"created_at"`
	UpdatedAt    int64     `json:"updated_at"`
}

func (e *EnterprisePricingSheet) TableName() string {
	return "enterprise_pricing_sheets"
}

// Create inserts a new pricing sheet.
func (e *EnterprisePricingSheet) Create() error {
	return DB.Create(e).Error
}

// Update updates an existing pricing sheet.
func (e *EnterprisePricingSheet) Update() error {
	e.UpdatedAt = time.Now().Unix()
	return DB.Save(e).Error
}

// Delete deletes a pricing sheet by ID.
func (e *EnterprisePricingSheet) Delete() error {
	return DB.Delete(e).Error
}

// GetPricingSheetById retrieves a pricing sheet by ID.
func GetPricingSheetById(id int) (*EnterprisePricingSheet, error) {
	var sheet EnterprisePricingSheet
	err := DB.First(&sheet, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sheet, nil
}

// GetPricingSheetsByEnterpriseId returns all pricing sheets for an enterprise.
func GetPricingSheetsByEnterpriseId(enterpriseId int) ([]*EnterprisePricingSheet, error) {
	var sheets []*EnterprisePricingSheet
	err := DB.Where("enterprise_id = ?", enterpriseId).Order("id desc").Find(&sheets).Error
	return sheets, err
}

// GetActivePricingSheetsByEnterpriseId returns active (status=1, within time range, enterprise enabled) sheets.
func GetActivePricingSheetsByEnterpriseId(enterpriseId int) ([]*EnterprisePricingSheet, error) {
	if !IsEnterpriseEnabled(enterpriseId) {
		return []*EnterprisePricingSheet{}, nil
	}
	var sheets []*EnterprisePricingSheet
	now := time.Now().Unix()
	err := DB.Where("enterprise_id = ? AND status = ? AND start_time <= ? AND end_time >= ?",
		enterpriseId, PricingSheetStatusActive, now, now).
		Order("id desc").
		Find(&sheets).Error
	return sheets, err
}

// GetFirstActivePricingSheetByEnterpriseId returns the first active sheet or nil.
func GetFirstActivePricingSheetByEnterpriseId(enterpriseId int) (*EnterprisePricingSheet, error) {
	sheets, err := GetActivePricingSheetsByEnterpriseId(enterpriseId)
	if err != nil {
		return nil, err
	}
	if len(sheets) == 0 {
		return nil, nil
	}
	return sheets[0], nil
}

// DeletePricingSheet deletes a pricing sheet by ID.
// Returns an error if the pricing sheet has associated pricing items.
func DeletePricingSheet(id int) error {
	items, err := GetPricingItemsBySheetId(id)
	if err != nil {
		return err
	}
	if len(items) > 0 {
		return errors.New("cannot delete pricing sheet with associated pricing items")
	}
	return DB.Delete(&EnterprisePricingSheet{}, id).Error
}

// PricingSheetWithEnterprise represents a pricing sheet with enterprise info.
type PricingSheetWithEnterprise struct {
	EnterprisePricingSheet
	EnterpriseName string `json:"enterprise_name"`
}

// GetAllPricingSheets returns all pricing sheets across all enterprises with pagination and optional enterprise filter.
func GetAllPricingSheets(page, pageSize int, enterpriseIdStr string) ([]*PricingSheetWithEnterprise, int64, error) {
	var sheets []*PricingSheetWithEnterprise
	var total int64

	query := DB.Model(&EnterprisePricingSheet{})
	if enterpriseIdStr != "" {
		query = query.Where("enterprise_pricing_sheets.enterprise_id = ?", enterpriseIdStr)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	dbQuery := DB.Table("enterprise_pricing_sheets").
		Select("enterprise_pricing_sheets.*, enterprises.name as enterprise_name").
		Joins("LEFT JOIN enterprises ON enterprises.id = enterprise_pricing_sheets.enterprise_id")
	if enterpriseIdStr != "" {
		dbQuery = dbQuery.Where("enterprise_pricing_sheets.enterprise_id = ?", enterpriseIdStr)
	}
	err := dbQuery.
		Order("enterprise_pricing_sheets.id desc").
		Offset(offset).
		Limit(pageSize).
		Find(&sheets).Error
	if err != nil {
		return nil, 0, err
	}

	return sheets, total, nil
}
