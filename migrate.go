//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"gin-quickstart/internal/config"
)

func main() {
	dbURL := config.GetDBUrl()
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Cannot connect to database:", err)
	}

	files := []string{
		"migrations/001_create_users_table.sql",
		"migrations/002_create_app_event_tables.sql",
		"migrations/003_upgrade_subscription_logic.sql",
		"migrations/004_add_pause_resume.sql",
	}

	for _, file := range files {
		fmt.Printf("Running migration: %s...\n", file)
		sqlFile, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}

		if _, err := db.Exec(string(sqlFile)); err != nil {
			log.Fatalf("Failed to execute migration %s: %v", file, err)
		}
	}

	fmt.Println("🎉 Migration completed successfully on Cloud Database!")
}
