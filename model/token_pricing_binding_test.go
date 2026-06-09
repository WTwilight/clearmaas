package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

// NOTE: TestMain is defined in setup_test.go — all test files share the same setup.

// ============================================================================
// IsPlatformEnterprise
// ============================================================================

func TestIsPlatformEnterprise(t *testing.T) {
	truncateBindingTestData(t)

	// Set up platform enterprise (type='platform') and a regular enterprise
	now := time.Now().Unix()
	// Delete all enterprises first to avoid auto-increment conflicts.
	DB.Exec("DELETE FROM enterprises")
	// Insert platform enterprise with known id=-1.
	require.NoError(t, DB.Exec(
		"INSERT INTO enterprises (id, name, type, status, created_at, updated_at) VALUES (-1, '平台', ?, 1, ?, ?)",
		EnterpriseTypePlatform, now, now,
	).Error)
	// Insert regular enterprise with explicit id=2.
	require.NoError(t, DB.Exec(
		"INSERT INTO enterprises (id, name, type, status, created_at, updated_at) VALUES (2, 'RegularCorp', ?, 1, ?, ?)",
		EnterpriseTypeEnterprise, now, now,
	).Error)

	tests := []struct {
		name         string
		enterpriseId int
		want         bool
	}{
		{"platform enterprise id returns true", -1, true},
		{"regular enterprise id returns false", 1, false},
		{"zero returns false", 0, false},
		{"large positive returns false", 999999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPlatformEnterprise(tt.enterpriseId)
			if got != tt.want {
				t.Errorf("IsPlatformEnterprise(%d) = %v, want %v", tt.enterpriseId, got, tt.want)
			}
		})
	}
}

// ============================================================================
// TokenPricingModelBinding CRUD
// ============================================================================

func TestTokenPricingModelBinding_CRUD(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "crud-user")
	token := seedBindingToken(t, user.Id, "crud-key")
	e := seedBindingEnterprise(t, "CRUDCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "CRUDSheet")

	// Create
	binding := &TokenPricingModelBinding{
		UserId:        user.Id,
		TokenId:       token.Id,
		PricingSheetId: sheet.Id,
		Model:         "gpt-4o",
	}
	binding.CreatedAt = time.Now().Unix()
	require.NoError(t, binding.Create())
	require.NotZero(t, binding.Id)

	// Read by ID
	found, err := GetTokenPricingModelBindingById(binding.Id)
	require.NoError(t, err)
	require.NotNil(t, found)
	if found.UserId != user.Id || found.TokenId != token.Id || found.PricingSheetId != sheet.Id || found.Model != "gpt-4o" {
		t.Errorf("binding mismatch: got %+v", found)
	}

	// Delete
	err = binding.Delete()
	require.NoError(t, err)

	found2, err := GetTokenPricingModelBindingById(binding.Id)
	require.NoError(t, err)
	require.Nil(t, found2)
}

func TestTokenPricingModelBinding_GetById_NotFound(t *testing.T) {
	truncateBindingTestData(t)
	found, err := GetTokenPricingModelBindingById(99999)
	require.NoError(t, err)
	require.Nil(t, found)
}

func TestGetTokenPricingModelBindings(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "list-user")
	token := seedBindingToken(t, user.Id, "list-key")
	e := seedBindingEnterprise(t, "ListCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "ListSheet")

	for _, m := range []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"} {
		b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token.Id, PricingSheetId: sheet.Id, Model: m}
		b.CreatedAt = time.Now().Unix()
		require.NoError(t, b.Create())
	}

	bindings, err := GetTokenPricingModelBindings(token.Id)
	require.NoError(t, err)
	require.Len(t, bindings, 3)

	// Non-existent token returns empty
	bindings2, err := GetTokenPricingModelBindings(99999)
	require.NoError(t, err)
	require.Len(t, bindings2, 0)
}

func TestGetTokenPricingModelBindings_MultipleTokens(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "multi-token-user")
	e := seedBindingEnterprise(t, "MultiTokenCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "MultiTokenSheet")

	token1 := seedBindingToken(t, user.Id, "mt-key-1")
	token2 := seedBindingToken(t, user.Id, "mt-key-2")

	b1 := &TokenPricingModelBinding{UserId: user.Id, TokenId: token1.Id, PricingSheetId: sheet.Id, Model: "gpt-4o"}
	b1.CreatedAt = time.Now().Unix()
	require.NoError(t, b1.Create())

	b2 := &TokenPricingModelBinding{UserId: user.Id, TokenId: token2.Id, PricingSheetId: sheet.Id, Model: "claude-3-5-sonnet"}
	b2.CreatedAt = time.Now().Unix()
	require.NoError(t, b2.Create())

	bindings1, err := GetTokenPricingModelBindings(token1.Id)
	require.NoError(t, err)
	require.Len(t, bindings1, 1)
	if bindings1[0].Model != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", bindings1[0].Model)
	}

	bindings2, err := GetTokenPricingModelBindings(token2.Id)
	require.NoError(t, err)
	require.Len(t, bindings2, 1)
	if bindings2[0].Model != "claude-3-5-sonnet" {
		t.Errorf("expected claude-3-5-sonnet, got %s", bindings2[0].Model)
	}
}

func TestGetTokenPricingModelBindingByTokenAndModel_Found(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "find-user")
	token := seedBindingToken(t, user.Id, "find-key")
	e := seedBindingEnterprise(t, "FindCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "FindSheet")

	b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token.Id, PricingSheetId: sheet.Id, Model: "gpt-4o"}
	b.CreatedAt = time.Now().Unix()
	require.NoError(t, b.Create())

	found, err := GetTokenPricingModelBindingByTokenAndModel(token.Id, "gpt-4o")
	require.NoError(t, err)
	require.NotNil(t, found)
	if found.PricingSheetId != sheet.Id || found.Model != "gpt-4o" {
		t.Errorf("mismatch: got %+v", found)
	}
}

func TestGetTokenPricingModelBindingByTokenAndModel_NotFound(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "notfound-user")
	token := seedBindingToken(t, user.Id, "notfound-key")
	e := seedBindingEnterprise(t, "NotFoundCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "NotFoundSheet")

	b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token.Id, PricingSheetId: sheet.Id, Model: "gpt-4o"}
	b.CreatedAt = time.Now().Unix()
	require.NoError(t, b.Create())

	// Token matches but model doesn't
	found, err := GetTokenPricingModelBindingByTokenAndModel(token.Id, "claude-3-5-sonnet")
	require.NoError(t, err)
	require.Nil(t, found)

	// Model matches but token doesn't
	found2, err := GetTokenPricingModelBindingByTokenAndModel(99999, "gpt-4o")
	require.NoError(t, err)
	require.Nil(t, found2)
}

// ============================================================================
// DeleteTokenPricingModelBindings — transaction-aware
// ============================================================================

func TestDeleteTokenPricingModelBindings_WithTx(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "cascade-user")
	token := seedBindingToken(t, user.Id, "cascade-key")
	e := seedBindingEnterprise(t, "CascadeCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "CascadeSheet")

	for _, m := range []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"} {
		b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token.Id, PricingSheetId: sheet.Id, Model: m}
		b.CreatedAt = time.Now().Unix()
		require.NoError(t, b.Create())
	}

	tx := DB.Begin()
	err := DeleteTokenPricingModelBindings(token.Id, tx)
	require.NoError(t, err)
	tx.Commit()

	bindings, err := GetTokenPricingModelBindings(token.Id)
	require.NoError(t, err)
	require.Len(t, bindings, 0)
}

func TestDeleteTokenPricingModelBindings_Rollback(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "rollback-user")
	token := seedBindingToken(t, user.Id, "rollback-key")
	e := seedBindingEnterprise(t, "RollbackCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "RollbackSheet")

	for _, m := range []string{"gpt-4o", "gpt-4o-mini"} {
		b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token.Id, PricingSheetId: sheet.Id, Model: m}
		b.CreatedAt = time.Now().Unix()
		require.NoError(t, b.Create())
	}

	tx := DB.Begin()
	err := DeleteTokenPricingModelBindings(token.Id, tx)
	require.NoError(t, err)
	tx.Rollback()

	bindings, err := GetTokenPricingModelBindings(token.Id)
	require.NoError(t, err)
	require.Len(t, bindings, 2, "rollback should preserve bindings")
}

func TestDeleteTokenPricingModelBindings_Empty(t *testing.T) {
	truncateBindingTestData(t)

	tx := DB.Begin()
	err := DeleteTokenPricingModelBindings(99999, tx)
	require.NoError(t, err)
	tx.Commit()
}

func TestDeleteTokenPricingModelBindings_IsolatedToToken(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "isolate-user")
	e := seedBindingEnterprise(t, "IsolateCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "IsolateSheet")

	token1 := seedBindingToken(t, user.Id, "iso-key-1")
	token2 := seedBindingToken(t, user.Id, "iso-key-2")

	b1 := &TokenPricingModelBinding{UserId: user.Id, TokenId: token1.Id, PricingSheetId: sheet.Id, Model: "gpt-4o"}
	b1.CreatedAt = time.Now().Unix()
	require.NoError(t, b1.Create())

	b2 := &TokenPricingModelBinding{UserId: user.Id, TokenId: token2.Id, PricingSheetId: sheet.Id, Model: "claude-3-5-sonnet"}
	b2.CreatedAt = time.Now().Unix()
	require.NoError(t, b2.Create())

	tx := DB.Begin()
	err := DeleteTokenPricingModelBindings(token1.Id, tx)
	require.NoError(t, err)
	tx.Commit()

	bindings1, err := GetTokenPricingModelBindings(token1.Id)
	require.NoError(t, err)
	require.Len(t, bindings1, 0)

	bindings2, err := GetTokenPricingModelBindings(token2.Id)
	require.NoError(t, err)
	require.Len(t, bindings2, 1)
	if bindings2[0].Model != "claude-3-5-sonnet" {
		t.Errorf("expected claude-3-5-sonnet, got %s", bindings2[0].Model)
	}
}

// ============================================================================
// CountTokenBindingsBySheetId
// ============================================================================

func TestCountTokenBindingsBySheetId(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "count-user")
	e := seedBindingEnterprise(t, "CountCorp")
	sheet1 := seedBindingPricingSheet(t, e.Id, "Sheet1")
	sheet2 := seedBindingPricingSheet(t, e.Id, "Sheet2")

	token1 := seedBindingToken(t, user.Id, "count-key-1")
	token2 := seedBindingToken(t, user.Id, "count-key-2")

	for _, m := range []string{"gpt-4o", "gpt-4o-mini"} {
		b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token1.Id, PricingSheetId: sheet1.Id, Model: m}
		b.CreatedAt = time.Now().Unix()
		require.NoError(t, b.Create())
	}
	b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token2.Id, PricingSheetId: sheet2.Id, Model: "claude-3-5-sonnet"}
	b.CreatedAt = time.Now().Unix()
	require.NoError(t, b.Create())

	count1, err := CountTokenBindingsBySheetId(sheet1.Id)
	require.NoError(t, err)
	if count1 != 2 {
		t.Errorf("expected 2, got %d", count1)
	}

	count2, err := CountTokenBindingsBySheetId(sheet2.Id)
	require.NoError(t, err)
	if count2 != 1 {
		t.Errorf("expected 1, got %d", count2)
	}
}

func TestCountTokenBindingsBySheetId_Zero(t *testing.T) {
	truncateBindingTestData(t)
	count, err := CountTokenBindingsBySheetId(99999)
	require.NoError(t, err)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

// NOTE: The (token_id, model) uniqueness is enforced at the Controller/service layer,
// not at the database level (no unique index defined on the model).
// Hence no duplicate-constraint test needed here.

// ============================================================================
// Platform enterprise (type='platform')
// ============================================================================

func TestTokenPricingModelBinding_PlatformEnterprise(t *testing.T) {
	truncateBindingTestData(t)

	now := time.Now().Unix()
	// Ensure platform enterprise with known id=-1 for test stability.
	// type='platform' is set to match production logic (IsPlatformEnterprise uses type field).
	require.NoError(t, DB.Exec(
		"INSERT OR REPLACE INTO enterprises (id, name, type, status, created_at, updated_at) VALUES (-1, ?, ?, 1, ?, ?)",
		"平台", EnterpriseTypePlatform, now, now,
	).Error)

	user := seedBindingUser(t, 1, "platform-user")
	token := seedBindingToken(t, user.Id, "platform-key")

	platformSheet := &EnterprisePricingSheet{
		EnterpriseId: -1,
		Name:         "平台默认报价单",
		Status:       PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
	}
	platformSheet.CreatedAt = now
	platformSheet.UpdatedAt = now
	require.NoError(t, DB.Create(platformSheet).Error)

	binding := &TokenPricingModelBinding{
		UserId:        user.Id,
		TokenId:       token.Id,
		PricingSheetId: platformSheet.Id,
		Model:         "gpt-4o",
	}
	binding.CreatedAt = now
	require.NoError(t, binding.Create())

	found, err := GetTokenPricingModelBindingByTokenAndModel(token.Id, "gpt-4o")
	require.NoError(t, err)
	require.NotNil(t, found)
	if found.PricingSheetId != platformSheet.Id {
		t.Errorf("expected sheet %d, got %d", platformSheet.Id, found.PricingSheetId)
	}

	count, err := CountTokenBindingsBySheetId(platformSheet.Id)
	require.NoError(t, err)
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

// ============================================================================
// Token deletion cascade
// ============================================================================

func TestDeleteTokenById_CascadeBindings(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "del-cascade-user")
	token := seedBindingToken(t, user.Id, "del-cascade-key")
	e := seedBindingEnterprise(t, "DelCascadeCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "DelCascadeSheet")

	for _, m := range []string{"gpt-4o", "gpt-4o-mini", "claude-3-5-sonnet"} {
		b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token.Id, PricingSheetId: sheet.Id, Model: m}
		b.CreatedAt = time.Now().Unix()
		require.NoError(t, b.Create())
	}

	err := DeleteTokenById(token.Id, user.Id)
	require.NoError(t, err)

	bindings, err := GetTokenPricingModelBindings(token.Id)
	require.NoError(t, err)
	require.Len(t, bindings, 0, "bindings should be cascade-deleted with token")
}

func TestDeleteTokenById_CascadeBindings_OtherBindingsUntouched(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "other-untouch-user")
	e := seedBindingEnterprise(t, "OtherUntouchCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "OtherUntouchSheet")

	token1 := seedBindingToken(t, user.Id, "cascade-key-1")
	token2 := seedBindingToken(t, user.Id, "cascade-key-2")

	for _, m := range []string{"gpt-4o", "gpt-4o-mini"} {
		b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token1.Id, PricingSheetId: sheet.Id, Model: m}
		b.CreatedAt = time.Now().Unix()
		require.NoError(t, b.Create())
	}
	b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token2.Id, PricingSheetId: sheet.Id, Model: "claude-3-5-sonnet"}
	b.CreatedAt = time.Now().Unix()
	require.NoError(t, b.Create())

	err := DeleteTokenById(token1.Id, user.Id)
	require.NoError(t, err)

	bindings1, err := GetTokenPricingModelBindings(token1.Id)
	require.NoError(t, err)
	require.Len(t, bindings1, 0)

	bindings2, err := GetTokenPricingModelBindings(token2.Id)
	require.NoError(t, err)
	require.Len(t, bindings2, 1)
}

func TestDeleteTokenById_NoBindings(t *testing.T) {
	truncateBindingTestData(t)
	user := seedBindingUser(t, 1, "no-bind-del-user")
	token := seedBindingToken(t, user.Id, "no-bind-del-key")
	err := DeleteTokenById(token.Id, user.Id)
	require.NoError(t, err)
}

func TestDeleteTokenById_NonExistent(t *testing.T) {
	truncateBindingTestData(t)
	err := DeleteTokenById(99999, 99999)
	require.Error(t, err)
}

// ============================================================================
// Pricing sheet deletion protection
// ============================================================================

func TestDeletePricingSheet_Protection_WithBindings(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "protect-user")
	token := seedBindingToken(t, user.Id, "protect-key")
	e := seedBindingEnterprise(t, "ProtectCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "ProtectSheet")

	b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token.Id, PricingSheetId: sheet.Id, Model: "gpt-4o"}
	b.CreatedAt = time.Now().Unix()
	require.NoError(t, b.Create())

	err := DeletePricingSheet(sheet.Id)
	require.Error(t, err, "deleting a sheet with active bindings should be protected")

	found, err := GetPricingSheetById(sheet.Id)
	require.NoError(t, err)
	require.NotNil(t, found, "sheet should not be deleted")
}

func TestDeletePricingSheet_Protection_WithMultipleBindings(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "multi-protect-user")
	e := seedBindingEnterprise(t, "MultiProtectCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "MultiProtectSheet")

	for i := 0; i < 3; i++ {
		tk := seedBindingToken(t, user.Id, "multi-protect-key-"+string(rune('a'+i)))
		b := &TokenPricingModelBinding{UserId: user.Id, TokenId: tk.Id, PricingSheetId: sheet.Id, Model: "gpt-4o"}
		b.CreatedAt = time.Now().Unix()
		require.NoError(t, b.Create())
	}

	err := DeletePricingSheet(sheet.Id)
	require.Error(t, err, "deleting a sheet with multiple bindings should be protected")
}

func TestDeletePricingSheet_Protection_WithPricingItems(t *testing.T) {
	truncateBindingTestData(t)

	e := seedBindingEnterprise(t, "ItemsProtectCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "ItemsProtectSheet")

	item := &EnterprisePricingItem{
		PricingSheetId: sheet.Id,
		VendorType:    "openai",
		Models:        []string{"gpt-4o"},
		DiscountType:  DiscountTypeRatio,
		DiscountValue: 0.7,
	}
	require.NoError(t, DB.Create(item).Error)

	err := DeletePricingSheet(sheet.Id)
	require.Error(t, err, "deleting a sheet with pricing items should be protected")
}

func TestDeletePricingSheet_Protection_CannotDeletePlatformSheet(t *testing.T) {
	truncateBindingTestData(t)

	now := time.Now().Unix()
	require.NoError(t, DB.Exec(
		"INSERT INTO enterprises (id, name, type, status, created_at, updated_at) VALUES (-1, '平台', ?, 1, ?, ?)",
		EnterpriseTypePlatform, now, now,
	).Error)

	sheet := &EnterprisePricingSheet{
		EnterpriseId: -1,
		Name:         "PlatformDefaultSheet",
		Status:       PricingSheetStatusActive,
		StartTime:    now - 86400,
		EndTime:      now + 86400,
	}
	sheet.CreatedAt = now
	sheet.UpdatedAt = now
	require.NoError(t, DB.Create(sheet).Error)

	err := DeletePricingSheet(sheet.Id)
	require.Error(t, err, "cannot delete platform enterprise's pricing sheet")
}

func TestDeletePricingSheet_NoBindingsNoItems(t *testing.T) {
	truncateBindingTestData(t)

	e := seedBindingEnterprise(t, "NoBindingsCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "NoBindingsSheet")

	err := DeletePricingSheet(sheet.Id)
	require.NoError(t, err, "deleting a sheet without bindings or items should succeed")

	found, err := GetPricingSheetById(sheet.Id)
	require.NoError(t, err)
	require.Nil(t, found)
}

func TestDeletePricingSheet_NonExistent(t *testing.T) {
	truncateBindingTestData(t)
	err := DeletePricingSheet(99999)
	require.NoError(t, err, "deleting non-existent sheet should not error")
}

func TestDeletePricingSheet_DisabledEnterprise(t *testing.T) {
	truncateBindingTestData(t)

	e := seedBindingEnterprise(t, "DisabledProtectCorp")
	e.Status = common.UserStatusDisabled
	DB.Save(e)

	sheet := seedBindingPricingSheet(t, e.Id, "DisabledProtectSheet")

	err := DeletePricingSheet(sheet.Id)
	require.NoError(t, err)
}

func TestDeletePricingSheet_ForceDelete_WithBindings(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "force-user")
	token := seedBindingToken(t, user.Id, "force-key")
	e := seedBindingEnterprise(t, "ForceCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "ForceSheet")

	b := &TokenPricingModelBinding{UserId: user.Id, TokenId: token.Id, PricingSheetId: sheet.Id, Model: "gpt-4o"}
	b.CreatedAt = time.Now().Unix()
	require.NoError(t, b.Create())

	// Force delete: remove bindings then delete sheet
	tx := DB.Begin()
	err := DeleteTokenPricingModelBindings(token.Id, tx)
	require.NoError(t, err)
	err = tx.Where("id = ?", sheet.Id).Delete(&EnterprisePricingSheet{}).Error
	require.NoError(t, err)
	err = tx.Commit().Error
	require.NoError(t, err)

	found, err := GetPricingSheetById(sheet.Id)
	require.NoError(t, err)
	require.Nil(t, found)

	bindings, err := GetTokenPricingModelBindings(token.Id)
	require.NoError(t, err)
	require.Len(t, bindings, 0)
}

// ============================================================================
// SyncTokenPricingModelBindings
// ============================================================================

func TestSyncTokenPricingModelBindings(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "sync-user")
	token := seedBindingToken(t, user.Id, "sync-key")
	e := seedBindingEnterprise(t, "SyncCorp")
	sheet := seedBindingPricingSheet(t, e.Id, "SyncSheet")

	// Initially has gpt-4o
	b1 := &TokenPricingModelBinding{UserId: user.Id, TokenId: token.Id, PricingSheetId: sheet.Id, Model: "gpt-4o"}
	b1.CreatedAt = time.Now().Unix()
	require.NoError(t, b1.Create())

	// Sync to gpt-4o-mini, claude-3-5-sonnet
	tx := DB.Begin()
	err := SyncTokenPricingModelBindings(user.Id, token.Id, []string{"gpt-4o-mini", "claude-3-5-sonnet"}, sheet.Id, tx)
	require.NoError(t, err)
	err = tx.Commit().Error
	require.NoError(t, err)

	bindings, err := GetTokenPricingModelBindings(token.Id)
	require.NoError(t, err)
	require.Len(t, bindings, 2)

	models := make(map[string]bool)
	for _, b := range bindings {
		models[b.Model] = true
	}
	if !models["gpt-4o-mini"] || !models["claude-3-5-sonnet"] {
		t.Errorf("expected gpt-4o-mini and claude-3-5-sonnet, got %v", models)
	}
	if models["gpt-4o"] {
		t.Errorf("gpt-4o should have been removed")
	}
}

func TestSyncTokenPricingModelBindings_Empty(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "sync-empty-user")
	token := seedBindingToken(t, user.Id, "sync-empty-key")

	// Sync to empty list
	tx := DB.Begin()
	err := SyncTokenPricingModelBindings(user.Id, token.Id, []string{}, 0, tx)
	require.NoError(t, err)
	err = tx.Commit().Error
	require.NoError(t, err)

	bindings, err := GetTokenPricingModelBindings(token.Id)
	require.NoError(t, err)
	require.Len(t, bindings, 0)
}

// ============================================================================
// ModelLimits sync helpers
// ============================================================================

func TestToken_GetModelLimitsMap_FromLegacyField(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "legacy-user")
	token := &Token{
		UserId:             user.Id,
		Name:               "legacy-token",
		Key:                "legacy-key",
		Status:             common.TokenStatusEnabled,
		CreatedTime:        time.Now().Unix(),
		AccessedTime:       time.Now().Unix(),
		ExpiredTime:        -1,
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-4o,gpt-4o-mini,claude-3-5-sonnet",
		Group:              "default",
	}
	require.NoError(t, DB.Create(token).Error)

	limitsMap := token.GetModelLimitsMap()
	if limitsMap["gpt-4o"] != true || limitsMap["gpt-4o-mini"] != true || limitsMap["claude-3-5-sonnet"] != true {
		t.Errorf("unexpected limitsMap: %+v", limitsMap)
	}
}

func TestToken_GetModelLimitsMap_Empty(t *testing.T) {
	truncateBindingTestData(t)

	user := seedBindingUser(t, 1, "empty-limits-user")
	token := &Token{
		UserId:             user.Id,
		Name:               "empty-token",
		Key:                "empty-key",
		Status:             common.TokenStatusEnabled,
		CreatedTime:        time.Now().Unix(),
		AccessedTime:       time.Now().Unix(),
		ExpiredTime:        -1,
		ModelLimitsEnabled: false,
		ModelLimits:        "",
		Group:              "default",
	}
	require.NoError(t, DB.Create(token).Error)

	if len(token.GetModelLimits()) != 0 {
		t.Errorf("expected empty limits, got %v", token.GetModelLimits())
	}
	if len(token.GetModelLimitsMap()) != 0 {
		t.Errorf("expected empty limits map, got %+v", token.GetModelLimitsMap())
	}
}

// ============================================================================
// Helpers
// ============================================================================

func truncateBindingTestData(t *testing.T) {
	t.Helper()
	// Re-create platform enterprise so tests that need it always have it available.
	DB.Exec(
		"INSERT OR REPLACE INTO enterprises (name, type, status, created_at, updated_at) VALUES (?, ?, 1, ?, ?)",
		"平台", EnterpriseTypePlatform, time.Now().Unix(), time.Now().Unix(),
	)
	t.Cleanup(func() {
		DB.Exec("DELETE FROM token_pricing_model_bindings")
		DB.Exec("DELETE FROM tokens")
		DB.Exec("DELETE FROM enterprise_pricing_items")
		DB.Exec("DELETE FROM enterprise_pricing_sheets")
		DB.Exec("DELETE FROM enterprise_user_bindings")
		DB.Exec("DELETE FROM enterprises")
		DB.Exec("DELETE FROM users")
	})
}

func seedBindingUser(t *testing.T, userId int, username string) *User {
	t.Helper()
	u := &User{
		Id:       userId,
		Username: username,
		Status:   common.UserStatusEnabled,
		Quota:    1000,
		AffCode:  username,
	}
	u.CreatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(u).Error)
	return u
}

func seedBindingToken(t *testing.T, userId int, key string) *Token {
	t.Helper()
	tk := &Token{
		UserId:  userId,
		Name:    key + "-token",
		Key:     key,
		Status:  common.TokenStatusEnabled,
		Group:   "default",
	}
	tk.CreatedTime = time.Now().Unix()
	tk.AccessedTime = time.Now().Unix()
	tk.ExpiredTime = -1
	require.NoError(t, DB.Create(tk).Error)
	return tk
}

func seedBindingEnterprise(t *testing.T, name string) *Enterprise {
	t.Helper()
	e := &Enterprise{Name: name, EntType: EnterpriseTypeEnterprise, Status: EnterpriseStatusEnabled}
	e.CreatedAt = time.Now().Unix()
	e.UpdatedAt = time.Now().Unix()
	require.NoError(t, DB.Create(e).Error)
	return e
}

func seedBindingPricingSheet(t *testing.T, enterpriseId int, name string) *EnterprisePricingSheet {
	t.Helper()
	now := time.Now().Unix()
	sheet := &EnterprisePricingSheet{
		EnterpriseId: enterpriseId,
		Name:        name,
		Status:      PricingSheetStatusActive,
		StartTime:   now - 86400,
		EndTime:     now + 86400,
	}
	sheet.CreatedAt = now
	sheet.UpdatedAt = now
	require.NoError(t, DB.Create(sheet).Error)
	return sheet
}
