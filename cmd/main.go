package main

import (
	"forum/internal/auth"
	"forum/internal/db"
	"forum/internal/handlers"
	"log"
	"net/http"
)

func main() {
	// Initialize the database
	db.Init()
	go db.ScheduleSessionCleanup()

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

	log.Println("Server started at http://192.168.89.189:8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
