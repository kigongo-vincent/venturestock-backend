package branch

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/business"
	"github.com/kigongo-vincent/venturestock-backend/modules/user"
	"gorm.io/gorm"
)

type Branch struct {
	gorm.Model
	BusinessID *uint              `json:"businessId"`
	Business   *business.Business `json:"business" gorm:"foreignKey:BusinessID"`
	Name       string             `json:"name"`
	Members    []user.User        `json:"members" gorm:"many2many:branch_members"`
}

type BranchService struct {
	DB *gorm.DB
}
