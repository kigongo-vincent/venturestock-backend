package analytics

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/product"
	"gorm.io/gorm"
)

type AnalyticsService struct {
	DB *gorm.DB
}

type WeekSale struct {
	Day   string `json:"day"`
	Sales uint   `json:"sales"`
}

type ProductSale struct {
	Product string `json:"product"`
	Sales   uint   `json:"sales"`
}

type BranchSale struct {
	Branch string `json:"branch"`
	Sales  uint   `json:"sales"`
}

type MonthSales struct {
	Month string `json:"month"`
	Sales uint   `json:"sales"`
}

type MiniReport struct {
	Stock uint          `json:"stock"`
	Sales product.Price `json:"sales"`
}

type HomeAnalytics struct {
	Reports      MiniReport    `json:"reports"`
	WeekSales    []WeekSale    `json:"weekSales"`
	ProductSales []ProductSale `json:"productSales"`
	BranchSales  []BranchSale  `json:"branchSales"`
	MonthSales   []MonthSales  `json:"monthSales"`
}

type ReportsSummary struct {
	TotalStockValue product.Price `json:"totalStockValue"`
	TotalStock      uint          `json:"totalStock"`
	SalesToday      product.Price `json:"salesToday"`
	SalesThisMonth  product.Price `json:"salesThisMonth"`
	LowStockAlerts  uint          `json:"lowStockAlerts"`
}
