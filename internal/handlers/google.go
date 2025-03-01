package handlers

import (
	"context"
	"forum/internal/db"
	"forum/internal/utils"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOAuthConfig *oauth2.Config
)

// init sets up the OAuth2 configuration for google authentication.
func init() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if googleClientID == "" || googleClientSecret == "" || redirectURL == "" {
		log.Printf("Warning: Google OAuth environment variables not fully configured")
		return
	}

	googleOAuthConfig = &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}
}

// GoogleLoginHandler initiates the Google Oauth flow
func GoogleLoginHandler(w http.ResponseWriter, r *http.Request) {
	if googleOAuthConfig == nil {
		utils.DisplayError(w, http.StatusInternalServerError, "Google OAuth not configured")
		return
	}

	state := uuid.New().String()
	utils.SetStateCookie(w, state)

	url := googleOAuthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleCallbackHandler handles the OAuth callback from Google
func GoogleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if googleOAuthConfig == nil {
		utils.DisplayError(w, http.StatusInternalServerError, "Google OAuth not configured")
		return
	}

	// Verify state
	state := r.FormValue("state")
	savedState := utils.GetStateFromCookie(r)
	if state != savedState {
		utils.DisplayError(w, http.StatusBadRequest, "Invalid OAuth state")
		return
	}

	// Exchange code for token
	code := r.FormValue("code")
	token, err := googleOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Code exchange failed: %v", err)
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to complete authentication")
		return
	}

	// Get user info from Google
	googleUser, err := utils.GetGoogleUserInfo(token.AccessToken)
	if err != nil {
		log.Printf("Failed to get Google user info: %v", err)
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to get user information")
		return
	}

	// Ensure email is verified
	if !googleUser.VerifiedEmail {
		utils.DisplayError(w, http.StatusForbidden, "Email not verified with Google")
		return
	}

	// Find or create user in our database
	userID, err := utils.FindOrCreateGoogleUser(googleUser)
	if err != nil {
		log.Printf("Failed to process user: %v", err)
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to process user information")
		return
	}

	sessionID := uuid.New().String()
	expiration := time.Now().Add(24 * time.Hour)

	_, err = db.DB.Exec(
		`INSERT INTO sessions (session_id, user_id, expires_at) VALUES (?,?,?)`, sessionID, userID, expiration)
	if err != nil {
		log.Printf("Failed to insert sessions into database: %v", err)
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  expiration,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/?login_success=true", http.StatusSeeOther)
}
