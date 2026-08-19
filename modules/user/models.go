package user

import (
	"time"

	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/gorm"
)

type UserService struct {
	DB *gorm.DB
}

type CreateWorkerPayload struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	BranchID uint   `json:"branchId"`
}

type UpdateAccountRequest struct {
	Photo string `json:"photo"`
	Name  string `json:"name"`
}

type UpdatePhoneRequest struct {
	Phone string `json:"phone"`
}

type RequestOTPRequest struct {
	Email string `json:"email"`
}

type UpdateEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type OTP struct {
	gorm.Model
	UserID     *uint        `json:"userId"`
	User       *shared.User `json:"user" gorm:"foreignKey:UserID"`
	Code       string       `json:"code"`
	ExpiryDate time.Time    `json:"expiryDate"`
}
