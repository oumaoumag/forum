package main

import (
	"log"
	"net/http"
	"time"

	// "golang.org/x/crypto/acme/autocert"

	"forum/internal/auth"
	"forum/internal/db"
	"forum/internal/handlers"
	"forum/internal/server"
)

func main() {
	// Initialize the database
	if err := db.Init("./forum.db"); err != nil {
		log.Fatal("Database initialization failed: ", err)
	}

	go db.ScheduleSessionCleanup(1*time.Hour, db.CleanupExpiredSessions)

	// Main router creation
	mux := http.NewServeMux()

	// Static file handling
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Set up routes
	mux.HandleFunc("/", handlers.HomeHandler)
	mux.Handle("/login", auth.SessionMiddleware(auth.RedirectIfAuthenticated(http.HandlerFunc(handlers.LoginHandler))))
	mux.Handle("/register", auth.SessionMiddleware(auth.RedirectIfAuthenticated(http.HandlerFunc(handlers.RegisterHandler))))
	mux.Handle("/post/create", auth.SessionMiddleware(auth.RequireAuth(http.HandlerFunc(handlers.CreatePostHandler))))
	mux.Handle("/comment/create", auth.SessionMiddleware(auth.RequireAuth(http.HandlerFunc(handlers.CreateCommentHandler))))
	mux.Handle("/like", auth.SessionMiddleware(auth.RequireAuth(http.HandlerFunc(handlers.LikeHandler))))
	mux.Handle("/logout", auth.SessionMiddleware(auth.RequireAuth(http.HandlerFunc(handlers.LogoutHandler))))

	// Register GitHub OAuth routes with the same mux
	mux.HandleFunc("/auth/github", handlers.GitHubLoginHandler)
	mux.HandleFunc("/oauth2/callback/github", handlers.GitHubCallbackHandler)

	// Register GitHub OAuth routes with the same mux
	mux.HandleFunc("/auth/google", handlers.GoogleLoginHandler)
    mux.HandleFunc("/auth/callback/google", handlers.GoogleCallbackHandler)

	// server configuration
	config := server.NewDefaultConfig(mux)

	// Initialize & start server
	srv := server.New(config)
	if err := srv.Start(); err != nil {
		log.Fatal("Server error:", err)
	} 	
}
