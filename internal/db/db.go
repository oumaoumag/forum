package db

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3" // Import SQLite3 driver
)

var DB *sql.DB // Global variable to hold the database connection

// Init initializes the database and returns any error encountered.
func Init(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	if err = createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	if err = createCategories(); err != nil {
		return fmt.Errorf("failed to create categories: %v", err)
	}

	return nil
}

// createTables executes the schema.sql file and returns any error.
func createTables() error {
	sqlFile, err := os.Open("internal/db/schema.sql")
	if err != nil {
		return fmt.Errorf("failed to open schema file: %v", err)
	}
	defer sqlFile.Close()

	sqlBytes, err := io.ReadAll(sqlFile)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %v", err)
	}

	// Execute the SQL statements. If an error occurs, log it and terminate the program.
	if _, err := DB.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("failed to execute schema: %v", err)
	}

	return nil
}

// createCategories inserts predefined categories, returns error on failure.
func createCategories() error {
	categories := []struct {
		Name, Description string
	}{
		{"Technology", "Posts related to the latest technology and trends"},
		{"Health", "Discussions about health, fitness, and well-being"},
		{"Education", "Topics about learning and education"},
		{"Entertainment", "Movies, music, games, and all things fun"},
		{"Lifestyle", "Fashion, home decor, and daily living tips"},
		{"Travel", "Exploring the world, sharing travel experiences"},
	}

	for _, c := range categories {
		_, err := DB.Exec(`INSERT OR IGNORE INTO categories (name, description) VALUES (?, ?)`, c.Name, c.Description)
		if err != nil {
			return fmt.Errorf("error inserting category '%s': '%v'", c.Name, err)
		}
	}
	return nil
}

// CleanupExpiredSessions deletes sessions older than 24 hours.
func CleanupExpiredSessions() error {
	query := `DELETE FROM sessions WHERE expires_at < ?`
	_, err := DB.Exec(query, time.Now().Add(-24*time.Hour)) // Allocating 24-hour session duration
	
	return err
}

// ScheduleSessionCleanup starts a ticker to periodically clean up sessions. 
func ScheduleSessionCleanup(interval time.Duration, cleanupFunc func() error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := cleanupFunc(); err != nil {
			log.Printf("error: session cleanup failed: %v", err)
		}
	}
}
