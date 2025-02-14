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
    oauthStateString  = "random" 
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

    // Use Authorization header with Bearer token
    req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
    if err != nil {
        log.Printf("Failed to create request: %s", err.Error())
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }

    // Set the Authorization header with the token
    req.Header.Set("Authorization", "Bearer "+token.AccessToken)

    // Make the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Failed to get user info: %s", err.Error())
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }
    defer resp.Body.Close()

    // Handle the response
    var emails interface{} // Change to interface{} to inspect the structure of the response
    if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
        log.Printf("Failed to decode user info: %s", err.Error())
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }

    //fmt.Println("Emails response:", emails)

    // Handling the case where emails is an array
    var userEmail string
    switch v := emails.(type) {
    case []interface{}:
        // If the response is an array of emails, find the primary one
        for _, emailObj := range v {
            emailData, ok := emailObj.(map[string]interface{})
            if !ok {
                continue
            }
            if primary, ok := emailData["primary"].(bool); ok && primary {
                userEmail, _ = emailData["email"].(string)
                break
            }
        }
    case map[string]interface{}:
        // If it's a single email object, extract the email
        if primary, ok := v["primary"].(bool); ok && primary {
            userEmail, _ = v["email"].(string)
        }
    default:
        log.Printf("Unexpected response format: %v", v)
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }

    if userEmail == "" {
        log.Printf("Could not find primary email")
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
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

