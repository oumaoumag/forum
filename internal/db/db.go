package db

import (
	"database/sql"
	"io"
	"log"
	"os"
)

var DB *sql.DB

func Init() {
	var err error

	DB, err := sql.Open("sqlite3", "./forum.db")
	if err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}
}

func createTables() {
	sqlFile, err := os.Open("internal/db/schema.sql")
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
