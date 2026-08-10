package auth

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/user"
	"gorm.io/gorm"
)

type AuthService struct {
	DB *gorm.DB
}

type SignupRequest struct {
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	Name             string `json:"name"`
	Password         string `json:"password"`
	Photo            string `json:"photo"`
	BusinessName     string `json:"businessName"`
	BusinessLocation string `json:"businessLocation"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type LoginResponse struct {
	Token string    `json:"token"`
	User  user.User `json:"user"`
}
