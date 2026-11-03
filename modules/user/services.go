package user

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"

	// "github.com/kigongo-vincent/venturestock-backend/modules/branch"

	// "github.com/kigongo-vincent/venturestock-backend/modules/branch"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func NewService(db *gorm.DB) *UserService {
	return &UserService{DB: db}
}
func (s *UserService) Exists(user shared.User) (bool, error) {
	var count int64
	query := s.DB.Model(&shared.User{})

	if user.Phone != "" && user.Email != "" {
		query = query.Where("phone = ? OR email = ?", user.Phone, user.Email)
	} else if user.Phone != "" {
		query = query.Where("phone = ?", user.Phone)
	} else if user.Email != "" {
		query = query.Where("email = ?", user.Email)
	} else {
		// nothing to check against — treat as not existing rather than
		// matching every other user with a blank phone and email
		return false, nil
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *UserService) FindOne(user shared.User) (shared.User, error) {
	err := s.DB.Model(&shared.User{}).
		Where("phone = ? OR email = ?", user.Phone, user.Email).
		First(&user).Error
	if err != nil {
		return shared.User{}, err
	}
	return user, nil
}

func (s *UserService) Create(user shared.User) (uint, error) {
	if err := s.DB.Create(&user).Error; err != nil {
		return 0, err
	}
	return user.ID, nil
}

// CreateWorker handles POST /users — adds a worker under the requesting
// user's branch, hashing their password and adding them as a branch
// member in one transaction.
func (s *UserService) CreateWorker(c *fiber.Ctx) error {

	RequestingUserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}
	if err := base.DenyWorker(c, "workers cannot add other workers"); err != nil {
		return err
	}

	var payload CreateWorkerPayload
	if err := c.BodyParser(&payload); err != nil {
		return base.API_ERROR(c, "invalid payload")
	}

	if payload.Name == "" {
		return base.API_ERROR(c, "name is required")
	}
	if payload.Password == "" {
		return base.API_ERROR(c, "password is required")
	}
	if payload.BranchID == 0 {
		return base.API_ERROR(c, "branch is required")
	}
	if payload.Phone == "" && payload.Email == "" {
		return base.API_ERROR(c, "phone or email is required")
	}

	allowed, aErr := shared.UserHasBranchAccess(s.DB, RequestingUserID, payload.BranchID)
	if aErr != nil {
		return base.API_ERROR(c, "failed to verify branch access")
	}
	if !allowed {
		return base.API_ERROR(c, "branch not found")
	}

	exists, eErr := s.Exists(shared.User{Phone: payload.Phone, Email: payload.Email})
	if eErr != nil {
		return base.API_ERROR(c, "failed to verify worker details")
	}
	if exists {
		return base.API_ERROR(c, "a user with this phone or email already exists")
	}

	hashed, hErr := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if hErr != nil {
		return base.API_ERROR(c, "failed to process password")
	}

	newUser := shared.User{
		Name:     payload.Name,
		Email:    payload.Email,
		Phone:    payload.Phone,
		Password: string(hashed),
		Role:     shared.Default,
		IsActive: true,
	}

	tx := s.DB.Begin()
	txUserService := &UserService{DB: tx}

	newUserID, cErr := txUserService.Create(newUser)
	if cErr != nil {
		tx.Rollback()
		return base.API_ERROR(c, "failed to create user")
	}
	if err := shared.AddMember(tx, payload.BranchID, newUserID); err != nil {
		tx.Rollback()
		return base.API_ERROR(c, fmt.Sprintf("failed to add user to branch %v user: %v branch: %v", err.Error(), newUserID, payload.BranchID))
	}

	if err := tx.Commit().Error; err != nil {
		return base.API_ERROR(c, "failed to create user")
	}

	newUser.ID = newUserID
	newUser.Password = ""

	return c.JSON(base.APIResponse{Data: newUser})
}

func (s *UserService) FindByID(c *fiber.Ctx) error {

	RequestingUserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	param := c.Params("id")
	targetID, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "invalid user id")
	}

	// allow viewing your own profile always, otherwise require a shared branch
	if uint(targetID) != RequestingUserID {
		allowed, aErr := shared.UsersShareBranch(s.DB, RequestingUserID, uint(targetID))
		if aErr != nil {
			return base.API_ERROR(c, "failed to verify access")
		}
		if !allowed {
			return base.API_ERROR(c, "user not found")
		}
	}

	var target shared.User
	if err := s.DB.First(&target, uint(targetID)).Error; err != nil {
		return base.API_ERROR(c, "user not found")
	}

	target.Password = ""

	return c.JSON(base.APIResponse{Data: target})
}

func (s *UserService) UpdateAccount(c *fiber.Ctx) error {

	var body UpdateAccountRequest
	if err := c.BodyParser(&body); err != nil {
		return base.API_ERROR(c, "failed to get user info")
	}

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "failed to get user")
	}

	var User shared.User
	if err := s.DB.First(&User, UserID).Error; err != nil {
		return base.API_ERROR(c, "user account not found")
	}

	if body.Name != "" {
		User.Name = body.Name
	}

	if body.Photo != "" {
		User.Photo = body.Photo
	}

	if err := s.DB.Save(&User).Error; err != nil {
		return base.API_ERROR(c, "failed to update user account")
	}

	return c.SendStatus(200)

}

func (s *UserService) UpdatePhoneNumber(c *fiber.Ctx) error {

	var body UpdatePhoneRequest
	if err := c.BodyParser(&body); err != nil {
		return base.API_ERROR(c, "missing phone number")
	}

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "failed to get user")
	}

	var user shared.User
	if err := s.DB.First(&user, UserID).Error; err != nil {
		return base.API_ERROR(c, "failed to get user")
	}

	if body.Phone != "" {
		user.Phone = body.Phone
	}

	// check if exists
	e, _ := s.Exists(shared.User{Phone: body.Phone})
	if e {
		return base.API_ERROR(c, "phone number is already registered another account")
	}

	if err := s.DB.Save(&user).Error; err != nil {
		return base.API_ERROR(c, "failed to save user")
	}

	return c.SendStatus(200)

}

func cryptoRandInt(min, max int64) (int64, error) {
	if min > max {
		return 0, fmt.Errorf("minimum value cannot be greater than maximum value")
	}

	rangeSize := new(big.Int).SetInt64(max - min + 1)
	randomNum, err := rand.Int(rand.Reader, rangeSize)
	if err != nil {
		return 0, err
	}

	return randomNum.Int64() + min, nil
}

func SendCode(code string, email string) {
	fmt.Printf("code: %v - email: %v", code, email)
}

func (s *UserService) RequestOTP(c *fiber.Ctx) error {

	var body RequestOTPRequest
	if err := c.BodyParser(&body); err != nil {
		return base.API_ERROR(c, "failed to parse input")
	}

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "failed to get user")
	}

	rawCode, cErr := cryptoRandInt(100000, 999999)
	if cErr != nil {
		return base.API_ERROR(c, "failed to generate code")
	}

	// code := fmt.Sprintf("%v", rawCode)

	code := strconv.Itoa(int(rawCode))
	SendCode(code, body.Email)

	codeBytes, hErr := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if hErr != nil {
		return base.API_ERROR(c, "failed to store code securely")
	}

	code = string(codeBytes)
	if err := s.DB.Create(&OTP{UserID: &UserID, Code: code, ExpiryDate: time.Now().Add(time.Minute * 1)}).Error; err != nil {
		return base.API_ERROR(c, "failed to generate token")
	}
	return c.SendStatus(200)
}

func (s *UserService) UpdateEmail(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not allowed")
	}

	var body UpdateEmailRequest
	if err := c.BodyParser(&body); err != nil {
		return base.API_ERROR(c, "failed to parse input")
	}

	var otp OTP
	if err := s.DB.Where("user_id=?", UserID).Last(&otp).Error; err != nil {
		return base.API_ERROR(c, "code not found")
	}

	if bcrypt.CompareHashAndPassword([]byte(otp.Code), []byte(body.Code)) != nil {
		return base.API_ERROR(c, "invalid code")
	}

	diff := time.Now().Unix() - otp.ExpiryDate.Unix()
	if diff > int64(time.Minute*1) {
		return base.API_ERROR(c, "code is expired")
	}

	var user shared.User
	if err := s.DB.First(&user, UserID).Error; err != nil {
		return base.API_ERROR(c, "failed to get user")
	}

	user.Email = body.Email
	if err := s.DB.Save(&user).Error; err != nil {
		return base.API_ERROR(c, "failed to change email")
	}

	// clear the code
	if err := s.DB.Delete(&otp).Error; err != nil {
		return base.API_ERROR(c, "failed to clear code")
	}

	return c.SendStatus(200)
}
