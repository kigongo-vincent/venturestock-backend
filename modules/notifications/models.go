package notifications

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/gorm"
)

type NotificationType string

const (
	Stock NotificationType = "stock"
	Sale  NotificationType = "sale"
)

type Notification struct {
	gorm.Model
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Read        bool             `json:"read"`
	UserID      *uint            `json:"userId"`
	User        *shared.User     `json:"user" gorm:"foreignKey:UserID"`
	Path        string           `json:"path"`
	Type        NotificationType `json:"type"`
}

type NotificationService struct {
	DB *gorm.DB
}
