package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"forum/internal/db"
	"forum/internal/utils"

	"github.com/google/uuid"

	"golang.org/x/crypto/bcrypt"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Render the login page
		tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/login.html"))
		err := tmpl.Execute(w, nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else if r.Method == http.MethodPost {
		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Unable to process form", http.StatusBadRequest)
			return
		}

		// Extract form fields
		identifier := r.FormValue("identifier") // Can be email or username
		password := r.FormValue("password")

		// Validate inputs
		if identifier == "" || password == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		// Check if the user exist in the database
		var storedHash, userID string
		query := `SELECT user_id, password FROM users WHERE email = ? OR username = ?`
		err := db.DB.QueryRow(query, identifier, identifier).Scan(&userID, &storedHash)
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid email/username or password", http.StatusUnauthorized)
			return
		}

		// Compare the provided password with the stored hash
		if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
			http.Error(w, "Invalid email/username or password", http.StatusUnauthorized)
			return
		}

		// Create a session
		sessionID := uuid.New().String()
		expiration := time.Now().Add(24 * time.Hour) // 1-day session expiration
		insertQuery := `INSERT INTO sessions (session_id, user_id, expires_at) VALUES (?, ?, ?)`
		_, err = db.DB.Exec(insertQuery, sessionID, userID, expiration)
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			log.Printf("Database insert error: %v", err)
			return
		}

		// Set a session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true, // Prevent JavaScript access
		})

		// Redeirect to the homepage or dashboard
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Render the register page
		tmpl := template.Must(template.ParseFiles("../web/templates/layout.html", "../web/templates/register.html"))
		err := tmpl.Execute(w, nil)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else if r.Method == http.MethodPost {
		// Parse from data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		// Extract form fields
		username := strings.TrimSpace(r.FormValue("username"))
		email := strings.TrimSpace(r.FormValue("email"))
		password := r.FormValue("password")

		// Validate form data
		if username == "" || email == "" || password == "" {
			http.Error(w, "Please fill in all fields", http.StatusBadRequest)
			return
		}
		if !utils.ValidatePassword(password) {
			http.Error(w, "Password must be at least 6 characters long, contain one uppercase, lowercase, digit and specialist character", http.StatusBadRequest)
			return
		}

		// Check if email or username already exists
		var exists bool
		query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = ? OR username = ?)`
		err := db.DB.QueryRow(query, email, username).Scan(&exists)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Server error", http.StatusInternalServerError)
			log.Printf("Database query error: %v\n", err)
			return
		}
		if exists {
			http.Error(w, "Email or Usernamer already in use", http.StatusBadRequest)
			return
		}

		// Hash the password
		hashedPassword, err := utils.HashPassword(password)
		if err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			log.Printf("Password hashing error: %v\n", err)
			return
		}

		// Insert the new user into the database
		insertQuery := `INSERT INTO usrs (username, email, password) VALUES(?, ?,?)`
		_, err = db.DB.Exec(insertQuery, username, email, hashedPassword)
		if err != nil {
			http.Error(w, "Failed to register user", http.StatusInternalServerError)
			log.Printf("Database insert error: %v\n", err)
			return
		}
		// Redirect the login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the session cookie
	cookie, err := r.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		// Delete the session from the database
		query := `DELETE FROM sessions WHERE session_id = ?`
		_, err := db.DB.Exec(query, cookie.Value)
		if err != nil {
			log.Printf("Error deleting sessions: %v", err)
		}
	}

	// Clear the session cookie regardless of db operation success
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // Set expriry in the past
		Path:     "/",
		HttpOnly: true,
	})

	// Redirect to the login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
