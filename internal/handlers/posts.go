package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"forum/internal/auth"
	"forum/internal/db"
	"forum/internal/models"
	"forum/internal/utils"
)

func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	currentUserID := auth.GetCurrentUserID(r)

	// Retrieve the user ID from the context
	userID, ok := auth.GetUserID(r)
	if !ok || userID == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		// Fetch categories to display in the form
		categories := utils.FetchCategories()

		data := struct {
			CurrentUserID int
			Categories []models.Categories
		}{
			CurrentUserID: currentUserID,
			Categories: categories,
		}
		
		// Render the form
		tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/post.html", "web/templates/sidebar.html")
		if err != nil {
			http.Error(w, "Unable to load template", http.StatusInternalServerError)
			return
		}
		if err = tmpl.Execute(w, data); err != nil {
			http.Error(w, "Error rendering template", http.StatusInternalServerError)
		}
	} else if r.Method == http.MethodPost {
		// Parse form input
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		content := r.FormValue("content")
		categoryID := r.FormValue("category")

		// Validate inputs
		if title == "" || content == "" || categoryID == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		// Convert categoryID to integer
		categoryIDInt, err := strconv.Atoi(categoryID)
		if err != nil {
			http.Error(w, "Invalid category", http.StatusBadRequest)
			return
		}

		// Insert post into the database
		query := `
			INSERT INTO posts (user_id, category_id, title, content)
			VALUES (?, ?, ?, ?)`
		_, err = db.DB.Exec(query, userID, categoryIDInt, title, content)
		if err != nil {
			http.Error(w, "Unable to create post", http.StatusInternalServerError)
			return
		}

		// Redirect to homepage or posts page
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
