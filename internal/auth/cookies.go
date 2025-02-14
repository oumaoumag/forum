package auth

import (
    "database/sql"
    "fmt"
    "log"
    "net/http"
    "time"

    "forum/internal/db"

    "github.com/google/uuid"
)

// Session represents a user session
type Session struct {
    ID        string
    UserID    int
    ExpiresAt time.Time
}

// CreateSession creates a new session for the given user ID
func CreateSession(w http.ResponseWriter, userID int) error {
    // Delete any existing sessions for this user
    if err := deleteExistingSessions(userID); err != nil {
        return fmt.Errorf("failed to delete existing sessions: %w", err)
    }

    // Create new session
    session := &Session{
        ID:        uuid.New().String(),
        UserID:    userID,
        ExpiresAt: time.Now().Add(24 * time.Hour),
    }

    // Save session to database
    if err := saveSession(session); err != nil {
        return fmt.Errorf("failed to save session: %w", err)
    }

    // Set session cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "session_id",
        Value:    session.ID,
        Expires:  session.ExpiresAt,
        Path:     "/",
        HttpOnly: true,
        Secure:   true, // Enable for HTTPS
        SameSite: http.SameSiteStrictMode,
    })

    return nil
}

// GetSession retrieves a session from the database
func GetSession(sessionID string) (*Session, error) {
    session := &Session{}
    err := db.DB.QueryRow(
        "SELECT session_id, user_id, expires_at FROM sessions WHERE session_id = ?",
        sessionID,
    ).Scan(&session.ID, &session.UserID, &session.ExpiresAt)

    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get session: %w", err)
    }

    return session, nil
}

// DeleteSession removes a session from the database
func DeleteSession(w http.ResponseWriter, sessionID string) error {
    _, err := db.DB.Exec("DELETE FROM sessions WHERE session_id = ?", sessionID)
    if err != nil {
        return fmt.Errorf("failed to delete session: %w", err)
    }

    // Clear the session cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "session_id",
        Value:    "",
        Expires:  time.Now().Add(-24 * time.Hour),
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    })

    return nil
}

// CleanExpiredSessions removes all expired sessions from the database
func CleanExpiredSessions() error {
    result, err := db.DB.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
    if err != nil {
        return fmt.Errorf("failed to clean expired sessions: %w", err)
    }

    count, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get affected rows: %w", err)
    }

    if count > 0 {
        log.Printf("Cleaned %d expired sessions", count)
    }

    return nil
}

// Internal helper functions

func deleteExistingSessions(userID int) error {
    _, err := db.DB.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
    return err
}

func saveSession(session *Session) error {
    _, err := db.DB.Exec(
        "INSERT INTO sessions (session_id, user_id, expires_at) VALUES (?, ?, ?)",
        session.ID,
        session.UserID,
        session.ExpiresAt,
    )
    return err
}
