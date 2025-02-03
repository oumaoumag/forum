package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"forum/internal/db"
	"forum/internal/models"
)

// LikeHandler handles liking and disliking of posts and comments
func LikeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON request
	var req models.LikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.UserID == 0 || req.LikeType == "" || (req.PostID == nil && req.CommentID == nil) {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if req.LikeType != "like" && req.LikeType != "dislike" {
		http.Error(w, "Invalid like type", http.StatusBadRequest)
		return
	}

	// Determine whether it's a post or a comment
	var existingLikeType string
	query := `SELECT like_type FROM likes WHERE user_id = ? AND post_id IS ? AND comment_id IS ?`
	err := db.DB.QueryRow(query, req.UserID, req.PostID, req.CommentID).Scan(&existingLikeType)

	if err == sql.ErrNoRows {
		// No existing like/dislike → Insert a new reaction
		insertQuery := `INSERT INTO likes (user_id, post_id, comment_id, like_type) VALUES (?, ?, ?, ?)`
		_, err = db.DB.Exec(insertQuery, req.UserID, req.PostID, req.CommentID, req.LikeType)
		if err != nil {
			http.Error(w, "Failed to insert like", http.StatusInternalServerError)
			return
		}
		// w.WriteHeader(http.StatusCreated)
		// json.NewEncoder(w).Encode(map[string]string{"message": "Reaction added"})
		// return
	} else if err == nil {
		// If the user already reacted
		if existingLikeType == req.LikeType {
			// User clicked the same reaction → Remove it (unlike/undislike)
			deleteQuery := `DELETE FROM likes WHERE user_id = ? AND post_id IS ? AND comment_id IS ?`
			_, err = db.DB.Exec(deleteQuery, req.UserID, req.PostID, req.CommentID)
			if err != nil {
				http.Error(w, "Failed to remove like", http.StatusInternalServerError)
				return
			}
		} else {
			// User clicked opposite reaction → Toggle it (like ↔ dislike)
			updateQuery := `UPDATE likes SET like_type = ? WHERE user_id = ? AND post_id IS ? AND comment_id IS ?`
			_, err = db.DB.Exec(updateQuery, req.LikeType, req.UserID, req.PostID, req.CommentID)
			if err != nil {
				http.Error(w, "Failed to toggle reaction", http.StatusInternalServerError)
				return
			}
		}
	} else {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var likes, dislikes int
	countQuery := `
		SELECT 
			SUM(CASE WHEN like_type = 'like' THEN 1 ELSE 0 END) AS likes,
			SUM(CASE WHEN like_type = 'dislike' THEN 1 ELSE 0 END) AS dislikes
		FROM likes
		WHERE post_id IS ? AND comment_id IS ?`
	err = db.DB.QueryRow(countQuery, req.PostID, req.CommentID).Scan(&likes, &dislikes)
	if err != nil {
		http.Error(w, "Failed to fetch updated counts", http.StatusInternalServerError)
		return
	}

	// Return updated counts and user reaction status
	response := map[string]interface{}{
		"message":      "Reaction updated",
		"likes":        likes,
		"dislikes":     dislikes,
		"userReaction": req.LikeType,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
