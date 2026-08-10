package auth

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/branch"
	"github.com/kigongo-vincent/venturestock-backend/modules/business"
	"github.com/kigongo-vincent/venturestock-backend/modules/user"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func NewService(db *gorm.DB) *AuthService {
	return &AuthService{DB: db}
}

func (s *AuthService) Login(c *fiber.Ctx) error {

	var body LoginRequest
	if err := c.BodyParser(&body); err != nil {
		return base.API_ERROR(c, "invalid input")
	}

	userService := user.NewService(s.DB)
	user, uErr := userService.FindOne(user.User{Email: string(body.Identifier), Phone: string(body.Identifier)})
	if uErr != nil {
		return base.API_ERROR(c, "user account not found")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)) != nil {
		return base.API_ERROR(c, "invalid credentials")
	}
	user.Password = ""
	data := LoginUser(user)
	return c.JSON(base.APIResponse{Data: data})
}

func LoginUser(user user.User) LoginResponse {
	tkn, _ := GenerateJWT(user)
	return LoginResponse{Token: tkn, User: user}
}

func HashPassword(password string) string {
	pwdBytes := []byte(password)
	hashBytes, _ := bcrypt.GenerateFromPassword(pwdBytes, bcrypt.DefaultCost)
	return string(hashBytes)
}

func GenerateJWT(user user.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":  time.Now().Add(time.Hour * 24 * 30).Unix(),
		"id":   user.ID,
		"role": user.Role,
	})

	signedToken, tokenErr := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if tokenErr != nil {
		return signedToken, nil
	}
	return signedToken, nil
}

func (s *AuthService) Signup(c *fiber.Ctx) error {

	var body SignupRequest
	if err := c.BodyParser(&body); err != nil {
		return base.API_ERROR(c, "failed to process input")
	}

	// simple user service
	userService := user.NewService(s.DB)
	// simple business service
	businessService := business.NewService(s.DB)
	// simple branch service
	branchService := branch.NewService(s.DB)

	// check if the user exists
	exists, _ := userService.Exists(user.User{Phone: body.Phone, Email: body.Email, Role: user.Admin})
	// if existsErr == nil {
	if exists {
		return base.API_ERROR(c, "user account exists") // still nil here — still panics
	}

	// create the user
	UserID, uErr := userService.Create(user.User{Phone: body.Phone, Name: body.Name, Email: body.Email, Password: HashPassword(body.Password), Photo: body.Photo})
	if uErr != nil {
		return base.API_ERROR(c, "failed to create the user account")
	}

	// create the business with the user as the head
	BusinessID, bErr := businessService.Create(business.Business{Name: body.BusinessName, Location: body.BusinessLocation, OwnerID: &UserID})
	if bErr != nil {
		return base.API_ERROR(c, "failed to create the business")
	}

	// create the main branch for the business
	BranchID, braErr := branchService.Create(branch.Branch{Name: "Main", BusinessID: &BusinessID})
	if braErr != nil {
		return base.API_ERROR(c, "failed to create branch")
	}

	// create the branch member for the user
	mErr := branchService.AddMember(BranchID, UserID)
	if mErr != nil {
		return base.API_ERROR(c, "failed to add member to branch")
	}

	user := user.User{
		Email: body.Email,
		Phone: body.Phone,
		Name:  body.Name,
		Model: gorm.Model{ID: UserID},
	}
	data := LoginUser(user)
	return c.JSON(base.APIResponse{Data: data})
}

func (s *AuthService) GoogleAuth(c *fiber.Ctx) error {
	return nil
}
