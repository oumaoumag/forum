package models

type LikeRequest struct {
	UserID    int    `json:"user_id"`
	PostID    *int   `json:"post_id,omitempty"`
	CommentID *int   `json:"comment_id,omitempty"`
	LikeType  string `json:"like_type"`
}
