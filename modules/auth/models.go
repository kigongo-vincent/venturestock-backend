package auth

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/gorm"
)

type AuthService struct {
	DB *gorm.DB
}

type SignupRequest struct {
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	Name             string `json:"name"`
	Password         string `json:"password"`
	Photo            string `json:"photo"`
	BusinessName     string `json:"businessName"`
	BusinessLocation string `json:"businessLocation"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  shared.User `json:"user"`
}

type BranchMember struct {
	UserID   uint `json:"user_id"`
	BranchID uint `json:"branch_id"`
}

// GoogleAuthRequest is sent from the app after a successful native Google
// Sign-In. We only trust the idToken — everything else (name/email/photo)
// is re-derived server-side from Google, never taken from the client.
type GoogleAuthRequest struct {
	IDToken string `json:"idToken"`
}

// GoogleAuthResponse covers both outcomes of hitting /auth/googleAuth:
//   - IsNewUser=false: Token/User are populated, behave exactly like LoginResponse.
//   - IsNewUser=true: Token/User are empty; Profile carries verified fields from
//     Google so the client can prefill signup without asking the user to retype
//     anything, and without ever handling a password for this account.
type GoogleAuthResponse struct {
	IsNewUser bool           `json:"isNewUser"`
	Token     string         `json:"token,omitempty"`
	User      shared.User    `json:"user,omitempty"`
	Profile   *GoogleProfile `json:"profile,omitempty"`
}

// GoogleProfile is the verified identity handed back for prefill when
// IsNewUser is true.
type GoogleProfile struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Photo    string `json:"photo"`
	GoogleID string `json:"googleId"`
}

// GoogleSignupRequest is used for the "finish account creation after Google
// sign-in" path. No Password field — these accounts authenticate via Google,
// tokens are always minted by the backend in GenerateJWT.
type GoogleSignupRequest struct {
	IDToken          string `json:"idToken"` // re-verified server-side, source of truth for name/email/photo/googleId
	BusinessName     string `json:"businessName"`
	BusinessLocation string `json:"businessLocation"`
}
