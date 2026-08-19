package product

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func NewService(db *gorm.DB) *ProductService {
	return &ProductService{DB: db}
}

// deriveAvailability computes stock status from quantity since it isn't persisted.
// Threshold is a placeholder — confirm the real low-stock cutoff.
func deriveAvailability(qty uint) ProductAvailability {
	if qty == 0 {
		return Out
	}
	if qty < 5 {
		return Low
	}
	return ""
}

// FindAll returns products scoped to the branches the requesting user belongs to,
// with Availability and Ranking populated (both are gorm:"-", computed on read).
func (s *ProductService) FindAll(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	var branchIDs []uint
	if err := s.DB.Table("branches").
		Joins("JOIN branch_members ON branch_members.branch_id = branches.id").
		Where("branch_members.user_id = ?", UserID).
		Pluck("branches.id", &branchIDs).Error; err != nil {
		return base.API_ERROR(c, "failed to resolve branches")
	}

	query := s.DB.Model(&Product{})
	if len(branchIDs) > 0 {
		query = query.Where("branch_id IN ?", branchIDs)
	} else {
		query = query.Where("1 = 0") // no branches → no products
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	var products []Product
	if err := query.Find(&products).Error; err != nil {
		return base.API_ERROR(c, "failed to get products")
	}

	// rank by quantity, descending — highest stock is "1st"
	sorted := make([]Product, len(products))
	copy(sorted, products)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Quantity.Data().Value > sorted[i].Quantity.Data().Value {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	rankOf := make(map[uint]int, len(sorted))
	for i, p := range sorted {
		rankOf[p.ID] = i + 1
	}

	availability := c.Query("availability") // "", "low", "out"

	result := make([]Product, 0, len(products))
	for _, p := range products {
		qty := p.Quantity.Data().Value
		p.Availability = deriveAvailability(qty)
		p.Ranking = ordinal(rankOf[p.ID])

		if availability != "" && string(p.Availability) != availability {
			continue
		}
		result = append(result, p)
	}

	return c.JSON(base.APIResponse{Data: result})
}

func ordinal(n int) string {
	if n <= 0 {
		return ""
	}
	suffix := "th"
	switch n % 100 {
	case 11, 12, 13:
	default:
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return itoa(n) + suffix
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func (s *ProductService) Create(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	var payload Product
	if err := c.BodyParser(&payload); err != nil {
		return base.API_ERROR(c, "invalid payload")
	}

	if payload.Name == "" {
		return base.API_ERROR(c, "product name is required")
	}
	if payload.BranchID == nil {
		return base.API_ERROR(c, "branch is required")
	}

	// verify the user actually belongs to the branch they're posting into
	var isMember int64
	if err := s.DB.Table("branch_members").
		Where("branch_id = ? AND user_id = ?", *payload.BranchID, UserID).
		Count(&isMember).Error; err != nil {
		return base.API_ERROR(c, "failed to verify branch access")
	}
	if isMember == 0 {
		return base.API_ERROR(c, "branch not found")
	}

	if err := s.DB.Create(&payload).Error; err != nil {
		return base.API_ERROR(c, "failed to create product")
	}

	return c.JSON(base.APIResponse{Data: payload})
}

func (s *ProductService) Update(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	param := c.Params("id")
	id, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid product id")
	}

	var existing Product
	if err := s.DB.First(&existing, uint(id)).Error; err != nil {
		return base.API_ERROR(c, "product not found")
	}

	// verify the user belongs to the branch that currently owns this product
	if existing.BranchID != nil {
		var isMember int64
		if err := s.DB.Table("branch_members").
			Where("branch_id = ? AND user_id = ?", *existing.BranchID, UserID).
			Count(&isMember).Error; err != nil {
			return base.API_ERROR(c, "failed to verify branch access")
		}
		if isMember == 0 {
			return base.API_ERROR(c, "product not found")
		}
	}

	var payload Product
	if err := c.BodyParser(&payload); err != nil {
		return base.API_ERROR(c, "invalid payload")
	}

	if payload.Name == "" {
		return base.API_ERROR(c, "product name is required")
	}
	if payload.BranchID == nil {
		return base.API_ERROR(c, "branch is required")
	}

	// if the branch is being changed, verify the user belongs to the new one too
	if *payload.BranchID != *existing.BranchID {
		var isMember int64
		if err := s.DB.Table("branch_members").
			Where("branch_id = ? AND user_id = ?", *payload.BranchID, UserID).
			Count(&isMember).Error; err != nil {
			return base.API_ERROR(c, "failed to verify branch access")
		}
		if isMember == 0 {
			return base.API_ERROR(c, "branch not found")
		}
	}

	existing.Name = payload.Name
	existing.Photo = payload.Photo
	existing.Price = payload.Price
	existing.Quantity = payload.Quantity
	existing.Features = payload.Features
	existing.BranchID = payload.BranchID

	if err := s.DB.Save(&existing).Error; err != nil {
		return base.API_ERROR(c, "failed to update product")
	}

	return c.JSON(base.APIResponse{Data: existing})
}

func (s *ProductService) AdjustStock(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	param := c.Params("id")
	id, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid product id")
	}

	var existing Product
	if err := s.DB.First(&existing, uint(id)).Error; err != nil {
		return base.API_ERROR(c, "product not found")
	}

	if existing.BranchID != nil {
		var isMember int64
		if err := s.DB.Table("branch_members").
			Where("branch_id = ? AND user_id = ?", *existing.BranchID, UserID).
			Count(&isMember).Error; err != nil {
			return base.API_ERROR(c, "failed to verify branch access")
		}
		if isMember == 0 {
			return base.API_ERROR(c, "you are trying to access data that is not yours")
		}
	}

	var payload AdjustStockPayload
	if err := c.BodyParser(&payload); err != nil {
		return base.API_ERROR(c, "invalid payload")
	}
	if payload.Quantity == 0 {
		return base.API_ERROR(c, "quantity must be greater than zero")
	}

	current := existing.Quantity.Data()
	current.Value += payload.Quantity
	existing.Quantity = datatypes.NewJSONType(current)

	if err := s.DB.Save(&existing).Error; err != nil {
		return base.API_ERROR(c, "failed to update stock")
	}

	return c.JSON(base.APIResponse{Data: existing})
}
