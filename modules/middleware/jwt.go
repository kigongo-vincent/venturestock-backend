package middleware

import (
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
)

func VerifyJWT() fiber.Handler {

	return func(c *fiber.Ctx) error {

		// get token from request
		tokenString := c.Get("Authorization")
		if tokenString == "" {
			return c.Status(400).JSON(base.APIResponse{Message: "missing token"})
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		secret := []byte(os.Getenv("JWT_SECRET"))
		// verify jwt
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
			return secret, nil
		})
		if err != nil {
			fmt.Println(err)
			return c.Status(403).JSON(base.APIResponse{Message: "authentication failed"})
		}
		if !token.Valid {
			return c.Status(401).JSON(base.APIResponse{Message: "authentication failed"})
		}

		claims := token.Claims.(jwt.MapClaims)

		// get the user role and user id
		role, ok := claims["role"]
		ok = true
		if !ok {
			return c.Status(400).JSON(base.APIResponse{Message: "authentication failed"})
		}
		id, ok := claims["id"]
		if !ok {
			return c.Status(400).JSON(base.APIResponse{Message: "authentication failed"})
		}

		c.Locals("id", id)
		c.Locals("role", role)

		return c.Next()

	}

}
