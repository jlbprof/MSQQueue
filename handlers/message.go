package handlers

import (
	"database/sql"
	"encoding/json"
	"msgqueue/models"
	"net/http"
	"strconv"
	"strings"
)

type MessageQueue interface {
	Add(content string) (models.Message, error)
	GetAll(afterID int, limit int) ([]models.Message, error)
	DeleteOlderThan(days int) (int, error)
}

func MessagesHandler(queue MessageQueue, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		key := strings.TrimPrefix(auth, "Bearer ")
		role, err := ValidateAPIKey(db, key)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodPost:
			if role != "user" && role != "admin" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			handleAddMessage(w, r, queue)
		case http.MethodGet:
			if role != "user" && role != "admin" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			handleGetMessages(w, r, queue)
		case http.MethodDelete:
			if role != "admin" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			handleDeleteMessages(w, r, queue)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleAddMessage(w http.ResponseWriter, r *http.Request, queue MessageQueue) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}
	// Validate that content is valid JSON
	var js json.RawMessage
	if err := json.Unmarshal([]byte(req.Content), &js); err != nil {
		http.Error(w, "Content must be valid JSON", http.StatusBadRequest)
		return
	}
	msg, err := queue.Add(req.Content)
	if err != nil {
		http.Error(w, "Failed to add message", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}

func handleGetMessages(w http.ResponseWriter, r *http.Request, queue MessageQueue) {
	afterIDStr := r.URL.Query().Get("after_id")
	limitStr := r.URL.Query().Get("limit")

	afterID := 0
	if afterIDStr != "" {
		var err error
		afterID, err = strconv.Atoi(afterIDStr)
		if err != nil {
			http.Error(w, "Invalid after_id", http.StatusBadRequest)
			return
		}
	}

	limit := 0
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "Invalid limit", http.StatusBadRequest)
			return
		}
	}

	messages, err := queue.GetAll(afterID, limit)
	if err != nil {
		http.Error(w, "Failed to retrieve messages", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func handleDeleteMessages(w http.ResponseWriter, r *http.Request, queue MessageQueue) {
	daysStr := r.URL.Query().Get("days")
	if daysStr == "" {
		http.Error(w, "days parameter required", http.StatusBadRequest)
		return
	}
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		http.Error(w, "Invalid days", http.StatusBadRequest)
		return
	}
	deleted, err := queue.DeleteOlderThan(days)
	if err != nil {
		http.Error(w, "Failed to delete messages", http.StatusInternalServerError)
		return
	}
	response := map[string]int{"deleted": deleted}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}