package sale

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/product"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func NewService(db *gorm.DB) *SaleService {
	return &SaleService{DB: db}
}

func (s *SaleService) FindAll(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	branchIDs, bErr := shared.GetUserBranchIDs(s.DB, UserID)
	if bErr != nil {
		return base.API_ERROR(c, "failed to resolve branches")
	}

	// narrow to a single branch if requested — must still be one of the
	// user's own branches, never trust the param on its own
	if branchIDParam := c.Query("branchId"); branchIDParam != "" {
		requested, pErr := strconv.ParseUint(branchIDParam, 10, 64)
		if pErr != nil {
			return base.API_ERROR(c, "invalid branch id")
		}
		allowed := false
		for _, id := range branchIDs {
			if id == uint(requested) {
				allowed = true
				break
			}
		}
		if !allowed {
			return base.API_ERROR(c, "branch not found")
		}
		branchIDs = []uint{uint(requested)}
	}

	var productIDs []uint
	if len(branchIDs) > 0 {
		if err := s.DB.Table("products").
			Where("branch_id IN ?", branchIDs).
			Where("deleted_at IS NULL").
			Pluck("id", &productIDs).Error; err != nil {
			return base.API_ERROR(c, "failed to resolve products")
		}
	}

	query := s.DB.Model(&Sale{}).Preload("Seller").Preload("Product")
	if len(productIDs) > 0 {
		query = query.Where("product_id IN ?", productIDs)
	} else {
		query = query.Where("1 = 0")
	}

	now := time.Now()
	switch strings.ToLower(strings.TrimSpace(c.Query("range"))) {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		query = query.Where("created_at >= ?", start)
	case "this week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).
			AddDate(0, 0, -(weekday - 1))
		query = query.Where("created_at >= ?", start)
	case "this month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		query = query.Where("created_at >= ?", start)
	case "this year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		query = query.Where("created_at >= ?", start)
	case "promised payments":
		query = query.Where("status = ?", UnPaid)
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		query = query.Joins("JOIN products ON products.id = sales.product_id").
			Where("products.name ILIKE ?", "%"+search+"%")
	}

	var sales []Sale
	if err := query.Order("created_at DESC").Find(&sales).Error; err != nil {
		return base.API_ERROR(c, "failed to get sales")
	}

	return c.JSON(base.APIResponse{Data: sales})
}

func (s *SaleService) Create(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	var payload CreateSalePayload
	if err := c.BodyParser(&payload); err != nil {
		return base.API_ERROR(c, "invalid payload")
	}

	if payload.ProductID == 0 {
		return base.API_ERROR(c, "product is required")
	}
	if payload.Quantity.Value == 0 {
		return base.API_ERROR(c, "quantity must be greater than zero")
	}
	if payload.Amount.Value == 0 {
		return base.API_ERROR(c, "amount is required")
	}

	var prod product.Product
	if err := s.DB.First(&prod, payload.ProductID).Error; err != nil {
		return base.API_ERROR(c, "product not found")
	}

	if prod.BranchID != nil {
		allowed, aErr := shared.UserHasBranchAccess(s.DB, UserID, *prod.BranchID)
		if aErr != nil {
			return base.API_ERROR(c, "failed to verify branch access")
		}
		if !allowed {
			return base.API_ERROR(c, "product not found")
		}
	}

	currentQty := prod.Quantity.Data()
	if payload.Quantity.Value > currentQty.Value {
		return base.API_ERROR(c, "not enough stock for this sale")
	}

	status := payload.Status
	if status == "" {
		status = Paid
	}

	newSale := Sale{
		Quantity:    datatypes.NewJSONType(payload.Quantity),
		Amount:      datatypes.NewJSONType(payload.Amount),
		ExtraAmount: datatypes.NewJSONType(payload.ExtraAmount),
		Buyer:       datatypes.NewJSONType(shared.User{Name: payload.Buyer.Name, Phone: payload.Buyer.Phone}),
		SellerID:    &UserID,
		Status:      status,
		ProductID:   &payload.ProductID,
	}

	tx := s.DB.Begin()

	if err := tx.Create(&newSale).Error; err != nil {
		tx.Rollback()
		return base.API_ERROR(c, "failed to record sale")
	}

	currentQty.Value -= payload.Quantity.Value
	prod.Quantity = datatypes.NewJSONType(currentQty)
	if err := tx.Save(&prod).Error; err != nil {
		tx.Rollback()
		return base.API_ERROR(c, "failed to update stock")
	}

	if err := tx.Commit().Error; err != nil {
		return base.API_ERROR(c, "failed to record sale")
	}

	return c.JSON(base.APIResponse{Data: newSale})
}

func (s *SaleService) Update(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	param := c.Params("id")
	id, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid sale id")
	}

	var existing Sale
	if err := s.DB.First(&existing, uint(id)).Error; err != nil {
		return base.API_ERROR(c, "sale not found")
	}

	if existing.SellerID == nil || *existing.SellerID != UserID {
		return base.API_ERROR(c, "sale not found")
	}

	var payload CreateSalePayload
	if err := c.BodyParser(&payload); err != nil {
		return base.API_ERROR(c, "invalid payload")
	}

	if payload.Amount.Value == 0 {
		return base.API_ERROR(c, "amount is required")
	}

	existing.Amount = datatypes.NewJSONType(payload.Amount)
	existing.ExtraAmount = datatypes.NewJSONType(payload.ExtraAmount)
	existing.Buyer = datatypes.NewJSONType(shared.User{Name: payload.Buyer.Name, Phone: payload.Buyer.Phone})
	if payload.Status != "" {
		existing.Status = payload.Status
	}

	if err := s.DB.Save(&existing).Error; err != nil {
		return base.API_ERROR(c, "failed to update sale")
	}

	return c.JSON(base.APIResponse{Data: existing})
}

func (s *SaleService) FindBySeller(c *fiber.Ctx) error {

	RequestingUserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	param := c.Params("userId")
	sellerID, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid user id")
	}

	if uint(sellerID) != RequestingUserID {
		allowed, aErr := shared.UsersShareBranch(s.DB, RequestingUserID, uint(sellerID))
		if aErr != nil {
			return base.API_ERROR(c, "failed to verify access")
		}
		if !allowed {
			return base.API_ERROR(c, "user not found")
		}
	}

	var sales []Sale
	if err := s.DB.Preload("Product").
		Where("seller_id = ?", uint(sellerID)).
		Order("created_at DESC").
		Find(&sales).Error; err != nil {
		return base.API_ERROR(c, "failed to get sales")
	}

	return c.JSON(base.APIResponse{Data: sales})
}
