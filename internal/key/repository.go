package key

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	CountActiveKeys(userID string) (int64, error)
	CreateKey(key *APIKey) error
	GetKey(keyID string, userID string) (*APIKey, error)
	GetKeyByValue(keyValue string, userID string) (*APIKey, error)
	FindByKey(keyValue string) (*APIKey, error)
	GetKeysByUserID(userID string) ([]APIKey, error)
	RevokeKey(keyID string, userID string) error
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

func (r *repository) GetKeysByUserID(userID string) ([]APIKey, error) {
	var keys []APIKey
	err := r.db.Where("user_id = ?", userID).Order("created_at desc").Find(&keys).Error
	return keys, err
}

func (r *repository) RevokeKey(keyID string, userID string) error {
	result := r.db.Model(&APIKey{}).Where("id = ? AND user_id = ?", keyID, userID).Update("is_revoked", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *repository) GetKeyByValue(keyValue string, userID string) (*APIKey, error) {
	hashedKey := hashKey(keyValue)
	var key APIKey
	err := r.db.Where("key = ? AND user_id = ?", hashedKey, userID).First(&key).Error
	return &key, err
}

func (r *repository) FindByKey(keyValue string) (*APIKey, error) {
	hashedKey := hashKey(keyValue)
	var key APIKey
	err := r.db.Where("key = ?", hashedKey).First(&key).Error
	return &key, err
}

func hashKey(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}
