package utils

import (
	"encoding/json"
	"fmt"
	"forum/internal/db"
	"forum/internal/models"
	"net/http"

	"github.com/google/uuid"
)

var httpClient = &http.Client{}

// GetGoogleUserInfo retrieves user information from Google's OAuth2 API
func GetGoogleUserInfo(accessToken string) (*models.GoogleUser, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var user models.GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}
	return &user, nil
}

// FindOrCreateGoogleUser() finds a new user in the database based on the information provided by Google's OAuth2 API.
func FindOrCreateGoogleUser(googleUser *models.GoogleUser) (int, error) {
	var userID int

	// check if user exists by Google ID
	err := db.DB.QueryRow("SELECT user_id FROM users WHERE provider_id = ? AND auth_type = 'google'", googleUser.ID).Scan(&userID)
	if err == nil {
		return userID, nil
	}

	// Fallback check by email
	err = db.DB.QueryRow("SELECT user_id FROM users WHERE email = ?", googleUser.Email).Scan(&userID)
	if err == nil {
		// Update existing user with Goole info
		_, err = db.DB.Exec(
			"UPDATE users SET provider_id = ?, auth_type = 'google' WHERE user_id = ?",
			googleUser.ID,
			userID,
		)
		return userID, err
	}

	// Create new user
	username := googleUser.Name
	if username == "" {
		username = "google_user_" + uuid.New().String()[:8]
	}

	result, err := db.DB.Exec(
		"INSERT INTO users (email, username, password, auth_type, provider_id) VALUES (?,?,? 'google', ?)",
		googleUser.Email,
		username,
		"oauth_placeholder", // Password placeholder
		googleUser.ID,
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
