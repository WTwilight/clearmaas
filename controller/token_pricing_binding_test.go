package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ============================================================================
// Controller: BindTokenPricingModels / UnbindTokenPricingModels
// Route: /api/token/:id/bindings
// ============================================================================

func setupTokenBindingControllerDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := "file::memory:?cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db

	err = db.AutoMigrate(
		&model.Token{},
		&model.TokenPricingModelBinding{},
		&model.Enterprise{},
		&model.EnterprisePricingSheet{},
		&model.EnterprisePricingItem{},
		&model.EnterpriseUserBinding{},
		&model.User{},
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})

	return db
}

func truncateControllerBindings(t *testing.T) {
	t.Helper()
	cleanup := func() {
		model.DB.Exec("DELETE FROM token_pricing_model_bindings")
		model.DB.Exec("DELETE FROM tokens")
		model.DB.Exec("DELETE FROM enterprise_pricing_items")
		model.DB.Exec("DELETE FROM enterprise_pricing_sheets")
		model.DB.Exec("DELETE FROM enterprise_user_bindings")
		model.DB.Exec("DELETE FROM enterprises")
		model.DB.Exec("DELETE FROM users")
	}
	cleanup()
	t.Cleanup(cleanup)
}

// ---------------------------------------------------------------------------
// BindTokenPricingModels
// ---------------------------------------------------------------------------

func TestBindTokenPricingModels_Success(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "bind-user")
	e := seedControllerEnterprise(t, "BindCorp")
	seedControllerUserBinding(t, e.Id, user.Id)
	sheet := seedControllerPricingSheet(t, e.Id, "BindSheet")
	seedControllerPricingItem(t, sheet.Id, []string{"gpt-4o", "gpt-4o-mini"}, model.DiscountTypeRatio, 0.7)
	token := seedControllerToken(t, user.Id, "bind-token")

	reqBody := BindTokenPricingModelsRequest{
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-4o,gpt-4o-mini",
		Bindings: []struct {
			SheetId int    `json:"sheet_id"`
			Model   string `json:"model"`
		}{
			{SheetId: sheet.Id, Model: "gpt-4o"},
			{SheetId: sheet.Id, Model: "gpt-4o-mini"},
		},
	}
	ctx, recorder := newControllerContext(t, http.MethodPost, "/api/token/"+tokenKey(token)+"/bindings", reqBody, user.Id)
	ctx.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	BindTokenPricingModels(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var resp apiResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	assert.True(t, resp.Success, "expected success, got: %s", resp.Message)

	bindings, err := model.GetTokenPricingModelBindings(token.Id)
	require.NoError(t, err)
	require.Len(t, bindings, 2)
}

func TestBindTokenPricingModels_NonExistentToken(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "no-token-user")
	e := seedControllerEnterprise(t, "NoTokenCorp")
	seedControllerUserBinding(t, e.Id, user.Id)
	sheet := seedControllerPricingSheet(t, e.Id, "NoTokenSheet")
	seedControllerPricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.7)

	reqBody := BindTokenPricingModelsRequest{
		ModelLimitsEnabled: true,
		Bindings: []struct {
			SheetId int    `json:"sheet_id"`
			Model   string `json:"model"`
		}{
			{SheetId: sheet.Id, Model: "gpt-4o"},
		},
	}
	ctx, recorder := newControllerContext(t, http.MethodPost, "/api/token/99999/bindings", reqBody, user.Id)
	ctx.Params = gin.Params{{Key: "id", Value: "99999"}}
	BindTokenPricingModels(ctx)

	// API returns a response (not a panic); check it's not success
	var resp apiResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	assert.False(t, resp.Success, "non-existent token should not succeed")
}

func TestBindTokenPricingModels_EmptyModelList(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "empty-models-user")
	e := seedControllerEnterprise(t, "EmptyCorp")
	seedControllerUserBinding(t, e.Id, user.Id)
	seedControllerPricingSheet(t, e.Id, "EmptySheet")
	token := seedControllerToken(t, user.Id, "empty-models-token")

	// Empty bindings array — controller should accept it (just no bindings created)
	reqBody := BindTokenPricingModelsRequest{
		ModelLimitsEnabled: true,
		Bindings: []struct {
			SheetId int    `json:"sheet_id"`
			Model   string `json:"model"`
		}{},
	}
	ctx, recorder := newControllerContext(t, http.MethodPost, "/api/token/"+tokenKey(token)+"/bindings", reqBody, user.Id)
	ctx.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	BindTokenPricingModels(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestBindTokenPricingModels_ModelNotInPricingSheet(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "not-in-sheet-user")
	e := seedControllerEnterprise(t, "NotInSheetCorp")
	seedControllerUserBinding(t, e.Id, user.Id)
	sheet := seedControllerPricingSheet(t, e.Id, "NotInSheetSheet")
	seedControllerPricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.7)
	token := seedControllerToken(t, user.Id, "not-in-sheet-token")

	// Try to bind a model not in the pricing sheet
	reqBody := BindTokenPricingModelsRequest{
		ModelLimitsEnabled: true,
		Bindings: []struct {
			SheetId int    `json:"sheet_id"`
			Model   string `json:"model"`
		}{
			{SheetId: sheet.Id, Model: "claude-3-5-sonnet"},
		},
	}
	ctx, recorder := newControllerContext(t, http.MethodPost, "/api/token/"+tokenKey(token)+"/bindings", reqBody, user.Id)
	ctx.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	BindTokenPricingModels(ctx)

	var resp apiResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
}

func TestBindTokenPricingModels_SetsModelLimitsOnToken(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "limits-set-user")
	e := seedControllerEnterprise(t, "LimitsSetCorp")
	seedControllerUserBinding(t, e.Id, user.Id)
	sheet := seedControllerPricingSheet(t, e.Id, "LimitsSetSheet")
	seedControllerPricingItem(t, sheet.Id, []string{"gpt-4o", "gpt-4o-mini"}, model.DiscountTypeRatio, 0.7)
	token := seedControllerToken(t, user.Id, "limits-set-token")

	reqBody := BindTokenPricingModelsRequest{
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-4o,gpt-4o-mini",
		Bindings: []struct {
			SheetId int    `json:"sheet_id"`
			Model   string `json:"model"`
		}{
			{SheetId: sheet.Id, Model: "gpt-4o"},
			{SheetId: sheet.Id, Model: "gpt-4o-mini"},
		},
	}
	ctx, recorder := newControllerContext(t, http.MethodPost, "/api/token/"+tokenKey(token)+"/bindings", reqBody, user.Id)
	ctx.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	BindTokenPricingModels(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	// Verify token's ModelLimits fields were updated
	reloaded, err := model.GetTokenById(token.Id)
	require.NoError(t, err)
	assert.True(t, reloaded.ModelLimitsEnabled)
	assert.Equal(t, "gpt-4o,gpt-4o-mini", reloaded.ModelLimits)
}

func TestBindTokenPricingModels_IdempotentOverwrite(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "idempotent-user")
	e := seedControllerEnterprise(t, "IdempotentCorp")
	seedControllerUserBinding(t, e.Id, user.Id)
	sheet := seedControllerPricingSheet(t, e.Id, "IdempotentSheet")
	seedControllerPricingItem(t, sheet.Id, []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"}, model.DiscountTypeRatio, 0.7)
	token := seedControllerToken(t, user.Id, "idempotent-token")

	// First bind: gpt-4o, gpt-4o-mini
	reqBody1 := BindTokenPricingModelsRequest{
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-4o,gpt-4o-mini",
		Bindings: []struct {
			SheetId int    `json:"sheet_id"`
			Model   string `json:"model"`
		}{
			{SheetId: sheet.Id, Model: "gpt-4o"},
			{SheetId: sheet.Id, Model: "gpt-4o-mini"},
		},
	}
	ctx1, _ := newControllerContext(t, http.MethodPost, "/api/token/"+tokenKey(token)+"/bindings", reqBody1, user.Id)
	ctx1.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	BindTokenPricingModels(ctx1)

	// Second bind: replaces with claude-3-5-sonnet only
	reqBody2 := BindTokenPricingModelsRequest{
		ModelLimitsEnabled: true,
		ModelLimits:        "claude-3-5-sonnet",
		Bindings: []struct {
			SheetId int    `json:"sheet_id"`
			Model   string `json:"model"`
		}{
			{SheetId: sheet.Id, Model: "claude-3-5-sonnet"},
		},
	}
	ctx2, recorder2 := newControllerContext(t, http.MethodPost, "/api/token/"+tokenKey(token)+"/bindings", reqBody2, user.Id)
	ctx2.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	BindTokenPricingModels(ctx2)

	require.Equal(t, http.StatusOK, recorder2.Code)

	bindings, err := model.GetTokenPricingModelBindings(token.Id)
	require.NoError(t, err)
	require.Len(t, bindings, 1)
	assert.Equal(t, "claude-3-5-sonnet", bindings[0].Model)
}

// ---------------------------------------------------------------------------
// UnbindTokenPricingModels
// ---------------------------------------------------------------------------

func TestUnbindTokenPricingModels_Success(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "unbind-user")
	e := seedControllerEnterprise(t, "UnbindCorp")
	seedControllerUserBinding(t, e.Id, user.Id)
	sheet := seedControllerPricingSheet(t, e.Id, "UnbindSheet")
	seedControllerPricingItem(t, sheet.Id, []string{"gpt-4o", "gpt-4o-mini"}, model.DiscountTypeRatio, 0.7)
	token := seedControllerToken(t, user.Id, "unbind-token")

	// First bind
	bindReq := BindTokenPricingModelsRequest{
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-4o,gpt-4o-mini",
		Bindings: []struct {
			SheetId int    `json:"sheet_id"`
			Model   string `json:"model"`
		}{
			{SheetId: sheet.Id, Model: "gpt-4o"},
			{SheetId: sheet.Id, Model: "gpt-4o-mini"},
		},
	}
	bindCtx, _ := newControllerContext(t, http.MethodPost, "/api/token/"+tokenKey(token)+"/bindings", bindReq, user.Id)
	bindCtx.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	BindTokenPricingModels(bindCtx)

	// Now unbind all
	unbindCtx, recorder := newControllerContext(t, http.MethodDelete, "/api/token/"+tokenKey(token)+"/bindings", nil, user.Id)
	unbindCtx.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	UnbindTokenPricingModels(unbindCtx)

	require.Equal(t, http.StatusOK, recorder.Code)

	bindings, err := model.GetTokenPricingModelBindings(token.Id)
	require.NoError(t, err)
	require.Len(t, bindings, 0, "all bindings should be removed after unbind")

	// Token ModelLimits should be disabled
	reloaded, err := model.GetTokenById(token.Id)
	require.NoError(t, err)
	assert.False(t, reloaded.ModelLimitsEnabled)
	assert.Equal(t, "", reloaded.ModelLimits)
}

func TestUnbindTokenPricingModels_AllModels(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "all-unbind-user")
	e := seedControllerEnterprise(t, "AllUnbindCorp")
	seedControllerUserBinding(t, e.Id, user.Id)
	sheet := seedControllerPricingSheet(t, e.Id, "AllUnbindSheet")
	seedControllerPricingItem(t, sheet.Id, []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"}, model.DiscountTypeRatio, 0.7)
	token := seedControllerToken(t, user.Id, "all-unbind-token")

	// Bind all models
	bindReq := BindTokenPricingModelsRequest{
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-4o,gpt-4o-mini,claude-3-5-sonnet",
		Bindings: []struct {
			SheetId int    `json:"sheet_id"`
			Model   string `json:"model"`
		}{
			{SheetId: sheet.Id, Model: "gpt-4o"},
			{SheetId: sheet.Id, Model: "gpt-4o-mini"},
			{SheetId: sheet.Id, Model: "claude-3-5-sonnet"},
		},
	}
	bindCtx, _ := newControllerContext(t, http.MethodPost, "/api/token/"+tokenKey(token)+"/bindings", bindReq, user.Id)
	bindCtx.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	BindTokenPricingModels(bindCtx)

	// Unbind
	unbindCtx, recorder := newControllerContext(t, http.MethodDelete, "/api/token/"+tokenKey(token)+"/bindings", nil, user.Id)
	unbindCtx.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	UnbindTokenPricingModels(unbindCtx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	bindings, err := model.GetTokenPricingModelBindings(token.Id)
	require.NoError(t, err)
	require.Len(t, bindings, 0)
}

// ---------------------------------------------------------------------------
// GetTokenPricingModels
// ---------------------------------------------------------------------------

func TestGetTokenPricingModels_Success(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "get-models-user")
	e := seedControllerEnterprise(t, "GetModelsCorp")
	seedControllerUserBinding(t, e.Id, user.Id)
	sheet := seedControllerPricingSheet(t, e.Id, "GetModelsSheet")
	seedControllerPricingItem(t, sheet.Id, []string{"gpt-4o", "gpt-4o-mini"}, model.DiscountTypeRatio, 0.7)
	token := seedControllerToken(t, user.Id, "get-models-token")

	// Bind models
	bindReq := BindTokenPricingModelsRequest{
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-4o,gpt-4o-mini",
		Bindings: []struct {
			SheetId int    `json:"sheet_id"`
			Model   string `json:"model"`
		}{
			{SheetId: sheet.Id, Model: "gpt-4o"},
			{SheetId: sheet.Id, Model: "gpt-4o-mini"},
		},
	}
	bindCtx, _ := newControllerContext(t, http.MethodPost, "/api/token/"+tokenKey(token)+"/bindings", bindReq, user.Id)
	bindCtx.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	BindTokenPricingModels(bindCtx)

	// Get models
	ctx, recorder := newControllerContext(t, http.MethodGet, "/api/token/"+tokenKey(token)+"/bindings", nil, user.Id)
	ctx.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	GetTokenPricingModels(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var resp apiResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
}

func TestGetTokenPricingModels_NoBindings(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "no-bind-get-user")
	token := seedControllerToken(t, user.Id, "no-bind-get-token")

	ctx, recorder := newControllerContext(t, http.MethodGet, "/api/token/"+tokenKey(token)+"/bindings", nil, user.Id)
	ctx.Params = gin.Params{{Key: "id", Value: tokenKey(token)}}
	GetTokenPricingModels(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

// ---------------------------------------------------------------------------
// GetAvailablePricingSheets
// ---------------------------------------------------------------------------

func TestGetAvailablePricingSheets_EnterpriseAndPlatform(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "avail-sheets-user")
	e := seedControllerEnterprise(t, "AvailCorp")
	sheet := seedControllerPricingSheet(t, e.Id, "AvailSheet")
	seedControllerPricingItem(t, sheet.Id, []string{"gpt-4o"}, model.DiscountTypeRatio, 0.7)

	// Bind user to enterprise
	binding := &model.EnterpriseUserBinding{EnterpriseId: e.Id, UserId: user.Id}
	binding.CreatedAt = time.Now().Unix()
	require.NoError(t, model.DB.Create(binding).Error)

	ctx, recorder := newControllerContext(t, http.MethodGet, "/api/pricing/sheets/available", nil, user.Id)
	GetAvailablePricingSheets(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var resp apiResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
}

func TestGetAvailablePricingSheets_NoEnterprise(t *testing.T) {
	_ = setupTokenBindingControllerDB(t)
	truncateControllerBindings(t)

	user := seedControllerUser(t, 1, "no-ent-user")

	// No enterprise binding — returns platform models (or empty)
	ctx, recorder := newControllerContext(t, http.MethodGet, "/api/pricing/sheets/available", nil, user.Id)
	GetAvailablePricingSheets(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func newControllerContext(t *testing.T, method string, target string, body any, userId int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	var requestBody *bytes.Reader
	if body != nil {
		payload, err := common.Marshal(body)
		require.NoError(t, err)
		requestBody = bytes.NewReader(payload)
	} else {
		requestBody = bytes.NewReader(nil)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, requestBody)
	if body != nil {
		ctx.Request.Header.Set("Content-Type", "application/json")
	}
	ctx.Set("id", userId)
	return ctx, recorder
}

func tokenKey(token *model.Token) string {
	return strconv.FormatInt(int64(token.Id), 10)
}

func seedControllerUser(t *testing.T, userId int, username string) *model.User {
	t.Helper()
	u := &model.User{
		Id:       userId,
		Username: username,
		Status:   1,
		Quota:    1000,
		AffCode:  username,
	}
	require.NoError(t, model.DB.Create(u).Error)
	return u
}

func seedControllerToken(t *testing.T, userId int, key string) *model.Token {
	t.Helper()
	tk := &model.Token{
		UserId: userId,
		Name:   key + "-token",
		Key:    key,
		Status: 1,
		Group:  "default",
	}
	tk.CreatedTime = 1
	tk.AccessedTime = 1
	tk.ExpiredTime = -1
	require.NoError(t, model.DB.Create(tk).Error)
	return tk
}

func seedControllerEnterprise(t *testing.T, name string) *model.Enterprise {
	t.Helper()
	e := &model.Enterprise{Name: name, EntType: model.EnterpriseTypeEnterprise, Status: model.EnterpriseStatusEnabled}
	require.NoError(t, model.DB.Create(e).Error)
	return e
}

func seedControllerUserBinding(t *testing.T, enterpriseId int, userId int) *model.EnterpriseUserBinding {
	t.Helper()
	binding := &model.EnterpriseUserBinding{
		EnterpriseId: enterpriseId,
		UserId:       userId,
		CreatedAt:    time.Now().Unix(),
	}
	require.NoError(t, model.DB.Create(binding).Error)
	return binding
}

func seedControllerPricingSheet(t *testing.T, enterpriseId int, name string) *model.EnterprisePricingSheet {
	t.Helper()
	s := &model.EnterprisePricingSheet{
		EnterpriseId: enterpriseId,
		Name:         name,
		Status:       model.PricingSheetStatusActive,
		StartTime:    1,
		EndTime:      9999999999,
	}
	require.NoError(t, model.DB.Create(s).Error)
	return s
}

func seedControllerPricingItem(t *testing.T, sheetId int, models []string, discountType string, discountValue float64) *model.EnterprisePricingItem {
	t.Helper()
	item := &model.EnterprisePricingItem{
		PricingSheetId: sheetId,
		VendorType:     "openai",
		Models:         models,
		DiscountType:   discountType,
		DiscountValue:  discountValue,
	}
	require.NoError(t, model.DB.Create(item).Error)
	return item
}
