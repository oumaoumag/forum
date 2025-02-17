package handlers

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOuthConfig *oauth2.Config
)


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

	googleOuthConfig = &oauth2.Config{
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
