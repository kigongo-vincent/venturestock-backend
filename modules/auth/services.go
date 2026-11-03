package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/branch"
	"github.com/kigongo-vincent/venturestock-backend/modules/business"
	"github.com/kigongo-vincent/venturestock-backend/modules/notifications"
	"github.com/kigongo-vincent/venturestock-backend/modules/product"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"github.com/kigongo-vincent/venturestock-backend/modules/subscription"
	"github.com/kigongo-vincent/venturestock-backend/modules/user"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"
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
	user, uErr := userService.FindOne(shared.User{Email: string(body.Identifier), Phone: string(body.Identifier)})
	if uErr != nil {
		return base.API_ERROR(c, "user account not found")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)) != nil {
		return base.API_ERROR(c, "invalid credentials")
	}
	user.Password = ""

	// get the branch for the user
	var branchMember BranchMember
	if err := s.DB.Table("branch_members").Where("user_id=?", user.ID).First(&branchMember).Error; err != nil {
		return base.API_ERROR(c, fmt.Sprintf("failed to get branch member error: %v", err))
	}

	var branch shared.Branch
	if err := s.DB.Preload("Business").First(&branch, branchMember.BranchID).Error; err != nil {
		return base.API_ERROR(c, "failed to get branch")
	}
	subscription.NewService(s.DB).AttachSnapshot(branch.Business)

	user.Branch = branch
	data := LoginUser(user)
	return c.JSON(base.APIResponse{Data: data})
}

func LoginUser(user shared.User) LoginResponse {
	tkn, _ := GenerateJWT(user)
	return LoginResponse{Token: tkn, User: user}
}

func HashPassword(password string) string {
	pwdBytes := []byte(password)
	hashBytes, _ := bcrypt.GenerateFromPassword(pwdBytes, bcrypt.DefaultCost)
	return string(hashBytes)
}

func GenerateJWT(user shared.User) (string, error) {
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
	exists, _ := userService.Exists(shared.User{Phone: body.Phone, Email: body.Email, Role: shared.Admin})
	// if existsErr == nil {
	if exists {
		return base.API_ERROR(c, "user account exists") // still nil here — still panics
	}

	// create the user

	UserID, uErr := userService.Create(shared.User{Phone: body.Phone, Name: body.Name, Email: body.Email, Password: HashPassword(body.Password), Photo: body.Photo, Role: shared.Admin})
	if uErr != nil {
		return base.API_ERROR(c, "failed to create the user account")
	}

	nSVC := notifications.NewService(s.DB)
	nSVC.Create(notifications.Notification{Path: "/extras/notifications", UserID: &UserID, Title: "Welcome aboard 🎉🎉🎉🎉🎉", Description: fmt.Sprintf("hello, %s, your account was created successfully", body.Name)})

	// create the business with the user as the head
	BusinessID, bErr := businessService.Create(shared.Business{Name: body.BusinessName, Location: body.BusinessLocation, OwnerID: &UserID})
	if bErr != nil {
		return base.API_ERROR(c, "failed to create the business")
	}

	nSVC.Create(notifications.Notification{Path: "/settings/business", UserID: &UserID, Title: "Business registration completed 🎉🎉", Description: fmt.Sprintf("hello, %s, your business \"%s\" was registered successfully", body.Name, body.BusinessName)})

	if err := subscription.NewService(s.DB).CreateFreeSubscription(BusinessID); err != nil {
		return base.API_ERROR(c, "failed to create the free subscription")
	}

	// create the main branch for the business
	BranchID, braErr := branchService.Create(product.Branch{Name: "Main", BusinessID: &BusinessID})
	if braErr != nil {
		return base.API_ERROR(c, "failed to create branch")
	}

	// create the branch member for the user
	mErr := shared.AddMember(s.DB, BranchID, UserID)
	if mErr != nil {
		return base.API_ERROR(c, "failed to add member to branch")
	}

	var branch shared.Branch
	if err := s.DB.Preload("Business").First(&branch, BranchID).Error; err != nil {
		return base.API_ERROR(c, "failed to get branch")
	}
	subscription.NewService(s.DB).AttachSnapshot(branch.Business)

	nSVC.Create(notifications.Notification{Path: fmt.Sprintf(`/extras/branch?id=%v`, BranchID), UserID: &UserID, Title: "Branch created 🎉", Description: fmt.Sprintf("hello, %s, a main branch has been created for your business, you're initially added as a member of the branch", body.Name)})

	user := shared.User{
		Email:  body.Email,
		Phone:  body.Phone,
		Name:   body.Name,
		Role:   shared.Admin,
		Model:  gorm.Model{ID: UserID},
		Branch: branch,
	}
	data := LoginUser(user)
	return c.JSON(base.APIResponse{Data: data})
}

func verifyGoogleIDToken(ctx context.Context, rawToken string) (*GoogleProfile, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID") // web/android client ID configured in Google Cloud console

	payload, err := idtoken.Validate(ctx, rawToken, clientID)
	if err != nil {
		return nil, err
	}

	profile := &GoogleProfile{
		GoogleID: payload.Subject,
	}
	if email, ok := payload.Claims["email"].(string); ok {
		profile.Email = email
	}
	if name, ok := payload.Claims["name"].(string); ok {
		profile.Name = name
	}
	if picture, ok := payload.Claims["picture"].(string); ok {
		profile.Photo = picture
	}

	return profile, nil
}

func (s *AuthService) GoogleAuth(c *fiber.Ctx) error {

	var body GoogleAuthRequest
	if err := c.BodyParser(&body); err != nil {
		return base.API_ERROR(c, "invalid input")
	}
	if body.IDToken == "" {
		return base.API_ERROR(c, "missing google token")
	}

	profile, vErr := verifyGoogleIDToken(c.Context(), body.IDToken)
	if vErr != nil {
		return base.API_ERROR(c, "invalid or expired google token")
	}

	userService := user.NewService(s.DB)
	// NOTE: if GoogleID isn't a field on shared.User yet, add it (unique,
	// nullable) so returning Google users resolve even if they originally
	// signed up with a different/no email.
	existingUser, uErr := userService.FindOne(shared.User{Email: profile.Email})

	if uErr != nil {
		// no account yet — hand back verified profile for prefill, create nothing
		return c.JSON(base.APIResponse{Data: GoogleAuthResponse{
			IsNewUser: true,
			Profile:   profile,
		}})
	}

	// existing user — log in exactly like password login does
	existingUser.Password = ""

	var branchMember BranchMember
	if err := s.DB.Table("branch_members").Where("user_id=?", existingUser.ID).First(&branchMember).Error; err != nil {
		return base.API_ERROR(c, "failed to get branch member")
	}

	var userBranch shared.Branch
	if err := s.DB.Preload("Business").First(&userBranch, branchMember.BranchID).Error; err != nil {
		return base.API_ERROR(c, "failed to get branch")
	}
	subscription.NewService(s.DB).AttachSnapshot(userBranch.Business)

	existingUser.Branch = userBranch
	loginData := LoginUser(existingUser)

	return c.JSON(base.APIResponse{Data: GoogleAuthResponse{
		IsNewUser: false,
		Token:     loginData.Token,
		User:      loginData.User,
	}})
}

func (s *AuthService) GoogleSignup(c *fiber.Ctx) error {

	var body GoogleSignupRequest
	if err := c.BodyParser(&body); err != nil {
		return base.API_ERROR(c, "failed to process input")
	}
	if body.IDToken == "" {
		return base.API_ERROR(c, "missing google token")
	}

	profile, vErr := verifyGoogleIDToken(c.Context(), body.IDToken)
	if vErr != nil {
		return base.API_ERROR(c, "invalid or expired google token")
	}

	userService := user.NewService(s.DB)
	businessService := business.NewService(s.DB)
	branchService := branch.NewService(s.DB)

	exists, _ := userService.Exists(shared.User{Email: profile.Email, Role: shared.Admin})
	if exists {
		return base.API_ERROR(c, "user account exists")
	}

	UserID, uErr := userService.Create(shared.User{
		Phone: "", // not collected in the google flow; adjust if you still require it
		Name:  profile.Name,
		Email: profile.Email,
		Photo: profile.Photo,
		// Password intentionally left empty — this account authenticates via Google only.
		Role: shared.Admin,
	})
	if uErr != nil {
		return base.API_ERROR(c, "failed to create the user account")
	}

	nSVC := notifications.NewService(s.DB)
	nSVC.Create(notifications.Notification{UserID: &UserID, Title: "Welcome aboard 🎉🎉🎉🎉🎉", Description: "hello, " + profile.Name + ", your account was created successfully"})

	BusinessID, bErr := businessService.Create(shared.Business{Name: body.BusinessName, Location: body.BusinessLocation, OwnerID: &UserID})
	if bErr != nil {
		return base.API_ERROR(c, "failed to create the business")
	}

	nSVC.Create(notifications.Notification{UserID: &UserID, Title: "Business registration completed 🎉🎉", Description: "hello, " + profile.Name + ", your business\"" + body.BusinessName + "\" was registered successfully"})

	if err := subscription.NewService(s.DB).CreateFreeSubscription(BusinessID); err != nil {
		return base.API_ERROR(c, "failed to create the free subscription")
	}

	BranchID, braErr := branchService.Create(product.Branch{Name: "Main", BusinessID: &BusinessID})
	if braErr != nil {
		return base.API_ERROR(c, "failed to create branch")
	}

	if mErr := shared.AddMember(s.DB, BranchID, UserID); mErr != nil {
		return base.API_ERROR(c, "failed to add member to branch")
	}

	var userBranch shared.Branch
	if err := s.DB.Preload("Business").First(&userBranch, BranchID).Error; err != nil {
		return base.API_ERROR(c, "failed to get branch")
	}
	subscription.NewService(s.DB).AttachSnapshot(userBranch.Business)

	nSVC.Create(notifications.Notification{UserID: &UserID, Title: "Branch created 🎉", Description: "hello, " + profile.Name + ", a main branch has been created for your business, you're initially added as a member of the branch"})

	newUser := shared.User{
		Email:  profile.Email,
		Name:   profile.Name,
		Photo:  profile.Photo,
		Role:   shared.Admin,
		Model:  gorm.Model{ID: UserID},
		Branch: userBranch,
	}
	data := LoginUser(newUser)
	return c.JSON(base.APIResponse{Data: data})
}
