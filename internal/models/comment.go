package models

import "time"

type Comment struct {
	CommentID    int       `db:"comment_id"`
	PostID       int       `db:"post_id"`
	UserID       int       `db:"user_id"`
	Content      string    `db:"content"`
	Username     string    `db:"user_name"`
	CreatedAt    string    `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
	LikeCount    int
	DislikeCount int
}
