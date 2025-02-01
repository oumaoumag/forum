package models

type Post struct {
	PostID       int
	Title        string
	Content      string
	Username     string
	Category     string
	CreatedAt    string
	UpdatedAt    string
	Comments     []Comment
	LikeCount    int
	DislikeCount int
}
