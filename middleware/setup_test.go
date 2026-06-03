package middleware

import (
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Shared TestMain for the middleware package.
// All test files in this package must NOT define their own TestMain.
// Do NOT add another TestMain in any other _test.go file in this package.
func TestMain(m *testing.M) {
	// Initialize i18n so that i18n.T() works in middleware tests.
	if err := i18n.Init(); err != nil {
		panic("i18n.Init() failed: " + err.Error())
	}

	// Initialize DB for middleware tests that need it.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to open test db: " + err.Error())
	}
	model.DB = db
	model.LOG_DB = db

	common.UsingSQLite = true
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true

	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(1)

	_ = db.AutoMigrate(
		&model.User{},
		&model.Token{},
		&model.Channel{},
		&model.Log{},
		&model.Enterprise{},
		&model.EnterprisePricingSheet{},
		&model.EnterprisePricingItem{},
		&model.EnterpriseUserBinding{},
		&model.Task{},
		&model.TokenPricingModelBinding{},
	)

	os.Exit(m.Run())
}
