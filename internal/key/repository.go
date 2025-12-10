package key

import (
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	CountActiveKeys(userID string) (int64, error)
	CreateKey(key *APIKey) error
	GetKey(keyID string, userID string) (*APIKey, error)
	GetKeyByValue(keyValue string, userID string) (*APIKey, error)
	FindByKey(keyValue string) (*APIKey, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CountActiveKeys(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&APIKey{}).Where("user_id = ? AND is_revoked = ? AND expires_at > ?", userID, false, time.Now()).Count(&count).Error
	return count, err
}

func (r *repository) CreateKey(key *APIKey) error {
	return r.db.Create(key).Error
}

func (r *repository) GetKey(keyID string, userID string) (*APIKey, error) {
	var key APIKey
	err := r.db.Where("id = ? AND user_id = ?", keyID, userID).First(&key).Error
	return &key, err
}

func (r *repository) GetKeyByValue(keyValue string, userID string) (*APIKey, error) {
	var key APIKey
	err := r.db.Where("key = ? AND user_id = ?", keyValue, userID).First(&key).Error
	return &key, err
}

func (r *repository) FindByKey(keyValue string) (*APIKey, error) {
	var key APIKey
	err := r.db.Where("key = ?", keyValue).First(&key).Error
	return &key, err
}
