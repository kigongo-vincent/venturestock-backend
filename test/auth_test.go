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
		Phone:            "0782147146",
		Password:         "1234",
		Photo:            "https://s3.com/me.png",
		BusinessName:     "Tora Dynamics",
		BusinessLocation: "Kampala Uganda",
	}})
	t.Error()
}

func TestLogin(t *testing.T) {
	t.Skip()
	HTTP[auth.LoginResponse, auth.LoginRequest](HTTPRequest[auth.LoginResponse, auth.LoginRequest]{Method: fiber.MethodPost, Path: "auth/login", Body: auth.LoginRequest{
		Password:   "1234",
		Identifier: "vincent4@gmail.com",
	}})
	t.Error()
}
