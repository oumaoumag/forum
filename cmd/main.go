package main

import (
	"log"
	"net/http"
	"time"

	"forum/internal/auth"
	"forum/internal/db"
	"forum/internal/handlers"
)

func main() {
	// Initialize the database
	if err := db.Init("./forum.db"); err != nil {
		log.Println("error:", err)
	}

	go db.ScheduleSessionCleanup(1*time.Hour, db.CleanupExpiredSessions)

	mux := http.NewServeMux()

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

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server started at http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
