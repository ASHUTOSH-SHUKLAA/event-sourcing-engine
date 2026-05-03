//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Cannot connect:", err)
	}

	statements := []string{
		"ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS paused_at TIMESTAMP WITH TIME ZONE DEFAULT NULL",
		"ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS pause_days_remaining INTEGER DEFAULT NULL",
	}

	for _, stmt := range statements {
		fmt.Printf("Running: %s\n", strings.TrimSpace(stmt))
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("Failed: %v", err)
		}
		fmt.Println("  OK")
	}

	fmt.Println("✅ Migration 004 completed!")
}
