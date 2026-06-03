package controller

import (
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// GetTokenPricingModels returns the pricing model bindings for a specific token.
func GetTokenPricingModels(c *gin.Context) {
	tokenId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")

	// Verify token belongs to user
	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	bindings, err := model.GetTokenPricingModelBindings(token.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// Enrich bindings with pricing sheet info
	type BindingWithSheet struct {
		Id            int    `json:"id"`
		TokenId       int    `json:"token_id"`
		Model         string `json:"model"`
		PricingSheetId int   `json:"pricing_sheet_id"`
		SheetName     string `json:"sheet_name"`
		DiscountType  string `json:"discount_type"`
		DiscountValue float64 `json:"discount_value"`
		CreatedAt     int64  `json:"created_at"`
	}

	result := make([]BindingWithSheet, 0, len(bindings))
	for _, b := range bindings {
		sheet, _ := model.GetPricingSheetById(b.PricingSheetId)
		pricingItem, _ := model.GetPricingItemBySheetIdAndModelName(b.PricingSheetId, b.Model)
		sheetName := ""
		discountType := ""
		discountValue := 0.0
		if sheet != nil {
			sheetName = sheet.Name
		}
		if pricingItem != nil {
			discountType = pricingItem.DiscountType
			discountValue = pricingItem.DiscountValue
		}
		result = append(result, BindingWithSheet{
			Id:             b.Id,
			TokenId:        b.TokenId,
			Model:          b.Model,
			PricingSheetId: b.PricingSheetId,
			SheetName:      sheetName,
			DiscountType:   discountType,
			DiscountValue:  discountValue,
			CreatedAt:      b.CreatedAt,
		})
	}

	common.ApiSuccess(c, gin.H{
		"token":      token,
		"bindings":   result,
	})
}

// BindTokenPricingModelsRequest is the request body for binding pricing models to a token.
type BindTokenPricingModelsRequest struct {
	ModelLimitsEnabled bool `json:"model_limits_enabled"`
	ModelLimits        string `json:"model_limits"`
	Bindings           []struct {
		SheetId int    `json:"sheet_id"`
		Model   string `json:"model"`
	} `json:"bindings"`
}

// BindTokenPricingModels replaces all pricing model bindings for a token (idempotent overwrite).
// All operations are atomic: update token fields + replace all bindings in a single transaction.
func BindTokenPricingModels(c *gin.Context) {
	tokenId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")

	var req BindTokenPricingModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	// Verify token belongs to user
	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	tx := model.DB.Begin()

	// Step 1: Update token ModelLimitsEnabled and ModelLimits
	token.ModelLimitsEnabled = req.ModelLimitsEnabled
	token.ModelLimits = req.ModelLimits
	if err := token.UpdateWithTx(tx); err != nil {
		tx.Rollback()
		common.ApiError(c, err)
		return
	}

	// Step 2: Delete all existing bindings for this token
	if err := model.DeleteTokenPricingModelBindings(token.Id, tx); err != nil {
		tx.Rollback()
		common.ApiError(c, err)
		return
	}

	// Step 3: Insert new bindings
	now := time.Now().Unix()
	for _, b := range req.Bindings {
		binding := &model.TokenPricingModelBinding{
			UserId:         userId,
			TokenId:        token.Id,
			PricingSheetId: b.SheetId,
			Model:          b.Model,
			CreatedAt:      now,
		}
		if err := binding.CreateWithTx(tx); err != nil {
			tx.Rollback()
			common.ApiError(c, err)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{"success": true})
}

// UnbindTokenPricingModels clears all pricing model bindings for a token and disables ModelLimits.
func UnbindTokenPricingModels(c *gin.Context) {
	tokenId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")

	// Verify token belongs to user
	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	tx := model.DB.Begin()

	// Step 1: Disable ModelLimits and clear whitelist
	token.ModelLimitsEnabled = false
	token.ModelLimits = ""
	if err := token.UpdateWithTx(tx); err != nil {
		tx.Rollback()
		common.ApiError(c, err)
		return
	}

	// Step 2: Delete all bindings
	if err := model.DeleteTokenPricingModelBindings(token.Id, tx); err != nil {
		tx.Rollback()
		common.ApiError(c, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{"success": true})
}

// GetAvailablePricingSheets returns the selectable models for the current user.
// This is used by the key creation/edit form to populate the model selector.
func GetAvailablePricingSheets(c *gin.Context) {
	userId := c.GetInt("id")
	models := service.GetSelectableModelsForUser(userId)
	common.ApiSuccess(c, gin.H{
		"models": models,
	})
}
