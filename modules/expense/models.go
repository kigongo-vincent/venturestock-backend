package expense

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/product"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Expense struct {
	gorm.Model
	Description string                            `json:"description"`
	Amount      datatypes.JSONType[product.Price] `json:"amount"`
	UserID      *uint                             `json:"userId"`
	User        *shared.User                      `json:"user" gorm:"foreignKey:UserID"`
	BranchID    *uint                             `json:"branchId"`
	Branch      *product.Branch                   `json:"branch" gorm:"foreignKey:BranchID"`
}
