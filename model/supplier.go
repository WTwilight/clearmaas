package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Supplier status constants
const (
	SupplierStatusEnabled  = 1
	SupplierStatusDisabled = 0
)

// Supplier represents a supplier/vendor for pricing.
type Supplier struct {
	Id        int   `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name"`
	Status    int    `json:"status"`
	Remark    string `json:"remark"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func (s *Supplier) TableName() string {
	return "suppliers"
}

// Create inserts a new supplier.
func (s *Supplier) Create() error {
	return DB.Create(s).Error
}

// Update updates an existing supplier.
func (s *Supplier) Update() error {
	s.UpdatedAt = time.Now().Unix()
	return DB.Save(s).Error
}

// GetSupplierById retrieves a supplier by ID.
// Returns nil, nil if the supplier does not exist.
func GetSupplierById(id int) (*Supplier, error) {
	var s Supplier
	err := DB.Take(&s, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// GetAllSuppliers returns all suppliers ordered by id desc.
func GetAllSuppliers() ([]*Supplier, error) {
	var suppliers []*Supplier
	err := DB.Order("id desc").Find(&suppliers).Error
	return suppliers, err
}

// GetSuppliers returns a paginated list of suppliers.
func GetSuppliers(page, pageSize int) ([]*Supplier, int64, error) {
	var suppliers []*Supplier
	var total int64

	if err := DB.Model(&Supplier{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := DB.Order("id desc").Offset(offset).Limit(pageSize).Find(&suppliers).Error
	if err != nil {
		return nil, 0, err
	}

	return suppliers, total, nil
}

// DeleteSupplier deletes a supplier by ID (soft delete).
func DeleteSupplier(id int) error {
	return DB.Delete(&Supplier{}, id).Error
}

// IsSupplierEnabled checks if a supplier is enabled.
func IsSupplierEnabled(supplierId int) bool {
	s, err := GetSupplierById(supplierId)
	if err != nil || s == nil {
		return false
	}
	return s.Status == SupplierStatusEnabled
}
