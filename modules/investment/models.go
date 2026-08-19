package investment

import (
	"time"

	"github.com/kigongo-vincent/venturestock-backend/modules/product"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type InvestmentService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *InvestmentService {
	return &InvestmentService{DB: db}
}

type CapitalInvestment struct {
	gorm.Model
	Amount datatypes.JSONType[product.Price] `json:"amount"`
	Note   string                            `json:"note"`
	UserID *uint                             `json:"userId"`
	User   *shared.User                      `json:"user" gorm:"foreignKey:UserID"`
	End    time.Time                         `json:"end"`
	Type   string                            `json:"type"`
}

type CreateInvestmentInput struct {
	Amount datatypes.JSONType[product.Price] `json:"amount"`
	Note   string                            `json:"note"`
	Type   string                            `json:"type"`
}

type InvestmentComparisonPoint struct {
	Date            time.Time `json:"date"`
	Investment      float64   `json:"investment"`
	Sales           float64   `json:"sales"`
	EstimatedProfit float64   `json:"estimatedProfit"`
}

type InvestmentComparisonResponse struct {
	InvestmentID uint                        `json:"investmentId"`
	Start        time.Time                   `json:"start"`
	End          *time.Time                  `json:"end,omitempty"`
	Amount       float64                     `json:"amount"`
	TotalSales   float64                     `json:"totalSales"`
	Performance  float64                     `json:"performance"`
	Points       []InvestmentComparisonPoint `json:"points"`
}
