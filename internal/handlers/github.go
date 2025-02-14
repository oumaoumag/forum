package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"forum/internal/db"
	"forum/internal/utils"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
    githubOAuthConfig *oauth2.Config
    oauthStateString  = "random" // Replace with a more secure method in production
)

func init() {
    // Load environment variables from the .env file
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }

    githubClientID := os.Getenv("GITHUB_CLIENT_ID")
    githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
    redirectURL := os.Getenv("GITHUB_REDIRECT_URL")

    // Check if any variable is empty
    if githubClientID == "" || githubClientSecret == "" || redirectURL == "" {
        log.Fatalf("Error: GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET, and GITHUB_REDIRECT_URL environment variables must be set.")
    }

    // Log the values of the environment variables to ensure they're correctly loaded
    log.Printf("GITHUB_CLIENT_ID: %s", githubClientID)
    log.Printf("GITHUB_CLIENT_SECRET: %s", githubClientSecret)
    log.Printf("GITHUB_REDIRECT_URL: %s", redirectURL)

    // Initialize GitHub OAuth Config
    githubOAuthConfig = &oauth2.Config{
        ClientID:     githubClientID,
        ClientSecret: githubClientSecret,
        RedirectURL:  redirectURL,
        Scopes: []string{
            "user:email", // Request access to the user's email address
        },
        Endpoint: github.Endpoint,
    }

    // Check if the oauthConfig was initialized correctly
    if githubOAuthConfig == nil {
        log.Fatal("Error: GitHub OAuth configuration is nil")
    } else {
        log.Println("GitHub OAuth configuration initialized successfully")
    }
}


// GitHubLoginHandler redirects the user to GitHub's OAuth consent screen.
func GitHubLoginHandler(w http.ResponseWriter, r *http.Request) {
    if githubOAuthConfig == nil {
        utils.DisplayError(w, http.StatusInternalServerError, "GitHub OAuth not configured")
        return
    }

    url := githubOAuthConfig.AuthCodeURL(oauthStateString)
    http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}


// GitHubCallbackHandler handles the callback from GitHub OAuth.
func GitHubCallbackHandler(w http.ResponseWriter, r *http.Request) {
    if githubOAuthConfig == nil {
        utils.DisplayError(w, http.StatusInternalServerError, "GitHub OAuth not configured")
        return
    }

    state := r.FormValue("state")
    if state != oauthStateString {
        log.Printf("Invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }

    code := r.FormValue("code")
    token, err := githubOAuthConfig.Exchange(context.Background(), code)
    if err != nil {
        log.Printf("oauthConf.Exchange() failed with '%s'\n", err)
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }

    resp, err := http.Get("https://api.github.com/user/emails?access_token=" + token.AccessToken)
    if err != nil {
        log.Printf("Failed to get user info: %s", err.Error())
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }
    defer resp.Body.Close()

    var emails []struct {
        Email   string `json:"email"`
        Primary bool   `json:"primary"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
        log.Printf("Failed to decode user info: %s", err.Error())
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }

    // Extract the primary email
    var userEmail string
    for _, email := range emails {
        if email.Primary {
            userEmail = email.Email
            break
        }
    }

    // Find or create user in the database
    userID, err := findOrCreateUser(userEmail, "GitHub User") // User name could be retrieved from GitHub API as well.
    if err != nil {
        log.Printf("Failed to find or create user: %s", err.Error())
        utils.DisplayError(w, http.StatusInternalServerError, "Failed to authenticate")
        return
    }

    // Delete any existing session for the user (enforcing single-session authentication)
    _, err = db.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
    if err != nil {
        utils.DisplayError(w, http.StatusInternalServerError, "Server error")
        log.Printf("Session delete error: %v", err)
        return
    }

    // Create a session
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

func findOrCreateUser(email, name string) (string, error) {
    var userID string
    err := db.DB.QueryRow(`SELECT user_id FROM users WHERE email = ?`, email).Scan(&userID)

    if err == sql.ErrNoRows {
        // User doesn't exist, create a new user
        _, err := db.DB.Exec(`INSERT INTO users (username, email, password) VALUES (?, ?, ?)`, name, email, "") // Password can be empty for GitHub users

        if err != nil {
            return "", fmt.Errorf("failed to create user: %w", err)
        }

        // Get the newly created user's ID
        err = db.DB.QueryRow(`SELECT user_id FROM users WHERE email = ?`, email).Scan(&userID)
        if err != nil {
            return "", fmt.Errorf("failed to retrieve user ID after creation: %w", err)
        }
        return userID, nil

    } else if err != nil {
        return "", fmt.Errorf("failed to query user: %w", err)
    }
    return userID, nil
}

// Example Usage
// http.HandleFunc("/login/github", handlers.GitHubLoginHandler)
// http.HandleFunc("/oauth2/callback/github", handlers.GitHubCallbackHandler)