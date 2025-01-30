package db

import (
	"database/sql"
	"log"
)

var DB *sql.DB


func Init() {
	var err error

	DB, err := sql.Open("sqlite3", "./forum.db")

	if err != nil {
		log.Fatalf("failed to connect to database: %v\n", err)
	}

}