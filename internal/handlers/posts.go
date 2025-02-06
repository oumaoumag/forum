package handlers

import (
	"html/template"
	"log"
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
	if !ok || userID == "" || userID == " " {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		// Fetch categories to display in the form
		categories := utils.FetchCategories()

		data := struct {
			CurrentUserID int
			Categories    []models.Categories
		}{
			CurrentUserID: currentUserID,
			Categories:    categories,
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
		categories := r.Form["category"]

		// Validate inputs
		if title == "" || content == "" || len(categories) == 0 {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}
		if title == " " || content == " "{
			http.Error(w, "Tittle or Content cannot be spaces", http.StatusBadRequest)
		}

		// Insert post into the database
		query := `
			INSERT INTO posts (user_id, title, content)
			VALUES (?, ?, ?)`
		result, err := db.DB.Exec(query, userID, title, content)
		if err != nil {
			log.Println(err)
			http.Error(w, "Unable to create post", http.StatusInternalServerError)
			return
		}

		// Get the newly created post's ID
		postID, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Failed to retrieve post ID", http.StatusInternalServerError)
			return
		}

		// Insert each selected category into post_categories
		for _, catIDStr := range categories {
			catID, err := strconv.Atoi(catIDStr)
			if err != nil {
				http.Error(w, "Invalid category ID: "+catIDStr, http.StatusBadRequest)
				return
			}
			_, err = db.DB.Exec("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)", postID, catID)
			if err != nil {
				http.Error(w, "Failed to link category to post: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Redirect to homepage or posts page
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
