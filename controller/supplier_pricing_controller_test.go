package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Test DB setup
// ---------------------------------------------------------------------------

func setupSupplierPricingControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Supplier{},
		&model.SupplierPricingSheet{},
		&model.SupplierPricingItem{},
	))

	t.Cleanup(func() {})

	return db
}

func adminAuthContextSupplier() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", 1) // admin user
	c.Set("role", common.RoleAdminUser)
	return c, w
}

var supplierStartTime = time.Now().Unix() - 86400
var supplierEndTime = time.Now().Unix() + 86400

// ---------------------------------------------------------------------------
// Supplier CRUD
// ---------------------------------------------------------------------------

func TestCreateSupplier_ValidInput(t *testing.T) {
	_ = setupSupplierPricingControllerTestDB(t)

	body := `{"name":"OpenAI","remark":"OpenAI 官方渠道"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/supplier", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	CreateSupplier(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
}

func TestCreateSupplier_MissingName(t *testing.T) {
	_ = setupSupplierPricingControllerTestDB(t)

	body := `{"remark":"备注"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/supplier", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	CreateSupplier(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp["success"].(bool))
}

func TestGetSupplier(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	// Create a supplier first
	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/supplier/1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	GetSupplier(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
}

func TestGetSupplier_NotFound(t *testing.T) {
	_ = setupSupplierPricingControllerTestDB(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/supplier/999", nil)
	c.Params = []gin.Param{{Key: "id", Value: "999"}}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	GetSupplier(c)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp["success"].(bool))
}

func TestListSupplier(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	// Create suppliers
	for i := 1; i <= 3; i++ {
		s := &model.Supplier{Name: "Supplier" + string(rune('0'+i)), Status: model.SupplierStatusEnabled}
		s.CreatedAt = time.Now().Unix()
		s.UpdatedAt = time.Now().Unix()
		require.NoError(t, db.Create(s).Error)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/supplier", nil)
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	ListSupplier(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
}

func TestUpdateSupplier(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	// Create a supplier first
	s := &model.Supplier{Name: "OriginalName", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	body := `{"name":"UpdatedName","status":0,"remark":"updated"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/supplier/1", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	UpdateSupplier(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
}

func TestDeleteSupplier(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	// Create a supplier first
	s := &model.Supplier{Name: "ToDelete", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/supplier/1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	DeleteSupplier(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
}

// ---------------------------------------------------------------------------
// SupplierPricingSheet CRUD
// ---------------------------------------------------------------------------

func TestCreateSupplierPricingSheet_ValidInput(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	// Create a supplier first
	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	body := `{"name":"2024报价单","status":1,"start_time":` + string(rune(supplierStartTime)) + `,"end_time":` + string(rune(supplierEndTime)) + `}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/supplier/1/pricing-sheet", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	CreateSupplierPricingSheet(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetSupplierPricingSheet(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	// Create supplier and sheet
	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	sheet := &model.SupplierPricingSheet{
		SupplierId: s.Id,
		Name:       "TestSheet",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  supplierStartTime,
		EndTime:    supplierEndTime,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(sheet).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/supplier/1/pricing-sheet/1", nil)
	c.Params = []gin.Param{
		{Key: "id", Value: "1"},
		{Key: "sheetId", Value: "1"},
	}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	GetSupplierPricingSheet(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListSupplierPricingSheet(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	// Create supplier and sheets
	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	for i := 1; i <= 3; i++ {
		sheet := &model.SupplierPricingSheet{
			SupplierId: s.Id,
			Name:       "Sheet" + string(rune('0'+i)),
			Status:     model.SupplierPricingSheetStatusActive,
			StartTime:  supplierStartTime,
			EndTime:    supplierEndTime,
		}
		sheet.CreatedAt = time.Now().Unix()
		sheet.UpdatedAt = time.Now().Unix()
		require.NoError(t, db.Create(sheet).Error)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/supplier/1/pricing-sheet", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	ListSupplierPricingSheet(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteSupplierPricingSheet(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	sheet := &model.SupplierPricingSheet{
		SupplierId: s.Id,
		Name:       "ToDelete",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  supplierStartTime,
		EndTime:    supplierEndTime,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(sheet).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/supplier/1/pricing-sheet/1", nil)
	c.Params = []gin.Param{
		{Key: "id", Value: "1"},
		{Key: "sheetId", Value: "1"},
	}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	DeleteSupplierPricingSheet(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// SupplierPricingItem CRUD
// ---------------------------------------------------------------------------

func TestAddSupplierPricingItem_ValidInput(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	// Create supplier and sheet
	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	sheet := &model.SupplierPricingSheet{
		SupplierId: s.Id,
		Name:       "TestSheet",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  supplierStartTime,
		EndTime:    supplierEndTime,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(sheet).Error)

	body := `{"models":["gpt-4o"],"discount_type":"ratio","discount_value":0.75,"remark":"test"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/supplier/1/pricing-sheet/1/items", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{
		{Key: "id", Value: "1"},
		{Key: "sheetId", Value: "1"},
	}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	AddSupplierPricingItem(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
}

func TestAddSupplierPricingItem_DuplicateModel(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	sheet := &model.SupplierPricingSheet{
		SupplierId: s.Id,
		Name:       "TestSheet",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  supplierStartTime,
		EndTime:    supplierEndTime,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(sheet).Error)

	item := &model.SupplierPricingItem{
		PricingSheetId: sheet.Id,
		Models:         []string{"gpt-4o"},
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.7,
	}
	require.NoError(t, db.Create(item).Error)

	body := `{"models":["gpt-4o"],"discount_type":"ratio","discount_value":0.6}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/supplier/1/pricing-sheet/1/items", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{
		{Key: "id", Value: "1"},
		{Key: "sheetId", Value: "1"},
	}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	AddSupplierPricingItem(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp["success"].(bool))
}

func TestListSupplierPricingItems(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	sheet := &model.SupplierPricingSheet{
		SupplierId: s.Id,
		Name:       "TestSheet",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  supplierStartTime,
		EndTime:    supplierEndTime,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(sheet).Error)

	models := []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"}
	item := &model.SupplierPricingItem{
		PricingSheetId: sheet.Id,
		Models:         models,
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.75,
	}
	require.NoError(t, db.Create(item).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/supplier/1/pricing-sheet/1/items", nil)
	c.Params = []gin.Param{
		{Key: "id", Value: "1"},
		{Key: "sheetId", Value: "1"},
	}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	ListSupplierPricingItems(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
}

func TestUpdateSupplierPricingItem(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	sheet := &model.SupplierPricingSheet{
		SupplierId: s.Id,
		Name:       "TestSheet",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  supplierStartTime,
		EndTime:    supplierEndTime,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(sheet).Error)

	item := &model.SupplierPricingItem{
		PricingSheetId: sheet.Id,
		Models:        []string{"gpt-4o"},
		DiscountType:  model.DiscountTypeRatio,
		DiscountValue: 0.7,
	}
	require.NoError(t, db.Create(item).Error)

	body := `{"id":1,"models":["gpt-4o"],"discount_type":"ratio","discount_value":0.5,"remark":"updated"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/supplier/1/pricing-sheet/1/items/1", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{
		{Key: "id", Value: "1"},
		{Key: "sheetId", Value: "1"},
		{Key: "itemId", Value: "1"},
	}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	UpdateSupplierPricingItem(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp["success"].(bool))
}

func TestDeleteSupplierPricingItem(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	sheet := &model.SupplierPricingSheet{
		SupplierId: s.Id,
		Name:       "TestSheet",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  supplierStartTime,
		EndTime:    supplierEndTime,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(sheet).Error)

	item := &model.SupplierPricingItem{
		PricingSheetId: sheet.Id,
		Models:        []string{"gpt-4o"},
		DiscountType:  model.DiscountTypeRatio,
		DiscountValue: 0.7,
	}
	require.NoError(t, db.Create(item).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/supplier/1/pricing-sheet/1/items/1", nil)
	c.Params = []gin.Param{
		{Key: "id", Value: "1"},
		{Key: "sheetId", Value: "1"},
		{Key: "itemId", Value: "1"},
	}
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	DeleteSupplierPricingItem(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// ListAllSupplierPricingSheets
// ---------------------------------------------------------------------------

func TestListAllSupplierPricingSheets(t *testing.T) {
	db := setupSupplierPricingControllerTestDB(t)

	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(s).Error)

	sheet := &model.SupplierPricingSheet{
		SupplierId: s.Id,
		Name:       "TestSheet",
		Status:     model.SupplierPricingSheetStatusActive,
		StartTime:  supplierStartTime,
		EndTime:    supplierEndTime,
	}
	sheet.CreatedAt = time.Now().Unix()
	sheet.UpdatedAt = time.Now().Unix()
	require.NoError(t, db.Create(sheet).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/supplier-pricing-sheets", nil)
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	ListAllSupplierPricingSheets(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Service integration: IsSupplierEnabled wrapper
// ---------------------------------------------------------------------------

func TestServiceIsSupplierEnabled_Enabled(t *testing.T) {
	_ = setupSupplierPricingControllerTestDB(t)

	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusEnabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, model.DB.Create(s).Error)

	enabled := service.IsSupplierEnabled(s.Id)
	assert.True(t, enabled)
}

func TestServiceIsSupplierEnabled_Disabled(t *testing.T) {
	_ = setupSupplierPricingControllerTestDB(t)

	s := &model.Supplier{Name: "TestSupplier", Status: model.SupplierStatusDisabled}
	s.CreatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	require.NoError(t, model.DB.Create(s).Error)

	enabled := service.IsSupplierEnabled(s.Id)
	assert.False(t, enabled)
}
