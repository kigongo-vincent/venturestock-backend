package middleware

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/gorm"
)

func VerifyJWT(databases ...*gorm.DB) fiber.Handler {

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

		if len(databases) > 0 && databases[0] != nil {
			userID, idErr := base.GetUserIDFromCtx(c)
			if idErr != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(base.APIResponse{Message: "authentication failed"})
			}

			locked, lockErr := subscriptionLocked(databases[0], userID)
			if lockErr != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(base.APIResponse{Message: "failed to verify business subscription"})
			}
			if locked {
				return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
					"data":   nil,
					"msg":    "your business subscription has expired. ask the owner to renew it.",
					"status": fiber.StatusPaymentRequired,
					"code":   "subscription_expired",
				})
			}
		}

		return c.Next()

	}

}

func subscriptionLocked(db *gorm.DB, userID uint) (bool, error) {
	var businessID uint

	var ownedIDs []uint
	if err := db.Model(&shared.Business{}).
		Where("owner_id = ?", userID).
		Limit(1).
		Pluck("id", &ownedIDs).Error; err != nil {
		return false, err
	}
	if len(ownedIDs) > 0 {
		businessID = ownedIDs[0]
	}

	if businessID == 0 {
		var memberBusinessIDs []uint
		if err := db.Table("branches").
			Select("branches.business_id").
			Joins("JOIN branch_members ON branch_members.branch_id = branches.id").
			Where("branch_members.user_id = ?", userID).
			Where("branches.deleted_at IS NULL").
			Limit(1).
			Pluck("branches.business_id", &memberBusinessIDs).Error; err != nil {
			return false, err
		}
		if len(memberBusinessIDs) > 0 {
			businessID = memberBusinessIDs[0]
		}
	}

	if businessID == 0 {
		return true, nil
	}

	var count int64
	if err := db.Table("business_subscriptions").
		Where("business_id = ?", businessID).
		Where("status = ?", "active").
		Where("deleted_at IS NULL").
		Where("expires_at > ?", time.Now()).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count == 0, nil
}
