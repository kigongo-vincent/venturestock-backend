package product

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProductAvailability string

const (
	Out ProductAvailability = "out"
	Low ProductAvailability = "low"
)

type Currency string

const (
	UGX Currency = "UGX"
)

type Price struct {
	Currency Currency `json:"currency"`
	Value    uint     `json:"value"`
}

type Quantity struct {
	Unit  string `json:"unit"`
	Value uint   `json:"value"`
}

type Feature struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Product struct {
	gorm.Model
	Photo        string                        `json:"photo"`
	Name         string                        `json:"name"`
	Availability ProductAvailability           `json:"availability" gorm:"-"`
	Ranking      string                        `json:"ranking" gorm:"-"`
	Price        datatypes.JSONType[Price]     `json:"price"`
	Quantity     datatypes.JSONType[Quantity]  `json:"quantity"`
	Features     datatypes.JSONType[[]Feature] `json:"features"`
	BranchID     *uint                         `json:"branchId"`
	Branch       *Branch                       `json:"branch" gorm:"foreignKey:BranchID"`
}

type Branch struct {
	gorm.Model
	BusinessID *uint            `json:"businessId"`
	Business   *shared.Business `json:"business" gorm:"foreignKey:BusinessID"`
	Name       string           `json:"name"`
	Members    []shared.User    `json:"members" gorm:"many2many:branch_members"`
}

type ProductService struct {
	DB *gorm.DB
}

type AdjustStockPayload struct {
	Quantity uint `json:"quantity"`
}
