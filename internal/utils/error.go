package utils

import (
	"log"
	"net/http"
	"text/template"
)

func DisplayError(w http.ResponseWriter, code int, message string) {
	data := struct {
		Code    int
		Message string
	}{
		Code:    code,
		Message: message,
	}

	tmpl, err := template.ParseFiles("web/templates/error.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}


	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
