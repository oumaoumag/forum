package handlers

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"forum/internal/auth"
	"forum/internal/db"
	"forum/internal/models"
	"forum/internal/utils"
)

func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/post/create" {
		utils.DisplayError(w, http.StatusNotFound, " page not found")
		return
	}
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
		userDetails, _ := db.GetUser(currentUserID)

		data := struct {
			CurrentUserID int
			Categories    []models.Categories
			Name          string
		}{
			CurrentUserID: currentUserID,
			Categories:    categories,
			Name:          userDetails[0],
		}

		// Render the form
		tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/post.html", "web/templates/sidebar.html", "web/templates/profile.html")
		if err != nil {
			utils.DisplayError(w, http.StatusInternalServerError, "server error")
			return
		}
		if err = tmpl.Execute(w, data); err != nil {
			utils.DisplayError(w, http.StatusInternalServerError, "server error")
			return
		}
	} else if r.Method == http.MethodPost {
		// Parse form input
		err := r.ParseMultipartForm(20)
		if err != nil {
			utils.DisplayError(w, http.StatusBadRequest, "Invalid form data")

			return
		}

		title := r.FormValue("title")
		content := r.FormValue("content")
		categories := r.Form["category"]
		file, headers, err := r.FormFile("img")
		imgurl := ""

		if err == nil {
			dst, err := os.Create(filepath.Join("web/static/images", headers.Filename))
			if err != nil {
				utils.DisplayError(w, http.StatusInternalServerError, "server error")
				return
			}
			_, err = io.Copy(dst, file)
			if err != nil {
				utils.DisplayError(w, http.StatusInternalServerError, "server error")
				return
			}
			imgurl = dst.Name()
			imgurl = strings.Replace(imgurl, "web", "..", 1)

		}

		// Validate inputs
		if title == "" || content == "" {
			utils.DisplayError(w, http.StatusBadRequest, "All fields are required")
			return
		}
		if strings.TrimSpace(title) == "" || strings.TrimSpace(content) == "" {
			utils.DisplayError(w, http.StatusBadRequest, "Tittle or Content cannot be spaces")
			return
		}

		// Insert post into the database
		query := `
			INSERT INTO posts (user_id, title, content)
			VALUES (?, ?, ?)`
		if imgurl != "" {
			query = `
			INSERT INTO posts (user_id, title, content, imgurl)
			VALUES (?, ?, ?,?)`
		}
		result, err := db.DB.Exec(query, userID, title, content, imgurl)
		if err != nil {
			log.Println(err)
			utils.DisplayError(w, http.StatusInternalServerError, "Unable to create post")
			return
		}

		// Get the newly created post's ID
		postID, err := result.LastInsertId()
		if err != nil {
			utils.DisplayError(w, http.StatusInternalServerError, "Failed to retrieve post ID")
			return
		}

		// Insert each selected category into post_categories
		for _, catIDStr := range categories {
			catID, err := strconv.Atoi(catIDStr)
			if err != nil {
				utils.DisplayError(w, http.StatusBadRequest, "Invalid category ID: "+catIDStr)
				return
			}
			_, err = db.DB.Exec("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)", postID, catID)
			if err != nil {
				utils.DisplayError(w, http.StatusInternalServerError, "Failed to link category to post: "+err.Error())
				return
			}
		}

		// Redirect to homepage or posts page
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
