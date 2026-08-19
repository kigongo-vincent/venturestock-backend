package branch

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/product"
	"gorm.io/gorm"
)

type BranchService struct {
	DB *gorm.DB
}

type BranchSummary struct {
	ID      uint          `json:"ID"`
	Name    string        `json:"name"`
	Workers uint          `json:"workers"`
	Revenue product.Price `json:"revenue"`
}

type BranchDetails struct {
	ID      uint          `json:"ID"`
	Name    string        `json:"name"`
	Workers uint          `json:"workers"`
	Revenue product.Price `json:"revenue"`
}

type BranchReports struct {
	TotalSaleValue product.Price `json:"totalSaleValue"`
	TotalSales     uint          `json:"totalSales"`
}

type BranchUser struct {
	ID      uint          `json:"ID"`
	Name    string        `json:"name"`
	Phone   string        `json:"phone"`
	Photo   string        `json:"photo"`
	Revenue product.Price `json:"revenue"`
}

type CreateBranchPayload struct {
	Name string `json:"name"`
}
