package main

import (
	"fmt"
	"log"
	"msgqueue/handlers"
	"net/http"
	"syscall"
)

func main() {
	// Nice the process to lower priority
	syscall.Setpriority(syscall.PRIO_PROCESS, 0, 10)

	queue, err := NewQueue()
	if err != nil {
		log.Fatal("Failed to initialize queue:", err)
	}
	defer queue.db.Close()

	// Auth routes
	http.HandleFunc("/login", handlers.LoginHandler(queue.db))
	http.HandleFunc("/logout", handlers.LogoutHandler(queue.db))

	// Messages routes with auth
	http.HandleFunc("/messages", handlers.MessagesHandler(queue, queue.db))

	// UI route
	http.HandleFunc("/ui", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "ui.html")
	})

	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
