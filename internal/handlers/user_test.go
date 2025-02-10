package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"forum/internal/db"

	_ "github.com/mattn/go-sqlite3" // Allowed package
	"golang.org/x/crypto/bcrypt"
)

func TestLoginHandler_GET(t *testing.T) {
	setupTestDB(t)
	defer db.DB.Close()

	req := httptest.NewRequest("GET", "/login", nil)
	rr := httptest.NewRecorder()
	LoginHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestLoginHandler_POST_InvalidCredentials(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Insert test user
	hashedPass, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	testDB.Exec("INSERT INTO users (user_id, username, password) VALUES (?, ?, ?)", "1", "testuser", hashedPass)

	form := url.Values{}
	form.Add("identifier", "wronguser")
	form.Add("password", "wrongpass")
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	LoginHandler(rr, req)

	if rr.Code != http.StatusOK { // Handler re-renders login page with errors
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestLoginHandler_POST_DBError(t *testing.T) {
	testDB := setupTestDB(t)
	testDB.Close() // Force DB error

	form := url.Values{"identifier": {"test"}, "password": {"test"}}
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	LoginHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestRegisterHandler_GET(t *testing.T) {
	setupTestDB(t)
	defer db.DB.Close()

	req := httptest.NewRequest("GET", "/register", nil)
	rr := httptest.NewRecorder()
	RegisterHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestRegisterHandler_POST_ExistingUser(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	testDB.Exec("INSERT INTO users (username, email) VALUES (?, ?)", "existing", "existing@test.com")

	form := url.Values{
		"username":        {"existing"},
		"email":           {"existing@test.com"},
		"password":        {"pass"},
		"confirmpassword": {"pass"},
	}
	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	RegisterHandler(rr, req)

	if rr.Code != http.StatusOK { // Re-renders form with error
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestRegisterHandler_POST_DBError(t *testing.T) {
	testDB := setupTestDB(t)
	testDB.Close() // Force DB error

	form := url.Values{
		"username":        {"newuser"},
		"email":           {"new@test.com"},
		"password":        {"pass"},
		"confirmpassword": {"pass"},
	}
	req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	RegisterHandler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestLogoutHandler_InvalidMethod(t *testing.T) {
	setupTestDB(t)
	defer db.DB.Close()

	req := httptest.NewRequest("GET", "/logout", nil)
	rr := httptest.NewRecorder()
	LogoutHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", rr.Code)
	}
}

func TestLogoutHandler_POST(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Insert test session
	testDB.Exec("INSERT INTO sessions (session_id, user_id) VALUES (?, ?)", "testsession", "1")

	req := httptest.NewRequest("POST", "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "testsession"})
	rr := httptest.NewRecorder()

	LogoutHandler(rr, req)

	// Verify session deleted
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM sessions WHERE session_id = 'testsession'").Scan(&count)
	if count != 0 {
		t.Error("Session not deleted from database")
	}

	// Verify cookie cleared
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value != "" {
		t.Error("Session cookie not cleared")
	}
}
