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
        errors := make(map[string]string)

        if identifier == "" {
            errors["identifier"] = "Field cannot be empty"
        }
        if password == "" {
            errors["password"] = "Field cannot be empty"
        }

        if len(errors) > 0 {
            tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/login.html", "web/templates/sidebar.html"))
            if err := tmpl.Execute(w, errors); err != nil {
                log.Println(err)
            }
            return
        }

        var storedHash, userID string
        query := `SELECT user_id, password FROM users WHERE email = ? OR username = ?`
        err := db.DB.QueryRow(query, identifier, identifier).Scan(&userID, &storedHash)
        if err == sql.ErrNoRows {
            errors["password"] = "Invalid username or password"
        } else if err != nil {
            log.Printf("Database query error: %v", err)
            utils.DisplayError(w, http.StatusInternalServerError, "Server error")
            return
        } else if bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) != nil {
            errors["password"] = "Invalid username or password"
        }

        if len(errors) > 0 {
            tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/login.html", "web/templates/sidebar.html"))
            if err := tmpl.Execute(w, errors); err != nil {
                log.Println(err)
            }
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

        username := strings.TrimSpace(r.FormValue("username"))
        email := strings.TrimSpace(r.FormValue("email"))
        password := r.FormValue("password")
        confirmPass := r.FormValue("confirmpassword")

        errors := make(map[string]string)

        if username == "" {
            errors["username"] = "Username is required"
        }
        if email == "" {
            errors["email"] = "Email is required"
        }
        if password == "" {
            errors["password"] = "Password is required"
        }
        if confirmPass == "" {
            errors["confirmpassword"] = "Please confirm your password"
        }
        if password != confirmPass {
            errors["confirmpassword"] = "Passwords do not match"
        }

        var exists bool
        err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email = ? OR username = ?)`, email, username).Scan(&exists)
        if err != nil && err != sql.ErrNoRows {
            log.Printf("Database query error: %v", err)
            utils.DisplayError(w, http.StatusInternalServerError, "Server error")
            return
        }
        if exists {
            errors["username"] = "Username or email already exists"
        }

        if len(errors) > 0 {
            tmpl := template.Must(template.ParseFiles("web/templates/layout.html", "web/templates/register.html", "web/templates/sidebar.html"))
            if err := tmpl.Execute(w, errors); err != nil {
                log.Println(err)
            }
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
