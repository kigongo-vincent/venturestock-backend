package sale

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/product"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	Paid   PaymentStatus = "paid"
	UnPaid PaymentStatus = "unpaid"
)

type CapitalInvestment struct {
	gorm.Model
	Amount     float64         `gorm:"not null" json:"amount"`
	BusinessID *uint           `json:"businessId" `
	Business   shared.Business `json:"business" gorm:"foreignKey:BusinessID"`
	Note       string          `json:"note"`
}

type Sale struct {
	gorm.Model
	Quantity    datatypes.JSONType[product.Quantity] `json:"quantity"`
	Amount      datatypes.JSONType[product.Price]    `json:"amount"`
	ExtraAmount datatypes.JSONType[product.Price]    `json:"extraAmount"`
	Buyer       datatypes.JSONType[shared.User]      `json:"buyer"`
	SellerID    *uint                                `json:"sellerId"`
	Seller      *shared.User                         `json:"seller" gorm:"foreignKey:SellerID"`
	Status      PaymentStatus                        `json:"status" gorm:"default:paid"`
	ProductID   *uint                                `json:"productId"`
	Product     *product.Product                     `json:"product" gorm:"foreignKey:ProductID"`
}

type SaleService struct {
	DB *gorm.DB
}

type CreateSalePayload struct {
	ProductID   uint             `json:"productId"`
	Quantity    product.Quantity `json:"quantity"`
	Amount      product.Price    `json:"amount"`
	ExtraAmount product.Price    `json:"extraAmount"`
	Status      PaymentStatus    `json:"status"`
	Buyer       Buyer            `json:"buyer"`
}

type Buyer struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}
