package handlers

import (
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"forum/internal/auth"
	"forum/internal/db"
	"forum/internal/models"
	"forum/internal/utils"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		rxp, err := regexp.Compile(`/post/[\d]`)
		if err != nil {
			log.Println(err.Error())
			return
		}
		if !(r.URL.Path == "/" || rxp.MatchString(r.URL.Path)) {

			utils.DisplayError(w, http.StatusNotFound, " page not found")
			return
		}
	}

	currentUserID := auth.GetCurrentUserID(r)
	// Get filter query parameters
	categoryFilter := r.URL.Query().Get("category")
	createdFilter := r.URL.Query().Get("created")
	likedFilter := r.URL.Query().Get("liked")

	// Build base query
	query := `
    SELECT p.post_id, p.title, p.content, u.username, u.user_id,
		COALESCE(GROUP_CONCAT(DISTINCT c.name), '') AS categories,
		p.created_at,
        (SELECT COUNT(*) FROM likes WHERE post_id = p.post_id AND comment_id IS NULL AND like_type = 'like') AS like_count,
        (SELECT COUNT(*) FROM likes WHERE post_id = p.post_id AND comment_id IS NULL AND like_type = 'dislike') AS dislike_count,
        (SELECT COUNT(*) FROM comments WHERE post_id = p.post_id) AS total_comments
    FROM posts p
    JOIN users u ON p.user_id = u.user_id
	LEFT JOIN post_categories pc ON p.post_id = pc.post_id
	LEFT JOIN categories c ON pc.category_id = c.category_id
    LEFT JOIN likes l ON p.post_id = l.post_id`

	// Prepare a slice for query conditions and parameters
	conditions := []string{}
	params := []interface{}{}
	joins := []string{}

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
		joins = append(joins, `
        INNER JOIN likes lk 
        ON p.post_id = lk.post_id 
        AND lk.user_id = ? 
		AND lk.like_type = ?
        AND lk.comment_id IS NULL
    `)
		params = append(params, currentUserID, "like")
	}

	if len(joins) > 0 {
		query += " " + strings.Join(joins, " ")
	}

	// Append conditions if any
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Append order by clause
	query += " GROUP BY p.post_id ORDER BY p.created_at DESC"

	rows, err := db.DB.Query(query, params...)
	if err != nil {
		log.Println(err)
		utils.DisplayError(w, http.StatusInternalServerError, "Unable to fetch posts") // Enhanced error handling
		return
	}
	defer rows.Close()

	var posts []models.Post

	for rows.Next() {
		var rawCategories string
		var post models.Post
		var created_at time.Time
		err := rows.Scan(&post.PostID, &post.Title, &post.Content, &post.Username, &post.UserID, &rawCategories, &created_at, &post.LikeCount, &post.DislikeCount, &post.CommentCount)
		if err != nil {
			log.Println(err)
			utils.DisplayError(w, http.StatusInternalServerError, "Error retrieving post data") // Enhanced error handling
			return
		}
		if rawCategories != "" {
			post.Categories = strings.Split(rawCategories, ", ")
		} else {
			post.Categories = []string{}
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
			log.Println(err)
			utils.DisplayError(w, http.StatusInternalServerError, "Unable to fetch comments")
			return
		}

		var comments []models.Comment
		for commentRows.Next() {
			var comment models.Comment
			var created_at time.Time

			err := commentRows.Scan(&comment.CommentID, &comment.PostID, &comment.Content, &comment.Username, &comment.UserID, &created_at, &comment.LikeCount, &comment.DislikeCount)
			if err != nil {
				log.Println(err)
				utils.DisplayError(w, http.StatusInternalServerError, "Error scanning comments")
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

	name, _ := db.GetUser(currentUserID)

	data := struct {
		Posts         []models.Post
		CurrentUserID int
		Categories    []models.Categories
		Name          string
	}{
		Posts:         posts,
		CurrentUserID: currentUserID,
		Categories:    categories,
		Name:          name,
	}

	tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/home.html", "web/templates/sidebar.html")
	if err != nil {
		log.Println(err)
		utils.DisplayError(w, http.StatusInternalServerError, "server error")
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Println(err)
		utils.DisplayError(w, http.StatusInternalServerError, "server error")
		return
	}
}
