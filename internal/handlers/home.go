package handlers

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"forum/internal/auth"
	"forum/internal/db"
	"forum/internal/models"
	"forum/internal/utils"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	currentUserID := auth.GetCurrentUserID(r)

	// Get filter query parameters
	categoryFilter := r.URL.Query().Get("category")
	createdFilter := r.URL.Query().Get("created")
	likedFilter := r.URL.Query().Get("liked")

	// Build base query
	query := `
    SELECT p.post_id, p.title, p.content, u.username, u.user_id, c.name AS category, p.created_at,
        (SELECT COUNT(*) FROM likes WHERE post_id = p.post_id AND comment_id IS NULL AND like_type = 'like') AS like_count,
        (SELECT COUNT(*) FROM likes WHERE post_id = p.post_id AND comment_id IS NULL AND like_type = 'dislike') AS dislike_count,
        (SELECT COUNT(*) FROM comments WHERE post_id = p.post_id) AS total_comments
    FROM posts p
    JOIN users u ON p.user_id = u.user_id
    JOIN categories c ON p.category_id = c.category_id`

	// Prepare a slice for query conditions and parameters
	conditions := []string{}
	params := []interface{}{}

	// 1. Filter by category if set
	if categoryFilter != "" {
		conditions = append(conditions, "c.name = ?")
		params = append(params, categoryFilter)
	}

	// 2. Filter by created posts (only for registered users)
	if createdFilter == "true" && currentUserID != 0 {
		conditions = append(conditions, "p.user_id = ?")
		params = append(params, currentUserID)
	}

	// 3. Filter by liked posts (only for registered users)
	// We need to join the likes table to filter by posts that the user liked.
	if likedFilter == "true" && currentUserID != 0 {
		query += " JOIN likes l ON p.post_id = l.post_id "
		conditions = append(conditions, "l.user_id = ?")
		params = append(params, currentUserID)
	}

	// Append conditions if any
	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}

	}

	// Append order by clause
	query += " ORDER BY p.created_at DESC"

	rows, err := db.DB.Query(query, params...)
	if err != nil {
		log.Println(err)
		http.Error(w, "Unable to fetch posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		var created_at time.Time
		err := rows.Scan(&post.PostID, &post.Title, &post.Content, &post.Username, &post.UserID, &post.Category, &created_at, &post.LikeCount, &post.DislikeCount, &post.CommentCount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		post.CreatedAt = utils.FormatTime(created_at)

		// Fetch comments for each post
		commentQuery := `
			SELECT c.comment_id, c.post_id, c.content, u.username, u.user_id, c.created_at,
				(SELECT COUNT(*) FROM likes WHERE comment_id = c.comment_id AND like_type = 'like') AS like_count,
				(SELECT COUNT(*) FROM likes WHERE comment_id = c.comment_id AND like_type = 'dislike') AS dislike_count
			FROM comments c
			JOIN users u ON c.user_id = u.user_id
			WHERE c.post_id = ?
			ORDER BY c.created_at ASC`
		commentRows, err := db.DB.Query(commentQuery, post.PostID)
		if err != nil {
			log.Println(err)
			http.Error(w, "Unable to fetch comments", http.StatusInternalServerError)
			return
		}

		var comments []models.Comment
		for commentRows.Next() {
			var comment models.Comment
			var created_at time.Time

			err := commentRows.Scan(&comment.CommentID, &comment.PostID, &comment.Content, &comment.Username, &comment.UserID, &created_at, &comment.LikeCount, &comment.DislikeCount)
			if err != nil {
				http.Error(w, "Error scanning comments", http.StatusInternalServerError)
				return
			}
			comment.CreatedAt = utils.FormatTime(created_at)
			comments = append(comments, comment)
		}
		commentRows.Close()

		post.Comments = comments
		posts = append(posts, post)
	}

	categories := utils.FetchCategories()

	data := struct {
		Posts         []models.Post
		CurrentUserID int
		Categories    []models.Categories
	}{
		Posts:         posts,
		CurrentUserID: currentUserID,
		Categories:    categories,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/home.html", "web/templates/sidebar.html"))

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Unable to render template", http.StatusInternalServerError)
		return
	}
}
