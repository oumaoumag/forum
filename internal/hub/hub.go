package hub

import (
	"encoding/json"
	"fmt"
	"forum/internal/db"

	"net/http"
	"time"

	"github.com/google/uuid"
)

// findOrCreateUser either finds an existing user or creates a new one
func FindOrCreateUser(githubUser *GitHubUser) (int, error) {
	var userID int
	username := githubUser.Login
	if username == "" {
		username = "github_user_" + uuid.New().String()[:8]
	}
	// Check if the username exists
	err := db.DB.QueryRow("SELECT user_id FROM users WHERE username = ?", username).Scan(&userID)
	if err == nil {
		// Username already exists, so we can handle it by generating a new one
		// For example, append a random number or UUID to the username
		username = username + "_" + uuid.New().String()[:8]
		return userID, nil
	}
	// Check if user exists by Github ID
	err = db.DB.QueryRow("SELECT user_id FROM users WHERE provider_id = ? AND auth_type = 'github'", githubUser.ID).Scan(&userID)
	if err == nil {
		return userID, nil
	}
	
	// Check if user exists by email
	// Fallback check by email
	err = db.DB.QueryRow("SELECT user_id FROM users WHERE email = ?", githubUser.Email).Scan(&userID)
	if err == nil {
		// Update existing user with Github info
		_, err = db.DB.Exec(
			"UPDATE users SET provider_id = ?, auth_type = 'github' WHERE user_id = ?",
			githubUser.ID,
			userID,
		)
		return userID, err
	}

	result, err := db.DB.Exec(
		"INSERT INTO users (email, username, password, auth_type, provider_id) VALUES (?,?,?, 'github', ?)",
		githubUser.Email,
		username,
		"oauth_placeholder", // Password placeholder
		githubUser.ID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}
	return int(id), nil
}

// getGitHubUserInfo fetches the user's information from GitHub
func GetGitHubUserInfo(accessToken string) (*GitHubUser, error) {
	// Get basic user information
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

	// Fetch email information from GitHub
	emailReq, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create email request: %w", err)
	}

	emailReq.Header.Set("Authorization", "Bearer "+accessToken)
	emailReq.Header.Set("Accept", "application/json")

	emailResp, err := client.Do(emailReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get emails: %w", err)
	}
	defer emailResp.Body.Close()

	var emails []GitHubEmail
	if err := json.NewDecoder(emailResp.Body).Decode(&emails); err != nil {
		return nil, fmt.Errorf("failed to decode emails: %w", err)
	}

	// Check if there's a verified email
	for _, email := range emails {
		if email.Verified {
			user.Email = email.Email
			user.VerifiedEmail = true
			break
		}
	}

	return &user, nil
}

// GitHubEmail represents the structure of an email from GitHub's /user/emails API response
type GitHubEmail struct {
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
}

// GitHubUser represents the GitHub user data we need
type GitHubUser struct {
	Login string `json:"login"`
	Email string `json:"email"`
	Name  string `json:"name"`
	ID        int `json:"id"`
	VerifiedEmail bool   `json:"verified_email"`
}
// Function to set the state cookie
func SetStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Expires:  time.Now().Add(24 * time.Hour),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,  // Set to true in production for HTTPS
		SameSite: http.SameSiteLaxMode,
	})
}

// Function to get the state from the cookie
func GetStateFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("oauth_state")
	if err != nil {
		
		return ""  // Return empty string if cookie is not found
	}
	return cookie.Value
}