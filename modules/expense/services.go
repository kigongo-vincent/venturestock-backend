package expense

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
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

	branchIDs, bErr := shared.GetUserBranchIDs(s.DB, UserID)
	if bErr != nil {
		return base.API_ERROR(c, "failed to resolve branches")
	}
	if len(branchIDs) == 0 {
		return base.API_ERROR(c, "branch not found")
	}

	// default to the user's primary branch when the client does not send one
	if payload.BranchID == nil || *payload.BranchID == 0 {
		first := branchIDs[0]
		payload.BranchID = &first
	}

	allowed, aErr := shared.UserHasBranchAccess(s.DB, UserID, *payload.BranchID)
	if aErr != nil {
		return base.API_ERROR(c, "failed to verify branch access")
	}
	if !allowed {
		return base.API_ERROR(c, "branch not found")
	}

	payload.UserID = &UserID

	if err := s.DB.Create(&payload).Error; err != nil {
		return base.API_ERROR(c, "failed to record expense")
	}

	return c.JSON(base.APIResponse{Data: payload})
}

// FindByBranch returns expenses recorded under a specific branch.
func (s *ExpenseService) FindByBranch(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	param := c.Params("branchId")
	branchID, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid branch id")
	}

	allowed, aErr := shared.UserHasBranchAccess(s.DB, UserID, uint(branchID))
	if aErr != nil {
		return base.API_ERROR(c, "failed to verify branch access")
	}
	if !allowed {
		return base.API_ERROR(c, "branch not found")
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

	allowed := false
	if expense.BranchID != nil {
		var aErr error
		allowed, aErr = shared.UserHasBranchAccess(s.DB, UserID, *expense.BranchID)
		if aErr != nil {
			return base.API_ERROR(c, "failed to verify branch access")
		}
	}
	if !allowed {
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
		allowed, aErr := shared.UsersShareBranch(s.DB, RequestingUserID, uint(targetUserID))
		if aErr != nil {
			return base.API_ERROR(c, "failed to verify access")
		}
		if !allowed {
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
