package user

import "gorm.io/gorm"

type Repository interface {
	FindByGoogleID(googleID string) (*User, error)
	CreateUser(user *User) error
	FindByID(id string) (*User, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) FindByGoogleID(googleID string) (*User, error) {
	var user User
	err := r.db.Where("google_id = ?", googleID).First(&user).Error
	return &user, err
}

func (r *repository) CreateUser(user *User) error {
	return r.db.Create(user).Error
}

func (r *repository) FindByID(id string) (*User, error) {
	var user User
	err := r.db.Where("id = ?", id).First(&user).Error
	return &user, err
}
