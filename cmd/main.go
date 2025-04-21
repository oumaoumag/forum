package main

import (
	"log"
	"net/http"
	"time"
	"crypto/tls"
	"golang.org/x/crypto/acme/autocert"

	"forum/internal/auth"
	"forum/internal/db"
	"forum/internal/handlers"
)

// handles HTTP to HTTPS redirection:
func redirectToHTTPS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") != "https" {
			sslUrl := "https://" + r.Host + r.RequestURI
			http.Redirect(w, r, sslUrl, http.StatusTemporaryRedirect)
			return
		}
		next.ServeHTTP(w, r)
	})
}

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

	// TLS Configuration
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		PreferServerCipherSuites: true,
	}

	// 	HTTPS Server
	httpsServer := &http.Server{
		Addr: ":443"
	}
	// Register GitHub OAuth routes with the same mux
	mux.HandleFunc("/auth/github", handlers.GitHubLoginHandler)
	mux.HandleFunc("/oauth2/callback/github", handlers.GitHubCallbackHandler)

	// Register GitHub OAuth routes with the same mux
	mux.HandleFunc("/auth/google", handlers.GoogleLoginHandler)
    mux.HandleFunc("/auth/callback/google", handlers.GoogleCallbackHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server started at http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
