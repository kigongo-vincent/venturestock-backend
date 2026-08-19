package test

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/auth"
)

func TestSignUp(t *testing.T) {
	t.Skip()
	HTTP[auth.LoginResponse, auth.SignupRequest](HTTPRequest[auth.LoginResponse, auth.SignupRequest]{Method: fiber.MethodPost, Path: "auth/signup", Body: auth.SignupRequest{
		Name:             "vincent",
		Email:            "vincent6@gmail.com",
		Phone:            "0745253954",
		Password:         "1234",
		BusinessName:     "Tora Dynamics",
		BusinessLocation: "Kampala Uganda",
	}})
	t.Error()
}

func TestLogin(t *testing.T) {
	t.Skip()
	HTTP[auth.LoginResponse, auth.LoginRequest](HTTPRequest[auth.LoginResponse, auth.LoginRequest]{Method: fiber.MethodPost, Path: "auth/login", Body: auth.LoginRequest{
		Password:   "1234",
		Identifier: "0745253954",
	}})
	t.Error()
}
