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
		tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/login.html", "web/templates/sidebar.html"))
		if err := tmpl.Execute(w, nil); err != nil {
			log.Println(err)
			utils.DisplayError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			utils.DisplayError(w, http.StatusBadRequest, "Unable to process form")
			return
		}

		identifier, password := r.FormValue("identifier"), r.FormValue("password")
		if identifier == "" || password == "" {
			utils.DisplayError(w, http.StatusBadRequest, "All fields are required")
			return
		}

		var storedHash, userID string
		query := `SELECT user_id, password FROM users WHERE email = ? OR username = ?`
		err := db.DB.QueryRow(query, identifier, identifier).Scan(&userID, &storedHash)
		if err == sql.ErrNoRows {
			utils.DisplayError(w, http.StatusUnauthorized, "Invalid email/username or password")
			return
		} else if err != nil {
			log.Printf("Database query error: %v", err)
			utils.DisplayError(w, http.StatusInternalServerError, "Server error")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err != nil {
			utils.DisplayError(w, http.StatusUnauthorized, "Invalid email/username or password")
			return
		}

		_, err = db.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
		if err != nil {
			log.Printf("Database delete error: %v", err)
			utils.DisplayError(w, http.StatusInternalServerError, "Failed to clear old sessions")
			return
		}

		sessionID := uuid.New().String()
		expiration := time.Now().Add(24 * time.Hour)
		_, err = db.DB.Exec(`INSERT INTO sessions (session_id, user_id, expires_at) VALUES (?, ?, ?)`, sessionID, userID, expiration)
		if err != nil {
			log.Printf("Database insert error: %v", err)
			utils.DisplayError(w, http.StatusInternalServerError, "Failed to create session")
			return
		}

		http.SetCookie(w, &http.Cookie{Name: "session_id", Value: sessionID, Expires: expiration, Path: "/", HttpOnly: true})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/register.html", "web/templates/sidebar.html"))
		if err := tmpl.Execute(w, nil); err != nil {
			log.Println(err)
			utils.DisplayError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			utils.DisplayError(w, http.StatusBadRequest, "Failed to parse form")
			return
		}

		username, email, password, confirmPass := strings.TrimSpace(r.FormValue("username")), strings.TrimSpace(r.FormValue("email")), r.FormValue("password"), r.FormValue("confirmpassword")
		if username == "" || email == "" || password == "" {
			utils.DisplayError(w, http.StatusBadRequest, "Please fill in all fields")
			return
		}
		if password != confirmPass {
			utils.DisplayError(w, http.StatusBadRequest, "Password and confirm password do not match")
			return
		}
		if !utils.ValidatePassword(password) {
			utils.DisplayError(w, http.StatusBadRequest, "Password must be at least 6 characters long, contain one uppercase, lowercase, digit, and special character")
			return
		}

		var exists bool
		err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email = ? OR username = ?)`, email, username).Scan(&exists)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Database query error: %v", err)
			utils.DisplayError(w, http.StatusInternalServerError, "Server error")
			return
		}
		if exists {
			utils.DisplayError(w, http.StatusBadRequest, "Email or Username already in use")
			return
		}

		hashedPassword, err := utils.HashPassword(password)
		if err != nil {
			log.Printf("Password hashing error: %v", err)
			utils.DisplayError(w, http.StatusInternalServerError, "Server error")
			return
		}

		_, err = db.DB.Exec(`INSERT INTO users (username, email, password) VALUES (?, ?, ?)`, username, email, hashedPassword)
		if err != nil {
			log.Printf("Database insert error: %v", err)
			utils.DisplayError(w, http.StatusInternalServerError, "Failed to register user")
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.DisplayError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	cookie, err := r.Cookie("session_id")
	if err == nil && cookie.Value != "" {
		_, err = db.DB.Exec(`DELETE FROM sessions WHERE session_id = ?`, cookie.Value)
		if err != nil {
			log.Printf("Error deleting session: %v", err)
		}
	}

	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", Expires: time.Now().Add(-time.Hour), Path: "/", HttpOnly: true})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
