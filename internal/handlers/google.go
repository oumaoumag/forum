package handlers

import (
	"forum/internal/utils"
	"log"
	"net/http"
	"os"

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
		log.Printf("Warning: Google 0Auth environment  variables not fully configured")
		return
	}

	googleOAuthConfig = &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}
}

// GoogleLoginHandler initiates the Google Oauth flow
func GoogleLoginHandler(w http.ResponseWriter, r *http.Request) {
	if googleOAuthConfig == nil {
		utils.DisplayError(w, http.StatusInternalServerError, "Goolgle OAuth not configured")
		return
	}

	state := uuid.New().String()
	setStateCookie(w, state)

	url := googleOAuthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
