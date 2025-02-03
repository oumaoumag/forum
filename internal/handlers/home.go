package handlers

import (
	"html/template"
	"log"
	"net/http"

	"forum/internal/auth"
	"forum/internal/db"
	"forum/internal/models"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {

	currentUserID := auth.GetCurrentUserID(r)

	// Get filter query parameters
	categoryFilter := r.URL.Query().Get("category")
	createdFilter := r.URL.Query().Get("created")
	likedFilter := r.URL.Query().Get("liked")

	query := `
		SELECT p.post_id, p.tittle, p.content, u.username, u.user_id, c.name AS category, p.created_at,
		       (SELECT COUNT(*) FROM likes WHERE post_id = p.post_id AND comment_id IS NULL AND like_type = 'like') AS like_count,
		       (SELECT COUNT(*) FROM likes WHERE post_id = p.post_id AND comment_id IS NULL AND like_type = 'dislike') AS dislike_count
		FROM posts p
		JOIN users u ON p.user_id = u.user_id
		JOIN categories c ON p.category_id = c.category_id`

	// Prepare slices for query conditions and parameters
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
		err := rows.Scan(&post.PostID, &post.Title, &post.Username, &post.Category, &post.CreatedAt, &post.LikeCount, &post.DislikeCount)
		if err != nil {
			http.Error(w, "Error scanning posts", http.StatusInternalServerError)
			return
		}

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
			err := commentRows.Scan(&comment.CommentID, &comment.PostID, &comment.Content, &comment.Username, &comment.UserID, &comment.CreatedAt, &comment, &comment.DislikeCount)
			if err != nil {
				http.Error(w, "Error scanning comments", http.StatusInternalServerError)
				return
			}
			comments = append(comments, comment)
		}
		commentRows.Close()

		post.Comments = comments
		posts = append(posts, post)
	}

	data := struct {
		Posts []models.Post
		CurrentUserID int
	}{
		Posts: posts,
		CurrentUserID: currentUserID,
	}

	tmpl := template.Must(template.ParseFiles("../web/templates/layout.html", "../web/templates/home.html"))

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Unable to render template", http.StatusInternalServerError)
		return
	}
}
