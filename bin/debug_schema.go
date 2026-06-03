// +build ignore

package main

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Enterprise struct {
	Id        int    `gorm:"primaryKey;autoIncrement"`
	Name      string
	EntType   string `gorm:"column:type;default:enterprise"`
	Status    int
	Remark    string
	CreatedAt int64
	UpdatedAt int64
}

func main() {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// Create table manually with "type" column
	db.Exec(`CREATE TABLE enterprises (
		"id" integer,"name" text,"status" integer,"remark" text,
		"created_at" integer,"updated_at" integer,
		"type" TEXT NOT NULL DEFAULT '',PRIMARY KEY ("id")
	)`)

	// Parse the schema
	s, err := schema.Parse(&Enterprise{}, &schema.LoadingConfig{}, db.Dialector)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		return
	}

	fmt.Println("Schema fields:")
	for _, f := range s.Fields {
		fmt.Printf("  DBName=%s JSONName=%s FieldName=%s DataType=%s\n",
			f.DBName, f.JSONName, f.FieldName, f.DataType.String())
	}

	// Try AlterColumn on type
	type Field struct {
		DBName    string
		DataType  schema.DataType
		FieldName string
		GORMTag   string `gorm:"column:type"`
	}
}
