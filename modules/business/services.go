package business

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/gorm"
)

func NewService(db *gorm.DB) *BusinessService {
	return &BusinessService{DB: db}
}

func (s *BusinessService) Create(business shared.Business) (uint, error) {

	err := s.DB.Create(&business).Error
	if err != nil {
		return 0, err
	}

	return business.ID, nil

}

func (s *BusinessService) Update(c *fiber.Ctx) error {

	var body UpdateBusinessRequest
	if err := c.BodyParser(&body); err != nil {
		return base.API_ERROR(c, "failed to parse input")
	}

	var business shared.Business
	if err := s.DB.First(&business, body.ID).Error; err != nil {
		return base.API_ERROR(c, "failed to get business")
	}

	if body.Name != "" {
		business.Name = body.Name
	}

	if body.Location != "" {
		business.Location = body.Location
	}

	if err := s.DB.Save(&business).Error; err != nil {
		return base.API_ERROR(c, "failed to save business")
	}

	return c.SendStatus(200)

}
