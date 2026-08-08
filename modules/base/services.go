package base

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func API_ERROR(c *fiber.Ctx, v string) error {
	return c.Status(400).JSON(APIResponse{Message: v})
}

func SetupApplication() *fiber.App {
	godotenv.Load()
	app := fiber.New()
	app.Use(limiter.New(limiter.Config{Max: 100, Expiration: 60 * time.Second}))

	return app
}

// FilteredPagination unwraps the PaginationControls from the APIRequest
// and hands it off to the base Paginate scopes function.
func FilteredPagination(r *APIRequest) func(db *gorm.DB) *gorm.DB {
	if r == nil {
		// Return an empty scope modifier if the request payload is nil
		return func(db *gorm.DB) *gorm.DB { return db }
	}
	return Paginate(&r.Pagination)
}

// Paginate provides the reusable GORM Scope logic for calculating offset bounds.
func Paginate(p *PaginationControls) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if p == nil {
			return db
		}
		if p.Limit == 0 {
			p.Limit = 10
		}
		if p.Page == 0 {
			p.Page = 1
		}

		offset := (p.Page - 1) * p.Limit
		return db.Offset(int(offset)).Limit(int(p.Limit))
	}
}
