package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"forum/internal/auth"
	"forum/internal/db"
)

func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve the user ID from the context
	userID, ok := auth.GetUserID(r)
	if !ok || userID == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if r.Method == http.MethodGet {
		// Fethc categoreies to display in the form
		rows, err := db.DB.Query("SELECT category_id, name FROM categories")
		if err != nil {
			http.Error(w, "Unable to fetch categories", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var categoreies []struct {
			categoryID int
			Name       string
		}

		for rows.Next() {
			var category struct {
				categoryID int
				Name       string
			}
			if err := rows.Scan(&category.categoryID, &category.Name); err != nil {
				http.Error(w, "Error scanning categories", http.StatusInternalServerError)
				return
			}
			categoreies = append(categoreies, category)
		}

		// Render the form
		tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/post.html")
		if err != nil {
			http.Error(w, "Unable to fetch categories", http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, categoreies)
	} else if r.Method == http.MethodPost {
		// Parse from input
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
