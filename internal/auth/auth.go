package auth

import (
	"net/http"

	"forum/internal/db"
)

// Add this in your auth handlers file (e.g., auth.go)
func GetCurrentUserID(r *http.Request) int {
	// Example implementation using session cookie
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return 0 // Not logged in
	}

	var userID int
	err = db.DB.QueryRow(`
        SELECT user_id FROM sessions 
        WHERE session_id = ? AND expires_at > datetime('now')
    `, cookie.Value).Scan(&userID)
	if err != nil {
		return 0 // Session invalid/expired
	}
	return userID
}
