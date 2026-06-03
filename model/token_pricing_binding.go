package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// TokenPricingModelBinding links a token+model to a specific pricing sheet.
// When ModelLimitsEnabled=true on the token, only models in this binding table can be used.
type TokenPricingModelBinding struct {
	Id            int            `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId        int            `json:"user_id" gorm:"index"`
	TokenId       int            `json:"token_id" gorm:"index"`
	PricingSheetId int           `json:"pricing_sheet_id" gorm:"index"`
	Model         string         `json:"model" gorm:"type:varchar(128)"`
	CreatedAt     int64         `json:"created_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (b *TokenPricingModelBinding) TableName() string {
	return "token_pricing_model_bindings"
}

// Create inserts a new binding record.
func (b *TokenPricingModelBinding) Create() error {
	return DB.Create(b).Error
}

// CreateWithTx inserts a new binding record within an existing transaction.
func (b *TokenPricingModelBinding) CreateWithTx(tx *gorm.DB) error {
	return tx.Create(b).Error
}

// Delete removes the binding record.
func (b *TokenPricingModelBinding) Delete() error {
	return DB.Delete(b).Error
}

// DeleteWithTx removes the binding record within an existing transaction.
func (b *TokenPricingModelBinding) DeleteWithTx(tx *gorm.DB) error {
	return tx.Delete(b).Error
}

// GetTokenPricingModelBindingById retrieves a binding by its ID.
// Returns (binding, nil) if found, (nil, nil) if not found, (nil, err) on database error.
func GetTokenPricingModelBindingById(id int) (*TokenPricingModelBinding, error) {
	var binding TokenPricingModelBinding
	err := DB.Take(&binding, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

// GetTokenPricingModelBindingsByIdTx retrieves a binding by its ID within a transaction.
// Returns (binding, nil) if found, (nil, nil) if not found, (nil, err) on database error.
func GetTokenPricingModelBindingByIdTx(id int, tx *gorm.DB) (*TokenPricingModelBinding, error) {
	var binding TokenPricingModelBinding
	err := tx.Take(&binding, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

// GetTokenPricingModelBindings returns all bindings for a given token.
func GetTokenPricingModelBindings(tokenId int) ([]*TokenPricingModelBinding, error) {
	var bindings []*TokenPricingModelBinding
	err := DB.Where("token_id = ?", tokenId).Order("id asc").Find(&bindings).Error
	return bindings, err
}

// GetTokenPricingModelBindingsTx returns all bindings for a given token within a transaction.
func GetTokenPricingModelBindingsTx(tokenId int, tx *gorm.DB) ([]*TokenPricingModelBinding, error) {
	var bindings []*TokenPricingModelBinding
	err := tx.Where("token_id = ?", tokenId).Order("id asc").Find(&bindings).Error
	return bindings, err
}

// GetTokenPricingModelBindingByTokenAndModel returns the binding for a specific token and model.
// Returns (binding, nil) if found, (nil, nil) if not found, (nil, err) on database error.
// This method distinguishes between "not found" and "error" so callers can decide whether to fall back.
func GetTokenPricingModelBindingByTokenAndModel(tokenId int, model string) (*TokenPricingModelBinding, error) {
	var binding TokenPricingModelBinding
	err := DB.Where("token_id = ? AND model = ?", tokenId, model).First(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

// GetTokenPricingModelBindingByTokenAndModelTx returns the binding for a specific token and model within a transaction.
func GetTokenPricingModelBindingByTokenAndModelTx(tokenId int, model string, tx *gorm.DB) (*TokenPricingModelBinding, error) {
	var binding TokenPricingModelBinding
	err := tx.Where("token_id = ? AND model = ?", tokenId, model).First(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

// DeleteTokenPricingModelBindings hard-deletes all bindings for a given token.
// Uses Unscoped() to bypass GORM's soft-delete so the UNIQUE constraint
// (token_id, model) does not block re-inserting the same model for this token.
func DeleteTokenPricingModelBindings(tokenId int, tx *gorm.DB) error {
	return tx.Unscoped().Where("token_id = ?", tokenId).Delete(&TokenPricingModelBinding{}).Error
}

// CountTokenBindingsBySheetId returns the number of bindings referencing a pricing sheet.
// Used for delete protection: a sheet with active bindings cannot be deleted.
func CountTokenBindingsBySheetId(sheetId int) (int64, error) {
	var count int64
	err := DB.Model(&TokenPricingModelBinding{}).Where("pricing_sheet_id = ?", sheetId).Count(&count).Error
	return count, err
}

// CountTokenBindingsBySheetIdTx returns the number of bindings referencing a pricing sheet within a transaction.
func CountTokenBindingsBySheetIdTx(sheetId int, tx *gorm.DB) (int64, error) {
	var count int64
	err := tx.Model(&TokenPricingModelBinding{}).Where("pricing_sheet_id = ?", sheetId).Count(&count).Error
	return count, err
}

// SyncTokenPricingModelBindings replaces all bindings for a token with a new set.
// This is used for atomic updates: delete all old bindings, then insert new ones.
// All operations run within a single transaction. Returns an error if any step fails.
func SyncTokenPricingModelBindings(userId int, tokenId int, models []string, pricingSheetId int, tx *gorm.DB) error {
	// Step 1: Delete all existing bindings for this token
	if err := tx.Where("token_id = ?", tokenId).Delete(&TokenPricingModelBinding{}).Error; err != nil {
		return err
	}

	// Step 2: Insert new bindings
	now := time.Now().Unix()
	for _, modelName := range models {
		binding := &TokenPricingModelBinding{
			UserId:         userId,
			TokenId:        tokenId,
			PricingSheetId: pricingSheetId,
			Model:          modelName,
			CreatedAt:      now,
		}
		if err := tx.Create(binding).Error; err != nil {
			return err
		}
	}

	return nil
}

// UpsertTokenPricingModelBinding inserts or updates a single binding.
// If a binding for (token_id, model) already exists, it updates the pricing_sheet_id.
// Returns (binding, error).
func UpsertTokenPricingModelBinding(userId int, tokenId int, model string, pricingSheetId int, tx *gorm.DB) error {
	var existing TokenPricingModelBinding
	err := tx.Where("token_id = ? AND model = ?", tokenId, model).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	now := time.Now().Unix()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Insert
		return tx.Create(&TokenPricingModelBinding{
			UserId:         userId,
			TokenId:        tokenId,
			PricingSheetId: pricingSheetId,
			Model:          model,
			CreatedAt:      now,
		}).Error
	}

	// Update
	return tx.Model(&existing).Updates(map[string]interface{}{
		"user_id":          userId,
		"pricing_sheet_id": pricingSheetId,
	}).Error
}

// GetTokenPricingBindingModels returns all distinct model names bound to a given token.
func GetTokenPricingBindingModels(tokenId int) ([]string, error) {
	var models []string
	err := DB.Model(&TokenPricingModelBinding{}).
		Where("token_id = ?", tokenId).
		Pluck("model", &models).Error
	return models, err
}

// DeleteTokenPricingModelBindingsBatch hard-deletes all bindings for multiple tokens.
// Uses Unscoped() to bypass GORM's soft-delete.
func DeleteTokenPricingModelBindingsBatch(tokenIds []int, tx *gorm.DB) error {
	if len(tokenIds) == 0 {
		return nil
	}
	return tx.Unscoped().Where("token_id IN ?", tokenIds).Delete(&TokenPricingModelBinding{}).Error
}
