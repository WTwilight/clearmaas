package helper

import (
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Shared TestMain for the relay/helper package.
// All test files in this package must NOT define their own TestMain.
// Do NOT add another TestMain in any other _test.go file in this package.
func TestMain(m *testing.M) {
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

	if err := db.AutoMigrate(
		&model.Enterprise{},
		&model.EnterprisePricingSheet{},
		&model.EnterprisePricingItem{},
		&model.EnterpriseUserBinding{},
		&model.User{},
		&model.Token{},
		&model.TokenPricingModelBinding{},
	); err != nil {
		panic("failed to migrate: " + err.Error())
	}

	// Seed test data for billing integration tests (defined in enterprise_pricing_billing_test.go)
	seedBillingTestData(db)

	_ = config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{}`,
	})

	os.Exit(m.Run())
}
