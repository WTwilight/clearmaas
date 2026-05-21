package model

import (
	"errors"

	"gorm.io/gorm"
)

// EnterpriseUserBinding represents the binding between a user and an enterprise.
type EnterpriseUserBinding struct {
	Id           int   `json:"id" gorm:"primaryKey;autoIncrement"`
	EnterpriseId int   `json:"enterprise_id" gorm:"uniqueIndex:idx_enterprise_user"`
	UserId       int   `json:"user_id" gorm:"uniqueIndex:idx_enterprise_user"`
	CreatedAt    int64 `json:"created_at"`
}

func (e *EnterpriseUserBinding) TableName() string {
	return "enterprise_user_bindings"
}

// Create inserts a new binding.
func (e *EnterpriseUserBinding) Create() error {
	return DB.Create(e).Error
}

// Delete deletes a binding by user ID.
func (e *EnterpriseUserBinding) Delete() error {
	return DB.Where("user_id = ?", e.UserId).Delete(&EnterpriseUserBinding{}).Error
}

// GetUserBinding retrieves a binding by user ID.
func GetUserBinding(userId int) (*EnterpriseUserBinding, error) {
	var binding EnterpriseUserBinding
	err := DB.Where("user_id = ?", userId).Take(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

// GetBindingsByEnterpriseId returns all bindings for an enterprise.
func GetBindingsByEnterpriseId(enterpriseId int) ([]*EnterpriseUserBinding, error) {
	var bindings []*EnterpriseUserBinding
	err := DB.Where("enterprise_id = ?", enterpriseId).Find(&bindings).Error
	return bindings, err
}

// GetAllUserBindings returns all user bindings.
func GetAllUserBindings() ([]*EnterpriseUserBinding, error) {
	var bindings []*EnterpriseUserBinding
	err := DB.Find(&bindings).Error
	return bindings, err
}

// DeleteUserBinding deletes a binding by user ID.
func DeleteUserBinding(userId int) error {
	return DB.Where("user_id = ?", userId).Delete(&EnterpriseUserBinding{}).Error
}

// IsUserInEnterprise checks if a user is bound to any enterprise.
// Returns (enterpriseId, found).
func IsUserInEnterprise(userId int) (int, bool) {
	binding, err := GetUserBinding(userId)
	if err != nil || binding == nil {
		return 0, false
	}
	return binding.EnterpriseId, true
}
