package sale

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/product"
	"github.com/kigongo-vincent/venturestock-backend/modules/user"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	Paid   PaymentStatus = "paid"
	UnPaid PaymentStatus = "unpaid"
)

type Sale struct {
	gorm.Model
	Quantity    datatypes.JSONType[product.Quantity] `json:"quantity"`
	Amount      datatypes.JSONType[product.Price]    `json:"amount"`
	ExtraAmount datatypes.JSONType[product.Price]    `json:"extraAmount"`
	Buyer       datatypes.JSONType[user.User]        `json:"buyer"`
	SellerID    *uint                                `json:"sellerId"`
	Seller      *user.User                           `json:"seller" gorm:"foreignKey:SellerID"`
	Status      PaymentStatus                        `json:"status" gorm:"default:paid"`
	ProductID   *uint                                `json:"productId"`
	Product     *product.Product                     `json:"product" gorm:"foreignKey:ProductID"`
}
