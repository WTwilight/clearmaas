package service

import (
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Shared TestMain for the service package.
// All test files in this package must NOT define their own TestMain.
// Do NOT add another TestMain in any other _test.go file in this package.
func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db: " + err.Error())
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(1)

	model.DB = db
	model.LOG_DB = db

	common.UsingSQLite = true
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true

	if err := db.AutoMigrate(
		&model.Task{},
		&model.User{},
		&model.Token{},
		&model.Log{},
		&model.Channel{},
		&model.Ability{},
		&model.Model{},
		&model.Vendor{},
		&model.PerfMetric{},
		&model.TopUp{},
		&model.UserSubscription{},
		&model.Enterprise{},
		&model.EnterprisePricingSheet{},
		&model.EnterprisePricingItem{},
		&model.EnterpriseUserBinding{},
		&model.EnterprisePricingSheetChannel{},
		&model.Supplier{},
		&model.SupplierPricingSheet{},
		&model.SupplierPricingItem{},
		&model.SupplierPricingSheetChannel{},
		&model.TokenPricingModelBinding{},
	); err != nil {
		panic("failed to migrate: " + err.Error())
	}

	os.Exit(m.Run())
}
