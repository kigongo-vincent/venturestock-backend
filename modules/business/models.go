package business

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/user"
	"gorm.io/gorm"
)

type Business struct {
	gorm.Model
	OwnerID  *uint      `json:"ownerId"`
	Owner    *user.User `json:"owner" gorm:"foreignKey:OwnerID"`
	Name     string     `json:"name"`
	Location string     `json:"location"`
}

type BusinessService struct {
	DB *gorm.DB
}
