package handlers

import (
	"context"
	
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"forum/internal/db"
	"forum/internal/utils"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"forum/internal/hub"
	
)

var (
	githubOAuthConfig *oauth2.Config
	//oauthStateString  = uuid.New().String() // Using UUID for better security
)

func init() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	redirectURL := os.Getenv("GITHUB_REDIRECT_URL")

	// Check if required variables are set
	if githubClientID == "" || githubClientSecret == "" || redirectURL == "" {
		log.Printf("Warning: GitHub OAuth environment variables not fully configured")
		return
	}

	// Initialize GitHub OAuth config
	githubOAuthConfig = &oauth2.Config{
		ClientID:     githubClientID,
		ClientSecret: githubClientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"user:email",
		},
		Endpoint: github.Endpoint,
	}
}



// GitHubLoginHandler initiates the GitHub OAuth flow
func GitHubLoginHandler(w http.ResponseWriter, r *http.Request) {
	if githubOAuthConfig == nil {
		utils.DisplayError(w, http.StatusInternalServerError, "Github OAuth not configured")
		return
	}

	state := uuid.New().String()
	hub.SetStateCookie(w, state)

	url := githubOAuthConfig.AuthCodeURL(state)
	
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GitHubCallbackHandler handles the OAuth callback from GitHub
func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if githubOAuthConfig == nil {
		utils.DisplayError(w, http.StatusInternalServerError, "Github OAuth not configured")
		return
	}

	// Verify state
	state := r.FormValue("state")
	savedState := hub.GetStateFromCookie(r)


	if state != savedState {
		utils.DisplayError(w, http.StatusBadRequest, "Invalid OAuth state")
		return
	}

	// Exchange code for token
	code := r.FormValue("code")
	if code == "" {
		utils.DisplayError(w, http.StatusBadRequest, "Authorization code missing")
		return
	}

	

	// Get user info from GitHub
	token, err := githubOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Code exchange failed: %v", err) // Detailed logging
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to complete authentication")
		return
	}

	githubUser, err := hub.GetGitHubUserInfo(token.AccessToken)
	if err != nil {
		log.Printf("Failed to get GitHub user info: %v", err)
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to get user information")
		return
	}

	// Ensure email is verified
	if !githubUser.VerifiedEmail {
		utils.DisplayError(w, http.StatusForbidden, "Email not verified with Github")
		return
	}

	// Find or create user in our database
	userID, err := hub.FindOrCreateUser(githubUser)
	if err != nil {
		log.Printf("Failed to process user: %v", err)
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to process user information")
		return
	}

	

	// Delete any existing sessions for this user
	if err := deleteExistingSessions(userID); err != nil {
		log.Printf("Failed to delete existing sessions: %v", err)
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to manage session")
		return
	}

	// Create new session ID
	sessionID := uuid.New().String()
	expiration := time.Now().Add(24 * time.Hour)

	// Insert new session into database
	_, err = db.DB.Exec(
		`INSERT INTO sessions (session_id, user_id, expires_at) VALUES (?, ?, ?)`,
		sessionID, userID, expiration,
	)
	if err != nil {
		log.Printf("Failed to insert session into database: %v", err)
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  expiration,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to home page
	http.Redirect(w, r, "/?login_success=true", http.StatusSeeOther)
}


// Helper function to delete existing sessions
func deleteExistingSessions(userID int) error {
	_, err := db.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete existing sessions: %w", err)
	}
	return nil
}




