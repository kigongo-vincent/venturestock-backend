package branch

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/product"
	"github.com/kigongo-vincent/venturestock-backend/modules/sale"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/gorm"
)

func NewService(db *gorm.DB) *BranchService {
	return &BranchService{DB: db}
}

func (s *BranchService) Create(branch product.Branch) (uint, error) {
	if err := s.DB.Create(&branch).Error; err != nil {
		return 0, err
	}
	return branch.ID, nil
}

func (s *BranchService) CreateHandler(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}
	if err := base.DenyWorker(c, "workers cannot add branches"); err != nil {
		return err
	}

	var payload CreateBranchPayload
	if err := c.BodyParser(&payload); err != nil {
		return base.API_ERROR(c, "invalid payload")
	}
	if payload.Name == "" {
		return base.API_ERROR(c, "branch name is required")
	}

	var biz shared.Business
	if err := s.DB.Where("owner_id = ?", UserID).First(&biz).Error; err != nil {
		return base.API_ERROR(c, "no business found for this account")
	}

	newBranch := product.Branch{
		Name:       payload.Name,
		BusinessID: &biz.ID,
	}

	if err := s.DB.Create(&newBranch).Error; err != nil {
		return base.API_ERROR(c, "failed to create branch")
	}

	return c.JSON(base.APIResponse{Data: newBranch})
}

// FindAll returns branches under the requesting user's business, with
// workers (member count) and revenue (all-time sum of sales on that
// branch's products) computed at read time.
func (s *BranchService) FindAll(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	// resolve the business this user's branches belong to via their own
	// branch memberships, then list every branch under that business
	var businessIDs []uint
	if err := s.DB.Table("branches").
		Joins("JOIN branch_members ON branch_members.branch_id = branches.id").
		Where("branch_members.user_id = ?", UserID).
		Pluck("DISTINCT branches.business_id", &businessIDs).Error; err != nil {
		return base.API_ERROR(c, "failed to resolve business")
	}

	var branches []product.Branch
	query := s.DB.Preload("Members")
	if base.IsWorker(c) {
		branchIDs, bErr := shared.GetUserBranchIDs(s.DB, UserID)
		if bErr != nil {
			return base.API_ERROR(c, "failed to resolve branches")
		}
		if len(branchIDs) > 0 {
			query = query.Where("id IN ?", branchIDs)
		} else {
			query = query.Where("1 = 0")
		}
	} else if len(businessIDs) > 0 {
		query = query.Where("business_id IN ?", businessIDs)
	} else {
		query = query.Where("1 = 0")
	}
	if err := query.Find(&branches).Error; err != nil {
		return base.API_ERROR(c, "failed to get branches")
	}

	result := make([]BranchSummary, 0, len(branches))
	for _, b := range branches {

		var products []product.Product
		if err := s.DB.Where("branch_id = ?", b.ID).Find(&products).Error; err != nil {
			return base.API_ERROR(c, "failed to get branch products")
		}

		productIDs := make([]uint, 0, len(products))
		for _, p := range products {
			productIDs = append(productIDs, p.ID)
		}

		var revenue uint
		currency := product.UGX
		if len(productIDs) > 0 {
			var sales []sale.Sale
			if err := s.DB.Where("product_id IN ?", productIDs).Find(&sales).Error; err != nil {
				return base.API_ERROR(c, "failed to get branch sales")
			}
			for _, sl := range sales {
				amt := sl.Amount.Data()
				revenue += amt.Value
				currency = amt.Currency
			}
		}

		result = append(result, BranchSummary{
			ID:      b.ID,
			Name:    b.Name,
			Workers: uint(len(b.Members)),
			Revenue: product.Price{Currency: currency, Value: revenue},
		})
	}

	return c.JSON(base.APIResponse{Data: result})
}

func (s *BranchService) FindOne(c *fiber.Ctx) error {

	param := c.Params("id")
	id, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid branch id")
	}

	var b product.Branch
	if err := s.DB.Preload("Members").First(&b, uint(id)).Error; err != nil {
		return base.API_ERROR(c, "branch not found")
	}
	if err := s.assertWorkerBranchAccess(c, b.ID); err != nil {
		return err
	}

	var products []product.Product
	if err := s.DB.Where("branch_id = ?", b.ID).Find(&products).Error; err != nil {
		return base.API_ERROR(c, "failed to get branch products")
	}

	productIDs := make([]uint, 0, len(products))
	for _, p := range products {
		productIDs = append(productIDs, p.ID)
	}

	var revenue uint
	currency := product.UGX
	if len(productIDs) > 0 {
		var sales []sale.Sale
		if err := s.DB.Where("product_id IN ?", productIDs).Find(&sales).Error; err != nil {
			return base.API_ERROR(c, "failed to get branch sales")
		}
		for _, sl := range sales {
			revenue += sl.Amount.Data().Value
			currency = sl.Amount.Data().Currency
		}
	}

	return c.JSON(base.APIResponse{Data: BranchDetails{
		ID:      b.ID,
		Name:    b.Name,
		Workers: uint(len(b.Members)),
		Revenue: product.Price{Currency: currency, Value: revenue},
	}})
}

func (s *BranchService) Reports(c *fiber.Ctx) error {

	param := c.Params("id")
	id, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid branch id")
	}
	if err := s.assertWorkerBranchAccess(c, uint(id)); err != nil {
		return err
	}

	var productIDs []uint
	if err := s.DB.Table("products").Where("branch_id = ?", uint(id)).Pluck("id", &productIDs).Error; err != nil {
		return base.API_ERROR(c, "failed to get branch products")
	}

	var totalValue uint
	var totalSales uint
	currency := product.UGX

	if len(productIDs) > 0 {
		var sales []sale.Sale
		if err := s.DB.Where("product_id IN ?", productIDs).Find(&sales).Error; err != nil {
			return base.API_ERROR(c, "failed to get branch sales")
		}
		totalSales = uint(len(sales))
		for _, sl := range sales {
			totalValue += sl.Amount.Data().Value
			currency = sl.Amount.Data().Currency
		}
	}

	return c.JSON(base.APIResponse{Data: BranchReports{
		TotalSaleValue: product.Price{Currency: currency, Value: totalValue},
		TotalSales:     totalSales,
	}})
}

func (s *BranchService) FindUsers(c *fiber.Ctx) error {

	param := c.Params("id")
	id, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid branch id")
	}

	var b product.Branch
	if err := s.DB.Preload("Members").First(&b, uint(id)).Error; err != nil {
		return base.API_ERROR(c, "branch not found")
	}
	if err := s.assertWorkerBranchAccess(c, b.ID); err != nil {
		return err
	}

	var productIDs []uint
	if err := s.DB.Table("products").Where("branch_id = ?", uint(id)).Pluck("id", &productIDs).Error; err != nil {
		return base.API_ERROR(c, "failed to get branch products")
	}

	result := make([]BranchUser, 0, len(b.Members))
	for _, m := range b.Members {

		var revenue uint
		currency := product.UGX
		if len(productIDs) > 0 {
			var sales []sale.Sale
			if err := s.DB.Where("product_id IN ? AND seller_id = ?", productIDs, m.ID).Find(&sales).Error; err != nil {
				return base.API_ERROR(c, "failed to get user sales")
			}
			for _, sl := range sales {
				revenue += sl.Amount.Data().Value
				currency = sl.Amount.Data().Currency
			}
		}

		result = append(result, BranchUser{
			ID:      m.ID,
			Name:    m.Name,
			Phone:   m.Phone,
			Photo:   m.Photo,
			Revenue: product.Price{Currency: currency, Value: revenue},
		})
	}

	return c.JSON(base.APIResponse{Data: result})
}

func (s *BranchService) assertWorkerBranchAccess(c *fiber.Ctx, branchID uint) error {
	if !base.IsWorker(c) {
		return nil
	}
	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}
	var isMember int64
	if err := s.DB.Table("branch_members").
		Where("branch_id = ? AND user_id = ?", branchID, UserID).
		Count(&isMember).Error; err != nil || isMember == 0 {
		return base.API_ERROR(c, "branch not found")
	}
	return nil
}
