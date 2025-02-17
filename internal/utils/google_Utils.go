package utils

import (
	"encoding/json"
	"fmt"
	"forum/internal/models"
	"net/http"
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
