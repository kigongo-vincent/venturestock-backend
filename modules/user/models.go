package user

import "gorm.io/gorm"

type UserRole string

const (
	Admin      UserRole = "admin"
	SuperAdmin UserRole = "super-admin"
	Default    UserRole = "default"
	Manager    UserRole = "manager"
)

type User struct {
	gorm.Model
	Email    string   `json:"email"`
	Phone    string   `json:"phone"`
	Name     string   `json:"name"`
	Role     UserRole `json:"role" gorm:"default:default"`
	Photo    string   `json:"photo"`
	IsActive bool     `json:"isActive" gorm:"default:false"`
	Password string   `json:"password"`
	GoogleID string   `json:"googleId"`
}

type UserService struct {
	DB *gorm.DB
}
