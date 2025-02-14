package handlers

import (
	"context"
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
// Define a struct to handle the GitHub user data
type GitHubUser struct {
    Login string `json:"login"`
    Email string `json:"email"`
}

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

    req, err := http.NewRequest("GET", "https://api.github.com/user", nil) // Use `/user` endpoint for user details
    if err != nil {
        log.Printf("Failed to create request: %s", err.Error())
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }

    req.Header.Set("Authorization", "Bearer "+token.AccessToken)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Failed to get user info: %s", err.Error())
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }
    defer resp.Body.Close()

    // Parse the response into GitHubUser struct
    var user GitHubUser
    if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
        log.Printf("Failed to decode user info: %s", err.Error())
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }

    // Now user.Login will contain the GitHub username, and user.Email contains the user's email
    if user.Email == "" {
        
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
        return
    }

    // Use GitHub login as username, or create a fallback if not available
    userGitHubUsername := "GitHubUser" // Fallback username
    if user.Login != "" {
        userGitHubUsername = user.Login
    }

    // Find or create the user with email and username
    userID, err := findOrCreateUser(user.Email, userGitHubUsername)
    if err != nil {
        log.Printf("Failed to find or create user: %s", err.Error())
        utils.DisplayError(w, http.StatusInternalServerError, "Failed to authenticate")
        return
    }

    // Handle session management and redirection
    _, err = db.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
    if err != nil {
        utils.DisplayError(w, http.StatusInternalServerError, "Server error")
        log.Printf("Session delete error: %v", err)
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

func findOrCreateUser(email, username string) (int, error) {
	var userID int
	var existingEmail string
	var existingUsername string

	// First check if the user already exists by email
	err := db.DB.QueryRow("SELECT id, email, username FROM users WHERE email = ?", email).Scan(&userID, &existingEmail, &existingUsername)
	if err == nil {
		// User already exists, return the existing user ID
		return userID, nil
	}

	// If no user exists by email, check if username is unique
	err = db.DB.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
	if err == nil {
		// User with the same username already exists
		return userID, nil
	}

	
	// If the username already exists, create a unique username
	if err := createUniqueUsername(&username); err != nil {
		return 0, err
	}

	// If the user does not exist, create a new user
	_, err = db.DB.Exec("INSERT INTO users (email, username) VALUES (?, ?)", email, username)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	// Fetch the newly created user's ID
	err = db.DB.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch user ID: %w", err)
	}

	return userID, nil
}

func createUniqueUsername(username *string) error {
	var existingUsername string
	for {
		// Check if the username is already taken
		err := db.DB.QueryRow("SELECT username FROM users WHERE username = ?", *username).Scan(&existingUsername)
		if err != nil {
			// No conflict, the username is unique
			return nil
		}
		// Username already exists, generate a new one (e.g., append a random string or increment a counter)
		*username = fmt.Sprintf("%s_%d", *username, uuid.New().ID())
	}
}
