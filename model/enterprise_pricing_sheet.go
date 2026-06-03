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
	err := DB.Take(&sheet, id).Error
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

// GetFirstActivePricingSheetByEnterpriseIdByType returns the first active pricing sheet
// for the enterprise identified by type (e.g., type='platform').
// DEPRECATED: Use GetAllActivePricingSheetsByEnterpriseIdByType for multi-sheet support.
func GetFirstActivePricingSheetByEnterpriseIdByType(enterpriseType string) (*EnterprisePricingSheet, error) {
	sheets, err := GetAllActivePricingSheetsByEnterpriseIdByType(enterpriseType)
	if err != nil {
		return nil, err
	}
	if len(sheets) == 0 {
		return nil, nil
	}
	return sheets[0], nil
}

// GetAllActivePricingSheetsByEnterpriseIdByType returns all active pricing sheets
// for the enterprise identified by type (e.g., type='platform').
// Sheets are returned ordered by ID descending (highest ID first).
func GetAllActivePricingSheetsByEnterpriseIdByType(enterpriseType string) ([]*EnterprisePricingSheet, error) {
	var sheets []*EnterprisePricingSheet
	now := time.Now().Unix()
	err := DB.
		Joins("JOIN `enterprises` ON `enterprises`.id = enterprise_pricing_sheets.enterprise_id").
		Where("`enterprises`.`ent_type` = ?", enterpriseType).
		Where("enterprise_pricing_sheets.status = ?", PricingSheetStatusActive).
		Where("enterprise_pricing_sheets.start_time <= ? AND enterprise_pricing_sheets.end_time >= ?", now, now).
		Order("enterprise_pricing_sheets.id desc").
		Find(&sheets).Error
	if err != nil {
		return nil, err
	}
	return sheets, nil
}

// DeletePricingSheet deletes a pricing sheet by ID.
// Returns an error if the pricing sheet has associated pricing items or active Token bindings.
// All checks and the delete operation run within a single transaction.
func DeletePricingSheet(id int) error {
	// Cannot delete the platform enterprise's pricing sheet
	if IsPlatformEnterpriseId(id) {
		return errors.New("cannot delete the platform enterprise's pricing sheet")
	}

	tx := DB.Begin()

	// Check for pricing items
	items, err := GetPricingItemsBySheetIdTx(id, tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	if len(items) > 0 {
		tx.Rollback()
		return errors.New("cannot delete pricing sheet with associated pricing items")
	}

	// Check for active Token bindings
	count, err := CountTokenBindingsBySheetIdTx(id, tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	if count > 0 {
		tx.Rollback()
		return errors.New("cannot delete pricing sheet with active token bindings")
	}

	err = tx.Delete(&EnterprisePricingSheet{}, id).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// PricingSheetWithEnterprise represents a pricing sheet with enterprise info.
type PricingSheetWithEnterprise struct {
	EnterprisePricingSheet
	EnterpriseName string `json:"enterprise_name"`
}

// GetPricingSheetTokenBindings returns all unique tokens that have bindings referencing this pricing sheet.
// Returns token info joined with user info and binding timestamps.
func GetPricingSheetTokenBindings(sheetId int) ([]*TokenBindingInfo, error) {
	var results []*TokenBindingInfo
	// Use GROUP BY to get one row per token, and MIN(created_at) for binding timestamp.
	// Use commonGroupCol for cross-DB compatibility since "group" is a reserved keyword.
	err := DB.Table("token_pricing_model_bindings").
		Select("t.id, t.user_id, u.username, t.name, t.status, t.key, t.created_time, t.accessed_time, t."+commonGroupCol+" as token_group, MIN(token_pricing_model_bindings.created_at) as binding_created_at").
		Joins("JOIN tokens t ON t.id = token_pricing_model_bindings.token_id").
		Joins("JOIN users u ON u.id = t.user_id").
		Where("token_pricing_model_bindings.pricing_sheet_id = ?", sheetId).
		Group("t.id, t.user_id, u.username, t.name, t.status, t.key, t.created_time, t.accessed_time, t."+commonGroupCol).
		Order("binding_created_at DESC").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

// TokenBindingInfo holds token info with binding metadata for display.
type TokenBindingInfo struct {
	Id               int    `json:"id"`
	UserId           int    `json:"user_id"`
	Username         string `json:"username"`
	Name             string `json:"name"`
	Status           int    `json:"status"`
	Key              string `json:"key"`
	CreatedTime      int64  `json:"created_time"`
	AccessedTime     int64  `json:"accessed_time"`
	TokenGroup       string `json:"token_group"`
	BindingCreatedAt int64  `json:"binding_created_at"`
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
