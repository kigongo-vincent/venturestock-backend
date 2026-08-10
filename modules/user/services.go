package user

import (
	"gorm.io/gorm"
)

func NewService(db *gorm.DB) *UserService {
	return &UserService{DB: db}
}

func (s *UserService) Exists(user User) (bool, error) {
	var count int64
	err := s.DB.Model(&User{}).
		Where("phone = ? OR email = ?", user.Phone, user.Email).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *UserService) FindOne(user User) (User, error) {
	err := s.DB.Model(&User{}).
		Where("phone = ? OR email = ?", user.Phone, user.Email).
		First(&user).Error
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *UserService) Create(user User) (uint, error) {
	if err := s.DB.Create(&user).Error; err != nil {
		return 0, err
	}
	return user.ID, nil
}
