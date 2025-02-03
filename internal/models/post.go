package models

import "time"

type Post struct {
	PostID       int
	UserID       int
	Title        string
	Content      string
	Username     string
	Category     string
	CreatedAt    string
	UpdatedAt    time.Time
	Comments     []Comment
	LikeCount    int
	DislikeCount int
	CommentCount int
}
