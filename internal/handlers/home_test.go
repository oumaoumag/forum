package handlers

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"forum/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestHomeDB(t *testing.T) *sql.DB {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	db.DB = database

	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			user_id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE
		);`,
		`CREATE TABLE IF NOT EXISTS categories (
			category_id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);`,
		`CREATE TABLE IF NOT EXISTS posts (
			post_id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			category_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(user_id),
			FOREIGN KEY (category_id) REFERENCES categories(category_id)
		);`,
	}

	for _, query := range queries {
		_, err = database.Exec(query)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
	}

	return database
}

func insertTestHomeData(t *testing.T, database *sql.DB) {
	_, err := database.Exec(`INSERT INTO users (username) VALUES (?)`, "testuser")
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	_, err = database.Exec(`INSERT INTO categories (name) VALUES (?)`, "Test Category")
	if err != nil {
		t.Fatalf("Failed to insert category: %v", err)
	}

	_, err = database.Exec(`INSERT INTO posts (user_id, category_id, title, content) VALUES (?, ?, ?, ?)`, 1, 1, "Test Post", "This is a test post")
	if err != nil {
		t.Fatalf("Failed to insert post: %v", err)
	}
}

func TestHomeHandler(t *testing.T) {
	db := setupTestHomeDB(t)
	defer db.Close()
	insertTestHomeData(t, db)

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}

	w := httptest.NewRecorder()
	HomeHandler(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK; got %v", w.Code)
	}
}
