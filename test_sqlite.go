package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite3", "./data/test.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Enable WAL mode
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		log.Fatalf("Failed to set WAL mode: %v", err)
	}

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_id INTEGER UNIQUE NOT NULL,
			balance INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	_, err = db.Exec("INSERT OR IGNORE INTO users (telegram_id, balance) VALUES (12345, 100000)")
	if err != nil {
		log.Fatalf("Failed to insert data: %v", err)
	}

	// Query data
	var balance int
	err = db.QueryRow("SELECT balance FROM users WHERE telegram_id = ?", 12345).Scan(&balance)
	if err != nil {
		log.Fatalf("Failed to query data: %v", err)
	}

	fmt.Printf("Success! Balance: %d\n", balance)
}
