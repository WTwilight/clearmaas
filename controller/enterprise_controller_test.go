package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
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
// Test DB setup (shared across all enterprise controller tests)
// ---------------------------------------------------------------------------

func setupEnterpriseControllerTestDB(t *testing.T) *gorm.DB {
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
		&model.Enterprise{},
		&model.EnterprisePricingSheet{},
		&model.EnterprisePricingItem{},
		&model.EnterpriseUserBinding{},
		&model.User{},
	))

	t.Cleanup(func() {})

	return db
}

func adminAuthContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", 1) // admin user
	c.Set("role", common.RoleAdminUser)
	return c, w
}

var startTime = time.Now().Unix() - 86400
var endTime = time.Now().Unix() + 86400

// ---------------------------------------------------------------------------
// Enterprise CRUD
// ---------------------------------------------------------------------------

func TestCreateEnterprise_ValidInput(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	_ = db

	body := `{"name":"新公司","remark":"测试备注"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/enterprise", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	CreateEnterprise(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "新公司", data["name"])
}

func TestCreateEnterprise_MissingName(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	_ = db

	body := `{"remark":"无名称"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/enterprise", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	CreateEnterprise(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateEnterprise_InvalidJSON(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	_ = db

	body := `{invalid json}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/enterprise", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	CreateEnterprise(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListEnterprise(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)

	now := time.Now().Unix()
	for i := 0; i < 3; i++ {
		e := &model.Enterprise{Name: "公司", Status: model.EnterpriseStatusEnabled}
		e.CreatedAt = now
		e.UpdatedAt = now
		require.NoError(t, db.Create(e).Error)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/enterprise", nil)
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)

	ListEnterprise(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].([]interface{})
	assert.Len(t, data, 3)
}

func TestUpdateEnterprise(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)

	now := time.Now().Unix()
	e := &model.Enterprise{Name: "旧名称", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	body := `{"name":"新名称","status":0}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/enterprise/:id", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	UpdateEnterprise(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["success"].(bool))
}

func TestUpdateEnterprise_NotFound(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	_ = db

	body := `{"name":"新名称"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/enterprise/99999", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{{Key: "id", Value: "99999"}}

	UpdateEnterprise(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteEnterprise(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)

	now := time.Now().Unix()
	e := &model.Enterprise{Name: "待删除", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/enterprise/:id", nil)
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	DeleteEnterprise(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify deleted
	found, err := model.GetEnterpriseById(int(e.Id))
	require.NoError(t, err)
	require.Nil(t, found)
}

// ---------------------------------------------------------------------------
// Enterprise-Pricing-Sheet CRUD
// ---------------------------------------------------------------------------

func TestCreatePricingSheet(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)

	now := time.Now().Unix()
	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	bodyMap := map[string]interface{}{
		"name":       "2024报价单",
		"status":     1,
		"start_time": startTime,
		"end_time":   endTime,
	}
	bodyBytes, _ := json.Marshal(bodyMap)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/enterprise/:id/pricing-sheet", bytes.NewBufferString(string(bodyBytes)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	CreatePricingSheet(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "2024报价单", data["name"])
}

func TestListPricingSheet(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	now := time.Now().Unix()

	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	for i := 0; i < 2; i++ {
		sheet := &model.EnterprisePricingSheet{
			EnterpriseId: e.Id,
			Name:        "报价单",
			Status:      model.PricingSheetStatusActive,
			StartTime:   now - 86400,
			EndTime:     now + 86400,
		}
		sheet.CreatedAt = now
		sheet.UpdatedAt = now
		require.NoError(t, db.Create(sheet).Error)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/enterprise/:id/pricing-sheet", nil)
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	ListPricingSheet(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
	items := data["items"].([]interface{})
	assert.Len(t, items, 2)
}

func TestUpdatePricingSheet(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	now := time.Now().Unix()

	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:        "旧报价单",
		Status:      model.PricingSheetStatusActive,
		StartTime:   now - 86400,
		EndTime:     now + 86400,
	}
	sheet.CreatedAt = now
	sheet.UpdatedAt = now
	require.NoError(t, db.Create(sheet).Error)

	body := `{"name":"新报价单"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/enterprise/:id/pricing-sheet/:sheetId", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "sheetId", Value: "1"},
	}

	UpdatePricingSheet(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeletePricingSheet(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	now := time.Now().Unix()

	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:        "待删除",
		Status:      model.PricingSheetStatusActive,
		StartTime:   now - 86400,
		EndTime:     now + 86400,
	}
	sheet.CreatedAt = now
	sheet.UpdatedAt = now
	require.NoError(t, db.Create(sheet).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/enterprise/:id/pricing-sheet/:sheetId", nil)
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "sheetId", Value: "1"},
	}

	DeletePricingSheet(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Enterprise-User Binding
// ---------------------------------------------------------------------------

func TestBindUsers(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	now := time.Now().Unix()

	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	u := &model.User{Id: 1, Username: "alice", Status: common.UserStatusEnabled, Quota: 1000}
	u.CreatedAt = now
	require.NoError(t, db.Create(u).Error)

	body := `{"user_ids":[1]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/enterprise/:id/users", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	BindUsers(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify binding
	id, found := service.IsUserInEnterprise(1)
	assert.True(t, found)
	assert.Equal(t, e.Id, id)
}

func TestBindUsers_DuplicateBinding(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	now := time.Now().Unix()

	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	u := &model.User{Id: 1, Username: "alice", Status: common.UserStatusEnabled, Quota: 1000}
	u.CreatedAt = now
	require.NoError(t, db.Create(u).Error)

	binding := &model.EnterpriseUserBinding{EnterpriseId: e.Id, UserId: u.Id}
	binding.CreatedAt = now
	require.NoError(t, db.Create(binding).Error)

	// Try to bind again
	body := `{"user_ids":[1]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/enterprise/:id/users", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	BindUsers(c)

	// Should handle duplicate gracefully (409 Conflict or OK with skip)
	assert.True(t, w.Code == http.StatusConflict || w.Code == http.StatusOK)
}

func TestUnbindUser(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	now := time.Now().Unix()

	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	u := &model.User{Id: 1, Username: "alice", Status: common.UserStatusEnabled, Quota: 1000}
	u.CreatedAt = now
	require.NoError(t, db.Create(u).Error)

	binding := &model.EnterpriseUserBinding{EnterpriseId: e.Id, UserId: u.Id}
	binding.CreatedAt = now
	require.NoError(t, db.Create(binding).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/enterprise/:id/users/:userId", nil)
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "userId", Value: "1"},
	}

	UnbindUser(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify unbinding
	id, found := service.IsUserInEnterprise(1)
	assert.False(t, found)
	assert.Zero(t, id)
}

func TestListEnterpriseUsers(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	now := time.Now().Unix()

	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	for i := 1; i <= 3; i++ {
		u := &model.User{Id: i, Username: fmt.Sprintf("user%d", i), Status: common.UserStatusEnabled, Quota: 1000, AffCode: fmt.Sprintf("aff%d", i)}
		u.CreatedAt = now
		require.NoError(t, db.Create(u).Error)

		binding := &model.EnterpriseUserBinding{EnterpriseId: e.Id, UserId: i}
		binding.CreatedAt = now
		require.NoError(t, db.Create(binding).Error)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/enterprise/:id/users", nil)
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	ListEnterpriseUsers(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].([]interface{})
	assert.Len(t, data, 3)
}

// ---------------------------------------------------------------------------
// Pricing Item CRUD
// ---------------------------------------------------------------------------

func TestAddPricingItem(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	now := time.Now().Unix()

	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:        "报价单",
		Status:      model.PricingSheetStatusActive,
		StartTime:   now - 86400,
		EndTime:     now + 86400,
	}
	sheet.CreatedAt = now
	sheet.UpdatedAt = now
	require.NoError(t, db.Create(sheet).Error)

	body := `{"model":"gpt-4o","discount_type":"ratio","discount_value":0.7,"remark":"7折"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/pricing-sheet/:sheetId/item", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{{Key: "sheetId", Value: "1"}}

	AddPricingItem(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "gpt-4o", data["model"])
	assert.Equal(t, 0.7, data["discount_value"])
}

func TestAddPricingItem_DuplicateModel(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	now := time.Now().Unix()

	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:        "报价单",
		Status:      model.PricingSheetStatusActive,
		StartTime:   now - 86400,
		EndTime:     now + 86400,
	}
	sheet.CreatedAt = now
	sheet.UpdatedAt = now
	require.NoError(t, db.Create(sheet).Error)

	item := &model.EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Models:         []string{"gpt-4o"},
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.7,
	}
	require.NoError(t, db.Create(item).Error)

	// Try to add same model again
	body := `{"model":"gpt-4o","discount_type":"ratio","discount_value":0.5}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/pricing-sheet/:sheetId/item", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{{Key: "sheetId", Value: "1"}}

	AddPricingItem(c)

	// Should reject duplicate
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestUpdatePricingItem(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	now := time.Now().Unix()

	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:        "报价单",
		Status:      model.PricingSheetStatusActive,
		StartTime:   now - 86400,
		EndTime:     now + 86400,
	}
	sheet.CreatedAt = now
	sheet.UpdatedAt = now
	require.NoError(t, db.Create(sheet).Error)

	item := &model.EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Models:         []string{"gpt-4o"},
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.7,
	}
	require.NoError(t, db.Create(item).Error)

	body := `{"discount_value":0.5}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/pricing-sheet/:sheetId/item/:itemId", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{
		{Key: "sheetId", Value: "1"},
		{Key: "itemId", Value: "1"},
	}

	UpdatePricingItem(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeletePricingItem(t *testing.T) {
	db := setupEnterpriseControllerTestDB(t)
	now := time.Now().Unix()

	e := &model.Enterprise{Name: "Corp", Status: model.EnterpriseStatusEnabled}
	e.CreatedAt = now
	e.UpdatedAt = now
	require.NoError(t, db.Create(e).Error)

	sheet := &model.EnterprisePricingSheet{
		EnterpriseId: e.Id,
		Name:        "报价单",
		Status:      model.PricingSheetStatusActive,
		StartTime:   now - 86400,
		EndTime:     now + 86400,
	}
	sheet.CreatedAt = now
	sheet.UpdatedAt = now
	require.NoError(t, db.Create(sheet).Error)

	item := &model.EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		Models:         []string{"gpt-4o"},
		DiscountType:   model.DiscountTypeRatio,
		DiscountValue:  0.7,
	}
	require.NoError(t, db.Create(item).Error)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/pricing-sheet/:sheetId/item/:itemId", nil)
	c.Set("id", 1)
	c.Set("role", common.RoleAdminUser)
	c.Params = gin.Params{
		{Key: "sheetId", Value: "1"},
		{Key: "itemId", Value: "1"},
	}

	DeletePricingItem(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify deleted
	found, err := model.GetPricingItemById(int(item.Id))
	require.NoError(t, err)
	require.Nil(t, found)
}
