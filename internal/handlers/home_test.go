package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"forum/internal/auth"

	_ "github.com/mattn/go-sqlite3"
)

func TestHomeHandler(t *testing.T) {
	// Setup in-memory database
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Insert test data
	insertHomeTestData(t, testDB)

	// Define test cases
	tests := []struct {
		name           string
		queryParams    url.Values
		currentUserID  string
		expectedStatus int
		expectedPosts  int
		expectedInHTML []string
		notInHTML      []string
	}{
		{
			name:           "No filters",
			queryParams:    url.Values{},
			currentUserID:  "",
			expectedStatus: http.StatusOK,
			expectedPosts:  1,
			expectedInHTML: []string{"Test Post"},
			notInHTML:      []string{},
		},
		{
			name:           "Filter by category",
			queryParams:    url.Values{"category": []string{"Test Category"}},
			currentUserID:  "",
			expectedStatus: http.StatusOK,
			expectedPosts:  1,
			expectedInHTML: []string{"Test Post"},
			notInHTML:      []string{"Another Post"},
		},
		{
			name:           "Filter by created (authenticated)",
			queryParams:    url.Values{"created": []string{"true"}},
			currentUserID:  "1",
			expectedStatus: http.StatusOK,
			expectedPosts:  1,
			expectedInHTML: []string{"Test Post"},
			notInHTML:      []string{"Another Post"},
		},
		{
			name:           "Filter by created (unauthenticated)",
			queryParams:    url.Values{"created": []string{"true"}},
			currentUserID:  "",
			expectedStatus: http.StatusOK,
			expectedPosts:  1, // Filter ignored, show all posts
			expectedInHTML: []string{"Test Post"},
			notInHTML:      []string{},
		},
		{
			name:           "Filter by liked (authenticated)",
			queryParams:    url.Values{"liked": []string{"true"}},
			currentUserID:  "1",
			expectedStatus: http.StatusOK,
			expectedPosts:  1,
			expectedInHTML: []string{"Liked Post"},
			notInHTML:      []string{"Test Post"},
		},
		{
			name:           "Invalid category filter",
			queryParams:    url.Values{"category": []string{"Invalid"}},
			currentUserID:  "",
			expectedStatus: http.StatusOK,
			expectedPosts:  0,
			expectedInHTML: []string{"No posts found"},
			notInHTML:      []string{"Test Post"},
		},
		{
			name:           "Check comments included",
			queryParams:    url.Values{},
			currentUserID:  "",
			expectedStatus: http.StatusOK,
			expectedPosts:  1,
			expectedInHTML: []string{"Test Comment"},
			notInHTML:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with query parameters
			req, err := http.NewRequest("GET", "/?"+tt.queryParams.Encode(), nil)
			if err != nil {
				t.Fatal(err)
			}

			// Set current user if provided
			if tt.currentUserID != "" {
				req = auth.SetUserID(req, tt.currentUserID)
			}

			// Record the response
			rr := httptest.NewRecorder()
			HomeHandler(rr, req)

			// Check status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			// Check response body content
			body := rr.Body.String()
			for _, content := range tt.expectedInHTML {
				if !strings.Contains(body, content) {
					t.Errorf("expected content %q not found in response", content)
				}
			}
			for _, content := range tt.notInHTML {
				if strings.Contains(body, content) {
					t.Errorf("unexpected content %q found in response", content)
				}
			}
		})
	}
}

// insertTestData inserts test data into the database
func insertHomeTestData(t *testing.T, db *sql.DB) {
	// Insert test user
	_, err := db.Exec(`INSERT INTO testUsers (username, email, password) VALUES (?, ?, ?)`, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatalf("Failed to insert test User: %v", err)
	}

	// Insert categories
	_, err = db.Exec(`INSERT INTO categories (name) VALUES ('Test Category'), ('Another Category')`)
	if err != nil {
		t.Fatalf("Failed to insert categories: %v", err)
	}

	// Insert posts
	_, err = db.Exec(`
		INSERT INTO posts (user_id, category_id, title, content) 
		VALUES (1, 1, 'Test Post', 'Test content'), 
			   (1, 2, 'Another Post', 'Another content'),
			   (1, 1, 'Liked Post', 'Liked content')`)
	if err != nil {
		t.Fatalf("Failed to insert posts: %v", err)
	}

	// Insert like for Liked Post
	_, err = db.Exec(`INSERT INTO likes (user_id, post_id, like_type) VALUES (1, 3, 'like')`)
	if err != nil {
		t.Fatalf("Failed to insert like: %v", err)
	}

	// Insert comment
	_, err = db.Exec(`INSERT INTO comments (post_id, user_id, content) VALUES (1, 1, 'Test Comment')`)
	if err != nil {
		t.Fatalf("Failed to insert comment: %v", err)
	}
}
