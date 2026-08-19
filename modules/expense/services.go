package expense

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"gorm.io/gorm"
)

type ExpenseService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *ExpenseService {
	return &ExpenseService{DB: db}
}

func (s *ExpenseService) Create(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	var payload Expense
	if err := c.BodyParser(&payload); err != nil {
		return base.API_ERROR(c, "invalid payload")
	}

	payload.UserID = &UserID

	if err := s.DB.Create(&payload).Error; err != nil {
		return base.API_ERROR(c, "failed to record expense")
	}

	return c.JSON(base.APIResponse{Data: payload})
}

// FindByBranch returns expenses recorded under a specific branch.
func (s *ExpenseService) FindByBranch(c *fiber.Ctx) error {

	param := c.Params("branchId")
	branchID, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid branch id")
	}

	var expenses []Expense
	if err := s.DB.Preload("User").
		Where("branch_id = ?", uint(branchID)).
		Order("created_at DESC").
		Find(&expenses).Error; err != nil {
		return base.API_ERROR(c, "failed to get expenses")
	}

	return c.JSON(base.APIResponse{Data: expenses})
}

func (s *ExpenseService) Delete(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	param := c.Params("id")
	id, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid id")
	}

	var expense Expense
	if err := s.DB.First(&expense, uint(id)).Error; err != nil {
		return base.API_ERROR(c, "expense not found")
	}

	var isMember int64
	if err := s.DB.Table("branch_members").
		Where("branch_id = ? AND user_id = ?", expense.BranchID, UserID).
		Count(&isMember).Error; err != nil {
		return base.API_ERROR(c, "failed to verify branch access")
	}
	if isMember == 0 {
		return base.API_ERROR(c, "expense not found")
	}

	if err := s.DB.Delete(&Expense{}, uint(id)).Error; err != nil {
		return base.API_ERROR(c, "failed to delete expense")
	}

	return c.SendStatus(200)
}

func (s *ExpenseService) FindByUser(c *fiber.Ctx) error {

	RequestingUserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	param := c.Params("userId")
	targetUserID, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid user id")
	}

	// viewing your own expenses is always allowed; viewing someone else's
	// requires a shared branch
	if uint(targetUserID) != RequestingUserID {
		var shared int64
		if err := s.DB.Table("branch_members bm1").
			Joins("JOIN branch_members bm2 ON bm1.branch_id = bm2.branch_id").
			Where("bm1.user_id = ? AND bm2.user_id = ?", RequestingUserID, uint(targetUserID)).
			Count(&shared).Error; err != nil {
			return base.API_ERROR(c, "failed to verify access")
		}
		if shared == 0 {
			return base.API_ERROR(c, "user not found")
		}
	}

	var expenses []Expense
	if err := s.DB.Preload("User").Where("user_id = ?", uint(targetUserID)).
		Order("created_at DESC").
		Find(&expenses).Error; err != nil {
		return base.API_ERROR(c, "failed to get expenses")
	}

	return c.JSON(base.APIResponse{Data: expenses})
}
