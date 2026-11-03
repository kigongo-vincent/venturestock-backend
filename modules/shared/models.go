package shared

import (
	"time"

	"gorm.io/gorm"
)

type Business struct {
	gorm.Model
	OwnerID      *uint                 `json:"ownerId"`
	Owner        *User                 `json:"owner" gorm:"foreignKey:OwnerID"`
	Name         string                `json:"name"`
	Location     string                `json:"location"`
	Subscription *SubscriptionSnapshot `json:"subscription,omitempty" gorm:"-"`
}

type SubscriptionSnapshot struct {
	ID          uint      `json:"id"`
	Status      string    `json:"status"`
	IsActive    bool      `json:"isActive"`
	PackageSlug string    `json:"packageSlug"`
	PackageName string    `json:"packageName"`
	StartsAt    time.Time `json:"startsAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type Branch struct {
	gorm.Model
	BusinessID *uint     `json:"businessId"`
	Business   *Business `json:"business" gorm:"foreignKey:BusinessID"`
	Name       string    `json:"name"`
	Members    []User    `json:"members" gorm:"many2many:branch_members"`
}

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
	Branch   Branch   `json:"branch" gorm:"-"`
}

type UserRole string

const (
	Admin      UserRole = "admin"
	SuperAdmin UserRole = "super-admin"
	Default    UserRole = "default"
	Manager    UserRole = "manager"
)
