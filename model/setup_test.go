package model

import (
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Shared TestMain for the model package.
// All test files in this package must use this TestMain.
// Do NOT define another TestMain in any other _test.go file.
func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db: " + err.Error())
	}

	DB = db
	LOG_DB = db

	common.UsingSQLite = true
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true

	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(1)

	// AutoMigrate ALL models used across the model package tests.
	// This is the single source of truth for test DB schema.
	if err := db.AutoMigrate(
		&Task{},
		&User{},
		&Token{},
		&Log{},
		&Channel{},
		&TopUp{},
		&SubscriptionPlan{},
		&SubscriptionOrder{},
		&UserSubscription{},
		&Enterprise{},
		&EnterprisePricingSheet{},
		&EnterprisePricingItem{},
		&EnterpriseUserBinding{},
		&EnterprisePricingSheetChannel{},
		&Supplier{},
		&SupplierPricingSheet{},
		&SupplierPricingItem{},
		&SupplierPricingSheetChannel{},
		&TokenPricingModelBinding{},
	); err != nil {
		panic("failed to migrate: " + err.Error())
	}

	os.Exit(m.Run())
}
