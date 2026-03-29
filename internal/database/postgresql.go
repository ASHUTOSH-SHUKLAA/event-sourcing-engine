package database

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect(connStr string) {
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatal("DB pool creation error:", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("DB ping error:", err)
	}

	Pool = pool
	log.Println("✅ Database connected")
}
