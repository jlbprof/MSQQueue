package main

import (
	"database/sql"
	"msgqueue/models"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type Queue struct {
	db *sql.DB
}

func NewQueue() (*Queue, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/app/data/messages.db"
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Initialize database schema from init.sql
	initSQL, err := os.ReadFile("init.sql")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(string(initSQL))
	if err != nil {
		return nil, err
	}

	return &Queue{db: db}, nil
}

func (q *Queue) Add(content string) (models.Message, error) {
	result, err := q.db.Exec("INSERT INTO messages (content) VALUES (?)", content)
	if err != nil {
		return models.Message{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.Message{}, err
	}

	var msg models.Message
	err = q.db.QueryRow("SELECT id, content, timestamp FROM messages WHERE id = ?", id).Scan(&msg.ID, &msg.Content, &msg.Timestamp)
	if err != nil {
		return models.Message{}, err
	}

	return msg, nil
}

func (q *Queue) GetAll(afterID int, limit int) ([]models.Message, error) {
	query := "SELECT id, content, timestamp FROM messages WHERE id > ? ORDER BY id"
	if limit > 0 {
		query += " LIMIT ?"
	}

	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = q.db.Query(query, afterID, limit)
	} else {
		rows, err = q.db.Query(query, afterID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

    messages := make([]models.Message, 0)
    for rows.Next() {
        var msg models.Message
        err := rows.Scan(&msg.ID, &msg.Content, &msg.Timestamp)
        if err != nil {
            return nil, err
        }
        messages = append(messages, msg)
    }
    return messages, rows.Err()
}

func (q *Queue) DeleteOlderThan(days int) (int, error) {
	result, err := q.db.Exec("DELETE FROM messages WHERE timestamp < datetime('now', '-? days')", days)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	return int(rowsAffected), err
}