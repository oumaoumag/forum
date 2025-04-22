package utils

import (
	"log"

	"forum/internal/db"
	"forum/internal/models"
)

func FetchCategories() []models.Categories {
	rows, err := db.DB.Query("SELECT category_id, name FROM categories")
	if err != nil {
		log.Println("err executing query", err)

		return nil
	}
	defer rows.Close()

	var categories []models.Categories

	for rows.Next() {
		var category struct {
			CategoryID int
			Name       string
		}
		if err := rows.Scan(&category.CategoryID, &category.Name); err != nil {
			log.Println("Error scanning categories", err)
			return nil
		}
		categories = append(categories, category)
	}
	return categories
}
