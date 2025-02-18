package models

import "time"

type Post struct {
	PostID       int
	UserID       int
	Title        string
	Content      string
	Username     string
	Categories   []string
	CreatedAt    string
	UpdatedAt    time.Time
	Comments     []Comment
	LikeCount    int
	DislikeCount int
	CommentCount int
	Imgurl string
}
