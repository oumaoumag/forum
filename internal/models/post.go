package models

import "time"

type Post struct {
	PostID       int
	UserID       int
	Title        string
	Content      string
	Username     string
	Category     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Comments     []Comment
	LikeCount    int
	DislikeCount int
}
