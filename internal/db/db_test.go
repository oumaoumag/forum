package db

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const testDBPath = "file:test.db?mode=memory&cache=shared"

func TestMain(m *testing.M) {
	// Change working directory to project root to match the server's environment
	if err := os.Chdir("../.."); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change directory: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	os.Exit(m.Run())
}

func setupTestDB(t *testing.T) {
	var err error
	DB, err = sql.Open("sqlite3", testDBPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
}

func teardownTestDB() {
	DB.Close()
}

func TestInit(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	err := Init(testDBPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	rows, err := DB.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan table name: %v", err)
		}
		tables = append(tables, name)
	}

	if !contains(tables, "users") || !contains(tables, "categories") {
		t.Errorf("expected tables 'users' and 'categories' to exist, got %v", tables)
	}

	if err = Init("invalid://driver"); err == nil {
		t.Error("expected an error for invalid driver, got nil")
	}
}

func TestCreateTables(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	if err := createTables(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	schemaPath := "internal/db/schema111.sql"
	backupPath := schemaPath + ".bak"
	_ = os.Rename(schemaPath, backupPath)
	defer os.Rename(backupPath, schemaPath)

	if err := createTables(); err != nil {
		t.Error("expected an error due to missing schema file, got nil")
	}
}

func TestCreateCategories(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	if err := createCategories(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var count int
	if err := DB.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count); err != nil {
		t.Fatalf("failed to count categories: %v", err)
	}
	if count != 6 {
		t.Errorf("expected 6 categories, got %d", count)
	}

	if err := createCategories(); err != nil {
		t.Errorf("expected no error on duplicate insert, got %v", err)
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	expired := time.Now().Add(-48 * time.Hour)
	valid := time.Now().Add(24 * time.Hour)
	_, err := DB.Exec(`INSERT INTO sessions (session_id, user_id, expires_at) VALUES (?, 1, ?), (?, 1, ?)`,
		"expired", expired, "valid", valid)
	if err != nil {
		t.Fatalf("failed to insert test sessions: %v", err)
	}

	if err := CleanupExpiredSessions(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var remaining int
	if err := DB.QueryRow("SELECT COUNT(*) FROM sessions WHERE session_id = 'expired'").Scan(&remaining); err != nil {
		t.Fatalf("failed to check expired session: %v", err)
	}
	if remaining != 0 {
		t.Errorf("expected expired session to be deleted, got %d", remaining)
	}

	if err := DB.QueryRow("SELECT COUNT(*) FROM sessions WHERE session_id = 'valid'").Scan(&remaining); err != nil {
		t.Fatalf("failed to check valid session: %v", err)
	}
	if remaining != 1 {
		t.Errorf("expected valid session to remain, got %d", remaining)
	}
}

func TestScheduleSessionCleanup(t *testing.T) {
	cleanupTriggered := false
	mockCleanup := func() error {
		cleanupTriggered = true
		return nil
	}

	go ScheduleSessionCleanup(10*time.Millisecond, mockCleanup)
	time.Sleep(15 * time.Millisecond)
	if !cleanupTriggered {
		t.Error("expected cleanup to be triggered but it was not")
	}
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
