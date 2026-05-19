package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Enterprise status constants
const (
	EnterpriseStatusEnabled  = 1
	EnterpriseStatusDisabled = 0
)

// Enterprise represents a TO B enterprise client.
type Enterprise struct {
	Id        int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name"`
	Status    int       `json:"status"`
	Remark    string    `json:"remark"`
	CreatedAt int64     `json:"created_at"`
	UpdatedAt int64     `json:"updated_at"`
}

func (e *Enterprise) TableName() string {
	return "enterprises"
}

// Create inserts a new enterprise.
func (e *Enterprise) Create() error {
	return DB.Create(e).Error
}

// Update updates an existing enterprise.
func (e *Enterprise) Update() error {
	e.UpdatedAt = time.Now().Unix()
	return DB.Save(e).Error
}

// Delete deletes an enterprise (soft delete via GORM).
func (e *Enterprise) Delete() error {
	return DB.Delete(e).Error
}

// GetEnterpriseById retrieves an enterprise by ID.
func GetEnterpriseById(id int) (*Enterprise, error) {
	var e Enterprise
	err := DB.First(&e, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

// GetAllEnterprises returns all enterprises.
func GetAllEnterprises() ([]*Enterprise, error) {
	var enterprises []*Enterprise
	err := DB.Order("id desc").Find(&enterprises).Error
	return enterprises, err
}

// GetEnterprises returns a paginated list of enterprises.
func GetEnterprises(page, pageSize int) ([]*Enterprise, int64, error) {
	var enterprises []*Enterprise
	var total int64

	if err := DB.Model(&Enterprise{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := DB.Order("id desc").Offset(offset).Limit(pageSize).Find(&enterprises).Error
	if err != nil {
		return nil, 0, err
	}

	return enterprises, total, nil
}

// DeleteEnterprise deletes an enterprise by ID.
// Returns an error if the enterprise has associated pricing sheets.
func DeleteEnterprise(id int) error {
	sheets, err := GetPricingSheetsByEnterpriseId(id)
	if err != nil {
		return err
	}
	if len(sheets) > 0 {
		return errors.New("cannot delete enterprise with associated pricing sheets")
	}
	return DB.Delete(&Enterprise{}, id).Error
}

// IsEnterpriseEnabled checks if an enterprise is enabled.
func IsEnterpriseEnabled(id int) bool {
	e, err := GetEnterpriseById(id)
	if err != nil || e == nil {
		return false
	}
	return e.Status == EnterpriseStatusEnabled
}
