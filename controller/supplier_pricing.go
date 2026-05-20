package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// Supplier discount type constants (mirrors model.DiscountType* for convenience)
const (
	SupplierDiscountTypeRatio      = "ratio"
	SupplierDiscountTypeFixedPrice = "fixed_price"
	SupplierDiscountTypePerCall    = "per_call"
)

// ---------------------------------------------------------------------------
// Supplier CRUD
// ---------------------------------------------------------------------------

func CreateSupplier(c *gin.Context) {
	var req struct {
		Name   string `json:"name"`
		Status int    `json:"status"`
		Remark string `json:"remark"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid json"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name is required"})
		return
	}
	if req.Status == 0 {
		req.Status = model.SupplierStatusEnabled
	}

	now := time.Now().Unix()
	s := &model.Supplier{
		Name:      req.Name,
		Status:    req.Status,
		Remark:    req.Remark,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Create(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": s})
}

func ListSupplier(c *gin.Context) {
	p, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if p < 1 {
		p = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	suppliers, total, err := model.GetSuppliers(p, pageSize)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"items":     suppliers,
		"total":     total,
		"page":      p,
		"page_size": pageSize,
	}})
}

func GetSupplier(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}

	s, err := model.GetSupplierById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if s == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "supplier not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": s})
}

func UpdateSupplier(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}

	var req struct {
		Name   string `json:"name"`
		Status int    `json:"status"`
		Remark string `json:"remark"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid json"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name is required"})
		return
	}

	s, err := model.GetSupplierById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if s == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "supplier not found"})
		return
	}

	s.Name = req.Name
	s.Status = req.Status
	s.Remark = req.Remark
	s.UpdatedAt = time.Now().Unix()
	if err := s.Update(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": s})
}

func DeleteSupplier(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}

	if err := model.DeleteSupplier(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------------------------------------------------------------------------
// SupplierPricingSheet CRUD
// ---------------------------------------------------------------------------

func CreateSupplierPricingSheet(c *gin.Context) {
	idStr := c.Param("id")
	supplierId, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid supplier id"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		Status    int    `json:"status"`
		ChannelId int    `json:"channel_id"`
		StartTime int64  `json:"start_time"`
		EndTime   int64  `json:"end_time"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid json"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name is required"})
		return
	}
	if req.Status == 0 {
		req.Status = model.SupplierPricingSheetStatusActive
	}

	now := time.Now().Unix()
	sheet := &model.SupplierPricingSheet{
		SupplierId: supplierId,
		ChannelId:  req.ChannelId,
		Name:       req.Name,
		Status:     req.Status,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if sheet.EndTime == 0 {
		sheet.EndTime = 1<<62 - 1 // permanent (max int64)
	}
	if err := sheet.Create(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sheet})
}

func ListSupplierPricingSheet(c *gin.Context) {
	idStr := c.Param("id")
	supplierId, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid supplier id"})
		return
	}

	sheets, err := model.GetPricingSheetsBySupplierId(supplierId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"items": sheets,
		"total": len(sheets),
	}})
}

func GetSupplierPricingSheet(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}

	sheet, err := model.GetSupplierPricingSheetById(sheetId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if sheet == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "pricing sheet not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sheet})
}

func UpdateSupplierPricingSheet(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		Status    int    `json:"status"`
		ChannelId int    `json:"channel_id"`
		StartTime int64  `json:"start_time"`
		EndTime   int64  `json:"end_time"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid json"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "name is required"})
		return
	}

	sheet, err := model.GetSupplierPricingSheetById(sheetId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if sheet == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "pricing sheet not found"})
		return
	}

	sheet.Name = req.Name
	sheet.Status = req.Status
	sheet.StartTime = req.StartTime
	sheet.EndTime = req.EndTime
	sheet.UpdatedAt = time.Now().Unix()
	if sheet.EndTime == 0 {
		sheet.EndTime = 1<<62 - 1 // permanent (max int64)
	}
	if err := sheet.Update(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sheet})
}

func DeleteSupplierPricingSheet(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}

	if err := model.DeleteSupplierPricingSheet(sheetId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------------------------------------------------------------------------
// SupplierPricingSheet Channel Binding (multi-channel)
// ---------------------------------------------------------------------------

func ListSupplierPricingSheetChannels(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}
	ids, err := model.GetSupplierPricingSheetChannelIds(sheetId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": ids})
}

func BindSupplierPricingSheetChannels(c *gin.Context) {
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
	if err := model.BindChannelsToSupplierPricingSheet(sheetId, req.ChannelIds); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func UnbindSupplierPricingSheetChannel(c *gin.Context) {
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
	binding := &model.SupplierPricingSheetChannel{
		PricingSheetId: sheetId,
		ChannelId:     channelId,
	}
	if err := binding.Delete(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------------------------------------------------------------------------
// SupplierPricingItem CRUD
// ---------------------------------------------------------------------------

func AddSupplierPricingItem(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid sheet id"})
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
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid json"})
		return
	}
	if req.VendorType == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "vendor_type is required"})
		return
	}
	if len(req.Models) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "models is required"})
		return
	}
	if req.DiscountType == "" {
		req.DiscountType = SupplierDiscountTypeRatio
	}

	sheet, err := model.GetSupplierPricingSheetById(sheetId)
	if err != nil || sheet == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "pricing sheet not found"})
		return
	}

	existingItems, err := model.GetSupplierPricingItemsBySheetId(sheetId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	existingModels := make(map[string]bool)
	for _, it := range existingItems {
		for _, m := range it.Models {
			existingModels[m] = true
		}
	}
	for _, modelName := range req.Models {
		if existingModels[modelName] {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": "model " + modelName + " already exists in this sheet"})
			return
		}
	}

	item := &model.SupplierPricingItem{
		PricingSheetId: sheetId,
		VendorType:     req.VendorType,
		Models:        req.Models,
		DiscountType:  req.DiscountType,
		DiscountValue: req.DiscountValue,
		Remark:        req.Remark,
	}
	if err := item.Create(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func ListSupplierPricingItems(c *gin.Context) {
	sheetIdStr := c.Param("sheetId")
	sheetId, err := strconv.Atoi(sheetIdStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid sheet id"})
		return
	}

	items, err := service.GetSupplierPricingSheetItems(sheetId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func UpdateSupplierPricingItem(c *gin.Context) {
	itemIdStr := c.Param("itemId")
	itemId, err := strconv.Atoi(itemIdStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid item id"})
		return
	}

	var req struct {
		Id            int      `json:"id"`
		VendorType    string   `json:"vendor_type"`
		Models        []string `json:"models"`
		DiscountType  string   `json:"discount_type"`
		DiscountValue float64  `json:"discount_value"`
		Remark        string   `json:"remark"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid json"})
		return
	}
	if len(req.Models) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "models is required"})
		return
	}

	item, err := model.GetSupplierPricingItemById(itemId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "pricing item not found"})
		return
	}

	sheet, err := model.GetSupplierPricingSheetById(item.PricingSheetId)
	if err != nil || sheet == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "pricing sheet not found"})
		return
	}

	existingItems, err := model.GetSupplierPricingItemsBySheetId(item.PricingSheetId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	existingModels := make(map[string]bool)
	for _, it := range existingItems {
		if it.Id == itemId {
			continue
		}
		for _, m := range it.Models {
			existingModels[m] = true
		}
	}
	for _, modelName := range req.Models {
		if existingModels[modelName] {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": "model " + modelName + " already exists in this sheet"})
			return
		}
	}

	if req.VendorType != "" {
		item.VendorType = req.VendorType
	}
	item.Models = req.Models
	item.DiscountType = req.DiscountType
	item.DiscountValue = req.DiscountValue
	item.Remark = req.Remark
	if err := item.Update(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func DeleteSupplierPricingItem(c *gin.Context) {
	itemIdStr := c.Param("itemId")
	itemId, err := strconv.Atoi(itemIdStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid item id"})
		return
	}

	if err := model.DeleteSupplierPricingItem(itemId); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ---------------------------------------------------------------------------
// ListAllSupplierPricingSheets
// ---------------------------------------------------------------------------

func ListAllSupplierPricingSheets(c *gin.Context) {
	p, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	supplierIdStr := c.Query("supplier_id")
	nameStr := c.Query("name")
	if p < 1 {
		p = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	sheets, total, err := model.GetAllSupplierPricingSheets(p, pageSize, supplierIdStr, nameStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"items":     sheets,
		"total":     total,
		"page":      p,
		"page_size": pageSize,
	}})
}

// ---------------------------------------------------------------------------
// Billing chain helper stubs
// ---------------------------------------------------------------------------

func GetChannelActivePricingSheetForBilling(userId int) (int, bool) {
	return service.GetChannelActivePricingSheetForBilling(userId)
}

func GetModelCostForBilling(sheetId int, modelName string) (float64, bool) {
	return service.GetModelCostForBilling(sheetId, modelName)
}
