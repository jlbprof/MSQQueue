package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// LoginRequest represents the login payload
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	APIKey string `json:"apiKey"`
}

// GenerateAPIKey generates a random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashKey hashes the API key for storage
func HashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// ValidateAPIKey checks if the key is valid and returns the role
func ValidateAPIKey(db *sql.DB, key string) (string, error) {
	hash := HashKey(key)
	var role string
	err := db.QueryRow("SELECT role FROM api_keys WHERE key_hash = ?", hash).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("invalid API key")
		}
		return "", err
	}
	return role, nil
}

// AuthenticateUser checks username/password and returns user ID
func AuthenticateUser(db *sql.DB, username, password string) (int, error) {
	var id int
	var hash string
	err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", username).Scan(&id, &hash)
	if err != nil {
		return 0, errors.New("invalid credentials")
	}
	if hash != HashKey(password) {
		return 0, errors.New("invalid credentials")
	}
	return id, nil
}

// IssueAPIKey generates and stores a new API key for the user, deleting old ones
func IssueAPIKey(db *sql.DB, userID int, role string) (string, error) {
	// Delete old keys for the user
	_, err := db.Exec("DELETE FROM api_keys WHERE user_id = ?", userID)
	if err != nil {
		return "", err
	}

	// Generate new key
	key, err := GenerateAPIKey()
	if err != nil {
		return "", err
	}

	// Hash and store
	hash := HashKey(key)

	_, err = db.Exec("INSERT INTO api_keys (key_hash, user_id, role) VALUES (?, ?, ?)", hash, userID, role)
	if err != nil {
		return "", err
	}

	return key, nil
}

// DeleteAPIKey removes the key
func DeleteAPIKey(db *sql.DB, key string) error {
	hash := HashKey(key)
	_, err := db.Exec("DELETE FROM api_keys WHERE key_hash = ?", hash)
	return err
}

// RequireAPIKey middleware
func RequireAPIKey(db *sql.DB, requiredRole string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			key := strings.TrimPrefix(auth, "Bearer ")
			role, err := ValidateAPIKey(db, key)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if requiredRole != "" && role != requiredRole && role != "admin" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}
}

// LoginHandler handles user login and API key issuance
func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		userID, err := AuthenticateUser(db, req.Username, req.Password)
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		key, err := IssueAPIKey(db, userID, "user")
		if err != nil {
			http.Error(w, "Failed to issue API key", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{APIKey: key})
	}
}

// LogoutHandler handles API key deletion
func LogoutHandler(db *sql.DB) http.HandlerFunc {
	return RequireAPIKey(db, "")(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		auth := r.Header.Get("Authorization")
		key := strings.TrimPrefix(auth, "Bearer ")
		if err := DeleteAPIKey(db, key); err != nil {
			http.Error(w, "Failed to logout", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}