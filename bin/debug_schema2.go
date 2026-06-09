//go:build ignore

package main

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type EnterpriseEntType struct {
	Id        int    `gorm:"primaryKey;autoIncrement"`
	Name      string
	EntType   string `gorm:"column:type;default:enterprise"`
	Status    int
	Remark    string
	CreatedAt int64
	UpdatedAt int64
}

func (e *EnterpriseEntType) TableName() string { return "enterprises" }

func main() {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Create table with "type" column (mimics current DB state)
	db.Exec(`CREATE TABLE enterprises (
		"id" integer,"name" text,"status" integer,"remark" text,
		"created_at" integer,"updated_at" integer,
		"type" TEXT NOT NULL DEFAULT '',PRIMARY KEY ("id")
	)`)

	// Now try AutoMigrate
	err = db.AutoMigrate(&EnterpriseEntType{})
	if err != nil {
		fmt.Printf("AutoMigrate error: %v\n", err)
	} else {
		fmt.Println("AutoMigrate succeeded")
	}
}
