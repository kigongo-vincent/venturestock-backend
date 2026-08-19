package investment

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/product"
	"github.com/kigongo-vincent/venturestock-backend/modules/sale"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func NewInvestmentService(db *gorm.DB) *InvestmentService {
	return &InvestmentService{DB: db}
}

func (s *InvestmentService) Create(c *fiber.Ctx) error {
	userID, err := base.GetUserIDFromCtx(c)
	if err != nil {
		return base.API_ERROR(c, "user not found")
	}

	var input CreateInvestmentInput

	if err := c.BodyParser(&input); err != nil {
		return base.API_ERROR(c, "Invalid request body")
	}

	inputPrice := input.Amount.Data()

	if inputPrice.Value <= 0 {
		return base.API_ERROR(c, "Investment amount must be greater than zero")
	}

	if input.Type == "topup" {
		var existing CapitalInvestment

		if err := s.DB.
			Where("user_id = ? AND end = ?", userID, time.Time{}).
			Order("created_at DESC").
			First(&existing).Error; err != nil {
			return base.API_ERROR(c, "No running investment found to top up")
		}

		existingPrice := existing.Amount.Data()

		currency := inputPrice.Currency
		if currency == "" {
			currency = existingPrice.Currency
		}

		existing.Amount = datatypes.NewJSONType(product.Price{
			Value:    existingPrice.Value + inputPrice.Value,
			Currency: currency,
		})

		if input.Note != "" {
			if existing.Note != "" {
				existing.Note += " | "
			}
			existing.Note += input.Note
		}

		if err := s.DB.Save(&existing).Error; err != nil {
			return base.API_ERROR(c, "Failed to apply top-up: "+err.Error())
		}

		return c.Status(fiber.StatusOK).JSON(existing)
	}

	var running CapitalInvestment

	if err := s.DB.
		Where("user_id = ? AND end = ?", userID, time.Time{}).
		First(&running).Error; err == nil {

		running.End = time.Now()

		if err := s.DB.Save(&running).Error; err != nil {
			return base.API_ERROR(c, "Failed to close previous investment: "+err.Error())
		}
	}

	investment := CapitalInvestment{
		Amount: input.Amount,
		Note:   input.Note,
		Type:   "initial",
		UserID: &userID,
	}

	if err := s.DB.Create(&investment).Error; err != nil {
		return base.API_ERROR(c, "Failed to record investment: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(investment)
}

func (s *InvestmentService) FindAll(c *fiber.Ctx) error {
	userID, err := base.GetUserIDFromCtx(c)
	if err != nil {
		return base.API_ERROR(c, "user not found")
	}

	var investments []CapitalInvestment

	if err := s.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&investments).Error; err != nil {
		return base.API_ERROR(c, "Failed to fetch investments: "+err.Error())
	}

	return c.JSON(base.APIResponse{
		Data: investments,
	})
}

func (s *InvestmentService) Delete(c *fiber.Ctx) error {
	userID, err := base.GetUserIDFromCtx(c)
	if err != nil {
		return base.API_ERROR(c, "user not found")
	}

	id := c.Params("id")

	if id == "" {
		return base.API_ERROR(c, "investment id is required")
	}

	var investment CapitalInvestment

	if err := s.DB.
		Where("id = ? AND user_id = ?", id, userID).
		First(&investment).Error; err != nil {
		return base.API_ERROR(c, "investment record not found")
	}

	if err := s.DB.Delete(&investment).Error; err != nil {
		return base.API_ERROR(c, "failed to delete investment: "+err.Error())
	}

	return c.JSON(base.APIResponse{
		Message: "investment deleted successfully",
	})
}

func (s *InvestmentService) Comparison(c *fiber.Ctx) error {
	userID, err := base.GetUserIDFromCtx(c)
	if err != nil {
		return base.API_ERROR(c, "user not found")
	}

	id := c.Params("id")

	if id == "" {
		return base.API_ERROR(c, "investment id is required")
	}

	var investment CapitalInvestment

	if err := s.DB.
		Where("id = ? AND user_id = ?", id, userID).
		First(&investment).Error; err != nil {
		return base.API_ERROR(c, "investment not found")
	}

	start := investment.CreatedAt
	end := time.Now()

	if !investment.End.IsZero() {
		end = investment.End
	}

	var sales []sale.Sale

	if err := s.DB.
		Where(
			"seller_id = ? AND created_at >= ? AND created_at < ?",
			userID,
			start,
			end,
		).
		Order("created_at ASC").
		Find(&sales).Error; err != nil {
		return base.API_ERROR(c, "failed to fetch sales: "+err.Error())
	}

	investmentAmount := float64(investment.Amount.Data().Value)

	var totalSales float64

	points := []InvestmentComparisonPoint{
		{
			Date:            investment.CreatedAt,
			Investment:      investmentAmount,
			Sales:           0,
			EstimatedProfit: -investmentAmount,
		},
	}

	for _, currentSale := range sales {
		totalSales += float64(currentSale.Amount.Data().Value)

		points = append(points, InvestmentComparisonPoint{
			Date:            currentSale.CreatedAt,
			Investment:      investmentAmount,
			Sales:           totalSales,
			EstimatedProfit: totalSales - investmentAmount,
		})
	}

	performance := float64(0)

	if investmentAmount > 0 {
		performance = (totalSales / investmentAmount) * 100
	}

	var investmentEnd *time.Time

	if !investment.End.IsZero() {
		investmentEnd = &investment.End
	}

	return c.JSON(base.APIResponse{
		Data: InvestmentComparisonResponse{
			InvestmentID: investment.ID,
			Start:        investment.CreatedAt,
			End:          investmentEnd,
			Amount:       investmentAmount,
			TotalSales:   totalSales,
			Performance:  performance,
			Points:       points,
		},
	})
}
