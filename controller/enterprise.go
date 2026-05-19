package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// Enterprise CRUD
// ---------------------------------------------------------------------------

func CreateEnterprise(c *gin.Context) {
	var req struct {
		Name   string `json:"name"`
		Status int    `json:"status"`
		Remark string `json:"remark"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid json"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "name is required"})
		return
	}
	if req.Status == 0 {
		req.Status = model.EnterpriseStatusEnabled
	}

	now := time.Now().Unix()
	e := &model.Enterprise{
		Name:      req.Name,
		Status:    req.Status,
		Remark:    req.Remark,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := e.Create(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": e})
}

func ListEnterprise(c *gin.Context) {
	p, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if p < 1 {
		p = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	enterprises, total, err := model.GetEnterprises(p, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"items":     enterprises,
		"total":     total,
		"page":      p,
		"page_size": pageSize,
	}})
}

func GetEnterprise(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}

	e, err := model.GetEnterpriseById(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if e == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "enterprise not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": e})
}

func UpdateEnterprise(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}

	e, err := model.GetEnterpriseById(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if e == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "enterprise not found"})
		return
	}

	var req struct {
		Name   string `json:"name"`
		Status int    `json:"status"`
		Remark string `json:"remark"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid json"})
		return
	}

	if req.Name != "" {
		e.Name = req.Name
	}
	if req.Remark != "" {
		e.Remark = req.Remark
	}
	if req.Status != 0 {
		e.Status = req.Status
	}
	e.UpdatedAt = time.Now().Unix()

	if err := e.Update(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": e})
}

func DeleteEnterprise(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}

	e, err := model.GetEnterpriseById(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if e == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "enterprise not found"})
		return
	}

	if err := model.DeleteEnterprise(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------------------------------------------------------------------------
// Enterprise User Binding
// ---------------------------------------------------------------------------

func BindUsers(c *gin.Context) {
	idStr := c.Param("id")
	enterpriseId, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid enterprise id"})
		return
	}

	e, err := model.GetEnterpriseById(enterpriseId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if e == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "enterprise not found"})
		return
	}

	var req struct {
		UserIds []int `json:"user_ids"`
		UserId  int    `json:"user_id"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid json"})
		return
	}

	// Support both single user_id and array user_ids
	targetUserIds := req.UserIds
	if len(targetUserIds) == 0 && req.UserId > 0 {
		targetUserIds = []int{req.UserId}
	}
	if len(targetUserIds) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "user_id or user_ids is required"})
		return
	}

	now := time.Now().Unix()
	for _, userId := range targetUserIds {
		existing, _ := model.GetUserBinding(userId)
		if existing != nil {
			continue
		}
		binding := &model.EnterpriseUserBinding{
			EnterpriseId: enterpriseId,
			UserId:       userId,
			CreatedAt:    now,
		}
		if err := binding.Create(); err != nil {
			continue
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func UnbindUser(c *gin.Context) {
	userIdStr := c.Param("userId")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid user id"})
		return
	}

	if err := model.DeleteUserBinding(userId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func ListEnterpriseUsers(c *gin.Context) {
	idStr := c.Param("id")
	enterpriseId, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid enterprise id"})
		return
	}

	bindings, err := model.GetBindingsByEnterpriseId(enterpriseId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	// Build a map of user_id to binding for quick lookup
	bindingMap := make(map[int]*model.EnterpriseUserBinding)
	for _, b := range bindings {
		bindingMap[b.UserId] = b
	}

	type UserWithBinding struct {
		UserId     int    `json:"user_id"`
		Username   string `json:"username"`
		DisplayName string `json:"display_name"`
		CreatedAt  int64  `json:"created_at"`
	}

	users := make([]UserWithBinding, 0, len(bindings))
	for _, b := range bindings {
		user, err := model.GetUserById(b.UserId, false)
		if err != nil || user == nil {
			continue
		}
		users = append(users, UserWithBinding{
			UserId:     user.Id,
			Username:   user.Username,
			DisplayName: user.DisplayName,
			CreatedAt:  b.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": users})
}

func ListAllUserBindings(c *gin.Context) {
	bindings, err := model.GetAllUserBindings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": bindings})
}

// ---------------------------------------------------------------------------
// Pricing Sheet CRUD
// ---------------------------------------------------------------------------

func CreatePricingSheet(c *gin.Context) {
	idStr := c.Param("id")
	enterpriseId, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid enterprise id"})
		return
	}

	e, err := model.GetEnterpriseById(enterpriseId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if e == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "enterprise not found"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		Status    int    `json:"status"`
		StartTime int64  `json:"start_time"`
		EndTime   int64  `json:"end_time"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid json"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "name is required"})
		return
	}
	if req.Status == 0 {
		req.Status = model.PricingSheetStatusActive
	}

	now := time.Now().Unix()
	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: enterpriseId,
		Name:         req.Name,
		Status:       req.Status,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := sheet.Create(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sheet})
}

func ListPricingSheet(c *gin.Context) {
	idStr := c.Param("id")
	enterpriseId, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid enterprise id"})
		return
	}

	sheets, err := model.GetPricingSheetsByEnterpriseId(enterpriseId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     sheets,
			"total":     len(sheets),
			"page":      1,
			"page_size": len(sheets),
		},
	})
}

func GetPricingSheet(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}

	sheet, err := model.GetPricingSheetById(sheetId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if sheet == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "pricing sheet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sheet})
}

// ListAllPricingSheets returns all pricing sheets across all enterprises with optional enterprise filter.
func ListAllPricingSheets(c *gin.Context) {
	p, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	enterpriseIdStr := c.Query("enterprise_id")
	if p < 1 {
		p = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	sheets, total, err := model.GetAllPricingSheets(p, pageSize, enterpriseIdStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     sheets,
			"total":     total,
			"page":      p,
			"page_size": pageSize,
		},
	})
}

func UpdatePricingSheet(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}

	sheet, err := model.GetPricingSheetById(sheetId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if sheet == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "pricing sheet not found"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		Status    int    `json:"status"`
		StartTime int64  `json:"start_time"`
		EndTime   int64  `json:"end_time"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid json"})
		return
	}

	if req.Name != "" {
		sheet.Name = req.Name
	}
	if req.Status != 0 {
		sheet.Status = req.Status
	}
	if req.StartTime != 0 {
		sheet.StartTime = req.StartTime
	}
	if req.EndTime != 0 {
		sheet.EndTime = req.EndTime
	}
	sheet.UpdatedAt = time.Now().Unix()

	if err := sheet.Update(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sheet})
}

func DeletePricingSheet(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}

	sheet, err := model.GetPricingSheetById(sheetId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if sheet == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "pricing sheet not found"})
		return
	}

	if err := model.DeletePricingSheet(sheetId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------------------------------------------------------------------------
// Pricing Item CRUD
// ---------------------------------------------------------------------------

func ListPricingItems(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}

	sheet, err := model.GetPricingSheetById(sheetId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if sheet == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "pricing sheet not found"})
		return
	}

	items, err := model.GetPricingItemsBySheetId(sheetId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func AddPricingItem(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}

	sheet, err := model.GetPricingSheetById(sheetId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if sheet == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "pricing sheet not found"})
		return
	}

	var req struct {
		VendorType    string   `json:"vendor_type"`
		Models        []string `json:"models"`
		DiscountType  string   `json:"discount_type"`
		DiscountValue float64  `json:"discount_value"`
		Remark        string   `json:"remark"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid json"})
		return
	}
	if req.VendorType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "vendor_type is required"})
		return
	}
	if len(req.Models) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "models is required and must not be empty"})
		return
	}
	if req.DiscountType == "" {
		req.DiscountType = model.DiscountTypeRatio
	}

	// Validate: each model must not already exist in any item in this sheet
	sheetItems, err := model.GetPricingItemsBySheetId(sheetId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	existingModels := make(map[string]bool)
	for _, it := range sheetItems {
		for _, m := range it.Models {
			existingModels[m] = true
		}
	}
	for _, modelName := range req.Models {
		if existingModels[modelName] {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": fmt.Sprintf("model %q already exists in this pricing sheet", modelName)})
			return
		}
	}

	// Create a single item with models as JSON array
	item := &model.EnterprisePricingItem{
		PricingSheetId: sheetId,
		VendorType:    req.VendorType,
		Models:        req.Models,
		DiscountType:  req.DiscountType,
		DiscountValue: req.DiscountValue,
		Remark:        req.Remark,
	}
	if err := item.Create(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func UpdatePricingItem(c *gin.Context) {
	itemIdStr := c.Param("itemId")
	itemId, err := strconv.Atoi(itemIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid item id"})
		return
	}

	item, err := model.GetPricingItemById(itemId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "pricing item not found"})
		return
	}

	var req struct {
		VendorType    string   `json:"vendor_type"`
		Models        []string `json:"models"`
		DiscountType  string   `json:"discount_type"`
		DiscountValue float64  `json:"discount_value"`
		Remark        string   `json:"remark"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid json"})
		return
	}

	if req.VendorType != "" {
		item.VendorType = req.VendorType
	}
	if len(req.Models) > 0 {
		// Validate: new models must not conflict with other items in this sheet
		sheetItems, err := model.GetPricingItemsBySheetId(item.PricingSheetId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			return
		}
		existingModels := make(map[string]bool)
		for _, it := range sheetItems {
			if it.Id == item.Id {
				continue
			}
			for _, m := range it.Models {
				existingModels[m] = true
			}
		}
		for _, modelName := range req.Models {
			if existingModels[modelName] {
				c.JSON(http.StatusConflict, gin.H{"success": false, "message": fmt.Sprintf("model %q already exists in another item in this pricing sheet", modelName)})
				return
			}
		}
		item.Models = req.Models
	}
	if req.DiscountType != "" {
		item.DiscountType = req.DiscountType
	}
	if req.DiscountValue != 0 {
		item.DiscountValue = req.DiscountValue
	}
	if req.Remark != "" {
		item.Remark = req.Remark
	}

	if err := item.Update(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func DeletePricingItem(c *gin.Context) {
	itemIdStr := c.Param("itemId")
	itemId, err := strconv.Atoi(itemIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid item id"})
		return
	}

	item, err := model.GetPricingItemById(itemId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "pricing item not found"})
		return
	}

	if err := model.DeletePricingItem(itemId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------------------------------------------------------------------------
// Billing integration helpers
// ---------------------------------------------------------------------------

// GetUserActivePricingSheetForBilling returns the active pricing sheet ID for a user.
func GetUserActivePricingSheetForBilling(userId int) (int, bool) {
	sheet, err := service.GetUserActivePricingSheet(userId)
	if err != nil || sheet == nil {
		return 0, false
	}
	return sheet.Id, true
}

// GetModelDiscountForBilling returns the discount ratio for a model in a pricing sheet.
func GetModelDiscountForBilling(sheetId int, modelName string) (float64, bool) {
	return service.GetModelDiscount(sheetId, modelName)
}

// ---------------------------------------------------------------------------
// Pricing Sheet Channel Binding
// ---------------------------------------------------------------------------

// ListPricingSheetChannels returns all channel IDs bound to a pricing sheet.
func ListPricingSheetChannels(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}
	ids, err := model.GetChannelIdsBySheetId(sheetId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": ids})
}

// BindPricingSheetChannels replaces all channel bindings for a pricing sheet.
func BindPricingSheetChannels(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}
	var req struct {
		ChannelIds []int `json:"channel_ids"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid json"})
		return
	}
	if err := model.BindChannelsToSheet(sheetId, req.ChannelIds); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UnbindPricingSheetChannel removes a single channel binding from a pricing sheet.
func UnbindPricingSheetChannel(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}
	channelIdStr := c.Param("channelId")
	channelId, err := strconv.Atoi(channelIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid channel id"})
		return
	}
	binding := &model.EnterprisePricingSheetChannel{
		PricingSheetId: sheetId,
		ChannelId:      channelId,
	}
	if err := binding.Delete(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
