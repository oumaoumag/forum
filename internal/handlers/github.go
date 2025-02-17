package handlers

import (
	"context"
	"encoding/json"
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
)

var (
	githubOAuthConfig *oauth2.Config
	oauthStateString  = uuid.New().String() // Using UUID for better security
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

// GitHubUser represents the GitHub user data we need
type GitHubUser struct {
	Login string `json:"login"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// GitHubLoginHandler initiates the GitHub OAuth flow
func GitHubLoginHandler(w http.ResponseWriter, r *http.Request) {
	if githubOAuthConfig == nil {
		utils.DisplayError(w, http.StatusInternalServerError, "GitHub OAuth not configured")
		return
	}

	url := githubOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// In github.go

// GitHubCallbackHandler handles the OAuth callback from GitHub
func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if githubOAuthConfig == nil {
		utils.DisplayError(w, http.StatusInternalServerError, "GitHub OAuth not configured")
		return
	}

	// Verify state to prevent CSRF
	state := r.FormValue("state")
	if state != oauthStateString {
		utils.DisplayError(w, http.StatusBadRequest, "Invalid OAuth state")
		return
	}

	// Exchange code for token
	code := r.FormValue("code")
	token, err := githubOAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("Code exchange failed: %v", err)
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to complete authentication")
		return
	}

	// Get user info from GitHub
	githubUser, err := getGitHubUserInfo(token.AccessToken)
	if err != nil {
		log.Printf("Failed to get GitHub user info: %v", err)
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to get user information")
		return
	}

	// Find or create user in our database
	userID, err := findOrCreateUser(githubUser)
	if err != nil {
		log.Printf("Failed to process user: %v", err)
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to process user information")
		return
	}

	log.Printf("Successfully found/created user with ID: %d", userID) // Debug log

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
		SameSite: http.SameSiteStrictMode,
	})

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Helper function to delete existing sessions
func deleteExistingSessions(userID int) error {
	_, err := db.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete existing sessions: %w", err)
	}
	return nil
}

// getGitHubUserInfo fetches the user's information from GitHub
func getGitHubUserInfo(accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &user, nil
}

// findOrCreateUser either finds an existing user or creates a new one
func findOrCreateUser(githubUser *GitHubUser) (int, error) {
	var userID int
	username := githubUser.Login
	if username == "" {
		username = "github_user_" + uuid.New().String()[:8]
	}

	// Check if user exists by email
	err := db.DB.QueryRow("SELECT user_id FROM users WHERE email = ?", githubUser.Email).Scan(&userID)
	if err == nil {
		// User exists, update username if necessary
		_, err = db.DB.Exec("UPDATE users SET username = ? WHERE user_id = ?", username, userID)
		return userID, err
	}

	// If user doesn't exist, create a new user
	fakePassword := "oauth_placeholder" // Placeholder password for OAuth users

	result, err := db.DB.Exec(
		"INSERT INTO users (email, username, password, auth_type) VALUES (?, ?, ?, 'github')",
		githubUser.Email,
		username,
		fakePassword,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get new user ID: %w", err)
	}

	return int(id), nil
}
