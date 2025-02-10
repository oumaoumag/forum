package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"forum/internal/auth"
	"forum/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

func TestMain(m *testing.M) {
	// Change working directory to project root to match the server's environment
	if err := os.Chdir("../.."); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change directory: %v\n", err)
		os.Exit(1)
	}

	exitCode := m.Run()
	if db.DB != nil {
		db.DB.Close()
	}

	// Run tests
	// os.Exit(m.Run())

	os.Exit(exitCode)
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite3", "file:testdb?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables matching your schema
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			user_id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE,
			email TEXT UNIQUE,
			password TEXT
		);

		CREATE TABLE IF NOT EXISTS posts (
			post_id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			title TEXT,
			content TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(user_id)
		);
		
		CREATE TABLE IF NOT EXISTS comments (
			comment_id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER,
			user_id INTEGER,
			content TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(post_id) REFERENCES posts(post_id),
			FOREIGN KEY(user_id) REFERENCES users(user_id)
		);
		
		CREATE TABLE IF NOT EXISTS categories (
			category_id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE
		);
		
		CREATE TABLE IF NOT EXISTS post_categories (
			post_id INTEGER,
			category_id INTEGER,
			PRIMARY KEY(post_id, category_id),
			FOREIGN KEY(post_id) REFERENCES posts(post_id),
			FOREIGN KEY(category_id) REFERENCES categories(category_id)
		);
		
		CREATE TABLE IF NOT EXISTS likes (
			like_id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			post_id INTEGER,
			comment_id INTEGER,
			like_type TEXT CHECK(like_type IN ('like', 'dislike')),
			FOREIGN KEY(user_id) REFERENCES users(user_id),
			FOREIGN KEY(post_id) REFERENCES posts(post_id),
			FOREIGN KEY(comment_id) REFERENCES comments(comment_id)
		);

		CREATE TABLE IF NOT EXISTS sessions (
			session_id TEXT PRIMARY KEY,
			user_id INTEGER,
			expires_at DATETIME,
			FOREIGN KEY(user_id) REFERENCES users(user_id)
		);
	`)
	if err != nil {
		t.Fatal("Failed to create tables:", err)
	}

	rows, _ := testDB.Query("SELECT name FROM sqlite_master WHERE type='table'")
	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Println("Table:", name)
	}

	db.DB = testDB // Override global DB connection
	t.Log("Db created Successfuly")
	return testDB
}

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
				t.Fatal("Be more specific ", err)
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
	_, err := db.Exec(`INSERT INTO users (username, email, password) VALUES (?, ?, ?)`, "testuser", "test@example.com", "password123")
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
		INSERT INTO posts (user_id, title, content) 
		VALUES (1,  'Test Post', 'Test content'), 
			   (1, 'Another Post', 'Another content'),
			   (1, 'Liked Post', 'Liked content')`)
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
