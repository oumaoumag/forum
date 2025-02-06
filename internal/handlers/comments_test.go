package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"forum/internal/auth"
	"forum/internal/db"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// setupTestDB initializes an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
//	var database *sql.DB

	// Open an in-memory SQLite databse
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	// set up global db insrance
	db.DB = database

	// create tables
	queries := []string{
	`CREATE TABLE IF NOT EXISTS testUsers (
		user_id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		profile_picture TEXT,
		bio TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS categories (
		post_id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS posts (
		post_id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		category_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES testUsers(user_id) ON DELETE CASCADE,
		FOREIGN KEY (category_id) REFERENCES categories(category_id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS comments (
				comment_id INTEGER PRIMARY KEY AUTOINCREMENT,
				post_id INTEGER NOT NULL,
				user_id INTEGER NOT NULL,
				content TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (post_id) REFERENCES posts(post_id) ON DELETE CASCADE,
				FOREIGN KEY (user_id) REFERENCES testUsers(user_id) ON DELETE CASCADE
);`,
}

for _,query := range queries {
	_, err = database.Exec(query)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
}
}
return database
}

// insertTestData inserts sample data into the database for testing
func insertTestData(t *testing.T, db *sql.DB) {
	// Isert a test user
	_, err := db.Exec(`INSERT INTO testUsers (username, email, password) VALUES (?,?,?)`, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	// Insert a test cstegory
	_, err = db.Exec(`INSERT INTO categories (name, description) VALUES (?, ?)`, "Test Category", "This is a test category")
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	// Insert a test post
	_, err = db.Exec(`INSERT INTO posts (user_id, category_id, title, content) VALUES (?, ?, ?, ?)`, 1, 1, "Test Post", "This is a test post")
	if err != nil {
		t.Fatalf("Failed to insert post: %v", err)
}
}

func TestCreateCommentHandler(t *testing.T) {
	// Set up the in-memory database
	db := setupTestDB(t)
	defer db.Close()

	// Insert test data
	insertTestData(t, db)

	// Test cases
	tests := []struct {
		name           string
		method         string
		userID         string
		postID         string
		content        string
		expectedStatus int
		expectComment  bool // New field to indicate if a comment should exist
	}{
		{
			name:           "Valid request",
			method:         http.MethodPost,
			userID:         "1",
			postID:         "1",
			content:        "This is a test comment",
			expectedStatus: http.StatusSeeOther,
			expectComment:  true, // Comment should be inserted
		},
		{
			name:           "Invalid method",
			method:         http.MethodGet,
			userID:         "1",
			postID:         "1",
			content:        "This is a test comment",
			expectedStatus: http.StatusMethodNotAllowed,
			expectComment:  false,
		},
		{
			name:           "Unauthorized user",
			method:         http.MethodPost,
			userID:         "",
			postID:         "1",
			content:        "This is a test comment",
			expectedStatus: http.StatusSeeOther,
			expectComment:  false, // No comment should be inserted
		},
		{
			name:           "Invalid post ID",
			method:         http.MethodPost,
			userID:         "1",
			postID:         "invalid",
			content:        "This is a test comment",
			expectedStatus: http.StatusBadRequest,
			expectComment:  false,
		},
		{
			name:           "Empty content",
			method:         http.MethodPost,
			userID:         "1",
			postID:         "1",
			content:        "",
			expectedStatus: http.StatusBadRequest,
			expectComment:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a form with the necessary data
			form := url.Values{}
			form.Add("post_id", tt.postID)
			form.Add("content", tt.content)

			// Create a request with the form data
			req, err := http.NewRequest(tt.method, "/comment", strings.NewReader(form.Encode()))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

			// Mock authentication (only if userID is provided)
			if tt.userID != "" {
				req = auth.SetUserID(req, tt.userID)
			}

			// Record the response
			rr := httptest.NewRecorder()
			CreateCommentHandler(rr, req)

			// Check the status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check for comment insertion ONLY if expectComment is true
			if tt.expectComment {
				var commentCount int
				err := db.QueryRow(
					"SELECT COUNT(*) FROM comments WHERE post_id = ? AND user_id = ? AND content = ?",
					tt.postID, tt.userID, tt.content,
				).Scan(&commentCount)
				if err != nil {
					t.Fatalf("Failed to query database: %v", err)
				}
				if commentCount != 1 {
					t.Errorf("expected 1 comment, got %d", commentCount)
				}
			}
		})
	}
}