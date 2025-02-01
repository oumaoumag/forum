package db

import (
	"database/sql"
	"io"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init() {
	var err error

	DB, err = sql.Open("sqlite3", "./forum.db")
	if err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}
	createTables()
	createCategories()
}

func createTables() {
	sqlFile, err := os.Open("../internal/db/schema.sql")
	if err != nil {
		log.Fatalf("failed to open the chema file: %v\n", err)
	}
	defer sqlFile.Close()

	sqlBytes, err := io.ReadAll(sqlFile)
	if err != nil {
		log.Fatalf("Failed to read the schema file: %v\n", err)
	}
	sqlStatements := string(sqlBytes)

	if _, err := DB.Exec(sqlStatements); err != nil {
		log.Fatalf("Failed to execute statements: %v\nQuery: %s\n", err, sqlStatements)
	}
	log.Println("All tables created successfully")
}

func createCategories() {
	predefinedCategories := []struct {
		Name        string
		Description string
	}{
		{"Technology", "Posts related to the latest technologies and trends"},
		{"Health", "Discussions about health, fitmess and well being"},
		{"Education", "Topics about learning and education"},
		{"Entertainment", "Movies, music, games and all things fun"},
		{"Lifestyle", "Fashion, home decore, and daily living tips."},
		{"Travel", "Exploring the world and sharing travel experience"},
	}
	for _, category := range predefinedCategories {
		_, err := DB.Exec(`INSERT OR IGNORE INTO categories (name, description) VALUES (?, ?)`, category.Name, category.Description)
		if err != nil {
			log.Printf("Error inserting category '%s': '%v'", category.Name, err)
		}
	}
}

func CleanupExpiredSession() {
	query := `DELETE FROM session WHERE expires_at < ?`
	_, err := DB.Exec(query, time.Now().Add(-24*time.Hour))
	if err != nil {
		log.Printf("Error cleaning up sessions: %v", err)
	}
}

func ScheduleSessionCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		CleanupExpiredSession()
	}
}
