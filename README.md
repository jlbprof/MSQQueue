# Message Queue API

A lightweight REST API for a message queue system built in Go, featuring SQLite persistence, role-based API key authentication, and a simple web UI.

## Features

- **RESTful API**: Endpoints for adding, querying, and deleting messages.
- **SQLite Persistence**: Messages stored in a local SQLite database with automatic schema initialization.
- **Authentication**: API key-based auth with roles (`user` for add/query, `admin` for delete).
- **Web UI**: Simple interface for user interaction (login, add messages, query, refresh).
- **JSON Validation**: Ensures message content is valid JSON.
- **Timestamp Indexing**: Efficient queries for age-based deletions.

## Architecture

### Components
- **`main.go`**: Server entry point, HTTP routing.
- **`queue.go`**: In-memory queue interface backed by SQLite.
- **`handlers/`**: HTTP handlers for auth and messages.
- **`models/`**: Data structures (Message).
- **`init.sql`**: Database schema and initial data.
- **`ui.html`**: Web UI for interactive use.

### Design Decisions
- **SQLite**: Chosen for simplicity and zero-config persistence.
- **API Keys**: Stateless auth suitable for scripts and users; keys hashed for security.
- **Roles**: Granular access control (users can't delete).
- **Web UI**: Self-contained HTML/JS for easy access without external tools.

## Database Schema

### Tables
- **`messages`**: Stores messages with ID, JSON content, and timestamp.
- **`users`**: User accounts for authentication.
- **`api_keys`**: Hashed API keys linked to users with roles.

Schema is initialized from `init.sql` on startup.

## Security

- **API Keys**: Generated on login, deleted on logout/re-login. Hashed with SHA256.
- **Roles**: `user` (add/query), `admin` (all + delete).
- **Validation**: All requests require valid key; delete requires `admin`.
- **No HTTPS**: For internal use; add TLS for production.

## Installation

### Prerequisites
- Go 1.24+ (upgraded during setup).
- SQLite (included with Go's driver).

### Setup
1. Clone/download the project.
2. Run `go mod tidy` to install dependencies.
3. Build: `go build -o msgqueue .`

### Initial Data
- Test user: `testuser` / `password` (hashes in `init.sql`).

## Usage

### Running the Server
```bash
./msgqueue
```
Server starts on `http://localhost:8080`.

### API Endpoints

#### Authentication
- `POST /login`  
  Body: `{"username": "testuser", "password": "password"}`  
  Response: `{"apiKey": "..."}`

- `POST /logout`  
  Headers: `Authorization: Bearer <apiKey>`  
  Deletes the key.

#### Messages (Require API Key)
- `POST /messages`  
  Headers: `Authorization: Bearer <apiKey>`, `Content-Type: application/json`  
  Body: `{"content": "{\"key\": \"value\"}"}`  (content must be valid JSON)  
  Response: Created message with ID/timestamp.

- `GET /messages`  
  Headers: `Authorization: Bearer <apiKey>`  
  Query: `?after_id=1&limit=10`  
  Response: Array of messages.

- `DELETE /messages` (Admin only)  
  Headers: `Authorization: Bearer <apiKey>`  
  Query: `?days=7`  
  Response: `{"deleted": count}`

#### UI
- `GET /ui`: Web interface for login, message management, and querying.

### Example Usage

#### For Scripts
```bash
# Login
KEY=$(curl -s -X POST http://localhost:8080/login -H "Content-Type: application/json" -d '{"username":"testuser","password":"password"}' | jq -r .apiKey)

# Add message
curl -X POST http://localhost:8080/messages -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" -d '{"content":"{\"event\":\"test\"}"}'

# Query
curl http://localhost:8080/messages -H "Authorization: Bearer $KEY"
```

#### For Users
1. Visit `http://localhost:8080/ui`.
2. Login with `testuser`/`password`.
3. Add messages, query, and refresh.

## Development

- **Modify Schema**: Edit `init.sql` and restart.
- **Add Users**: Insert into `users` table with hashed passwords.
- **Roles**: For admin keys, manually insert into `api_keys` (not via UI).
- **Testing**: Use curl or the UI; database resets on rebuild.

## Notes
- Database (`messages.db`) is created locally and not committed.
- For production, add HTTPS, stronger hashing (bcrypt), and user management.
- Built for internal use; scale as needed.