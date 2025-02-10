package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"forum/internal/auth"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// insertTestData inserts sample data into the database for testing
func insertTestData(t *testing.T, db *sql.DB) {
	// Isert a test user
	_, err := db.Exec(`INSERT INTO users (username, email, password) VALUES (?,?,?)`, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	// Insert a test cstegory
	_, err = db.Exec(`INSERT INTO categories (name) VALUES ( ?)`, "Test Category")
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	// Insert a test post
	_, err = db.Exec(`INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?)`, 1, "Test Post", "This is a test post")
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
