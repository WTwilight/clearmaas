package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Enterprise type constants
const (
	EnterpriseTypePlatform  = "platform"
	EnterpriseTypeEnterprise = "enterprise"
)

// Enterprise status constants
const (
	EnterpriseStatusEnabled  = 1
	EnterpriseStatusDisabled = 0
)

// Enterprise represents a TO B enterprise client.
type Enterprise struct {
	Id            int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name          string `json:"name"`
	EntType string `json:"type" gorm:"column:ent_type;default:enterprise"` // "platform" | "enterprise"
	Status        int    `json:"status"`
	Remark        string `json:"remark"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
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
	err := DB.Take(&e, id).Error
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
// When includePlatform is false, the built-in platform enterprise is excluded.
func GetEnterprises(page, pageSize int, includePlatform bool) ([]*Enterprise, int64, error) {
	var enterprises []*Enterprise
	var total int64

	query := DB.Model(&Enterprise{})
	if !includePlatform {
		query = query.Where("`ent_type` != ?", EnterpriseTypePlatform)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Order("id desc").Offset(offset).Limit(pageSize).Find(&enterprises).Error
	if err != nil {
		return nil, 0, err
	}

	return enterprises, total, nil
}

// DeleteEnterprise deletes an enterprise by ID.
// Returns an error if the enterprise has associated pricing sheets
// or if it is the built-in platform enterprise.
func DeleteEnterprise(id int) error {
	if IsPlatformEnterprise(id) {
		return errors.New("platform enterprise cannot be deleted")
	}
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

// IsPlatformEnterprise reports whether the given enterprise is the built-in platform enterprise.
// It checks the Type field.
func IsPlatformEnterprise(enterpriseId int) bool {
	e, err := GetEnterpriseById(enterpriseId)
	if err != nil || e == nil {
		return false
	}
	return e.EntType == EnterpriseTypePlatform
}

// IsPlatformEnterpriseByType reports whether the given enterprise is the platform enterprise by its Type field.
func IsPlatformEnterpriseByType(e *Enterprise) bool {
	return e != nil && e.EntType == EnterpriseTypePlatform
}

// IsPlatformEnterpriseId reports whether the given pricing sheet ID belongs to the platform enterprise.
// Since pricing sheets are associated with enterprises, this checks the sheet's enterprise_id.
func IsPlatformEnterpriseId(sheetId int) bool {
	sheet, err := GetPricingSheetById(sheetId)
	if err != nil || sheet == nil {
		return false
	}
	return IsPlatformEnterprise(sheet.EnterpriseId)
}

// GetPlatformEnterpriseId returns the ID of the platform enterprise by querying type='platform'.
// Returns 0 if no platform enterprise is found.
func GetPlatformEnterpriseId() int {
	e, err := GetPlatformEnterprise()
	if err != nil || e == nil {
		return 0
	}
	return e.Id
}

// GetPlatformEnterprise returns the platform enterprise record (type='platform').
func GetPlatformEnterprise() (*Enterprise, error) {
	var e Enterprise
	err := DB.Where("`ent_type` = ?", EnterpriseTypePlatform).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}
