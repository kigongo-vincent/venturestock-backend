package subscription

import (
	"time"

	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	StatusActive  = "active"
	StatusExpired = "expired"
	StatusPending = "pending"
	StatusSuccess = "successful"
	StatusFailed  = "failed"

	FreePackageSlug = "free"
	ProPackageSlug  = "pro"
)

type Package struct {
	gorm.Model
	Slug            string                       `json:"slug" gorm:"size:40;uniqueIndex;not null"`
	Name            string                       `json:"name" gorm:"not null"`
	Description     string                       `json:"description"`
	Amount          float64                      `json:"amount" gorm:"type:numeric(12,2);not null"`
	Currency        string                       `json:"currency" gorm:"size:3;not null"`
	BillingInterval string                       `json:"billingInterval" gorm:"size:20;not null"`
	IntervalCount   uint                         `json:"intervalCount" gorm:"not null;default:1"`
	Features        datatypes.JSONType[[]string] `json:"features"`
	SortOrder       uint                         `json:"sortOrder" gorm:"default:0"`
	IsActive        bool                         `json:"isActive" gorm:"default:true"`
}

func (Package) TableName() string {
	return "subscription_packages"
}

func (p Package) ExpiresFrom(start time.Time) time.Time {
	count := int(p.IntervalCount)
	if count < 1 {
		count = 1
	}

	switch p.BillingInterval {
	case "hour":
		return start.Add(time.Duration(count) * time.Hour)
	case "day":
		return start.AddDate(0, 0, count)
	case "week":
		return start.AddDate(0, 0, count*7)
	case "year":
		return start.AddDate(count, 0, 0)
	default:
		return start.AddDate(0, count, 0)
	}
}

type Subscription struct {
	gorm.Model
	BusinessID uint             `json:"businessId" gorm:"uniqueIndex;not null"`
	Business   *shared.Business `json:"business,omitempty" gorm:"foreignKey:BusinessID"`
	PackageID  uint             `json:"packageId" gorm:"not null"`
	Package    *Package         `json:"package,omitempty" gorm:"foreignKey:PackageID"`
	Status     string           `json:"status" gorm:"size:20;not null;index"`
	StartsAt   time.Time        `json:"startsAt" gorm:"not null"`
	ExpiresAt  time.Time        `json:"expiresAt" gorm:"not null;index"`
	AutoRenew  bool             `json:"autoRenew" gorm:"default:false"`
}

func (Subscription) TableName() string {
	return "business_subscriptions"
}

type Payment struct {
	gorm.Model
	BusinessID    uint             `json:"businessId" gorm:"not null;index"`
	Business      *shared.Business `json:"business,omitempty" gorm:"foreignKey:BusinessID"`
	PackageID     uint             `json:"packageId" gorm:"not null"`
	Package       *Package         `json:"package,omitempty" gorm:"foreignKey:PackageID"`
	TxRef         string           `json:"txRef" gorm:"size:100;uniqueIndex;not null"`
	FlutterwaveID *int64           `json:"flutterwaveId"`
	Amount        float64          `json:"amount" gorm:"type:numeric(12,2);not null"`
	Currency      string           `json:"currency" gorm:"size:3;not null"`
	Status        string           `json:"status" gorm:"size:20;not null;index"`
	CheckoutURL   string           `json:"checkoutUrl" gorm:"type:text"`
	PaidAt        *time.Time       `json:"paidAt"`
	RawResponse   string           `json:"-" gorm:"type:text"`
}

func (Payment) TableName() string {
	return "subscription_payments"
}

type CheckoutRequest struct {
	PackageID uint `json:"packageId"`
}

type CheckoutResponse struct {
	CheckoutURL string `json:"checkoutUrl"`
	TxRef       string `json:"txRef"`
	PaymentID   uint   `json:"paymentId"`
}

type VerifyRequest struct {
	TxRef string `json:"txRef"`
}

type Overview struct {
	Subscription *shared.SubscriptionSnapshot `json:"subscription"`
	Packages     []Package                    `json:"packages"`
	Payments     []Payment                    `json:"payments"`
}

type WebhookPayload struct {
	Event string `json:"event"`
	Type  string `json:"type"`
	Data  struct {
		ID       int64   `json:"id"`
		TxRef    string  `json:"tx_ref"`
		Status   string  `json:"status"`
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	} `json:"data"`
}
