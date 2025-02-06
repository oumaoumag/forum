package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"forum/internal/auth"
	"forum/internal/db"
	"forum/internal/utils" 
)

func CreateCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.DisplayError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	// retive userID from the request context
	userID, ok := auth.GetUserID(r)
	if !ok || userID == "" {
		// Redirect unauthenticated users to the login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form data
	postID, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil {
		utils.DisplayError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	content := r.FormValue("content")
	if content == "" || content == " " {
		utils.DisplayError(w, http.StatusBadRequest, "Content cannot be empty")
		return
	}

	// Insert comment into database
	_, err = db.DB.Exec("INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)", postID, userID, content)
	if err != nil {
		utils.DisplayError(w, http.StatusInternalServerError, "Failed to save comment")
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}
