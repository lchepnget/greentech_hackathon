package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

const migrationName = "20260820221438_users"

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	schema, err := os.ReadFile("migrations/20260820221438_users.postgres.up.sql")
	if err != nil {
		log.Fatalf("read schema: %v", err)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer conn.Close(ctx)
	if _, err = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS regenfeed_migrations (name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		log.Fatalf("create migration table: %v", err)
	}
	var applied bool
	if err = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM regenfeed_migrations WHERE name = $1)`, migrationName).Scan(&applied); err != nil {
		log.Fatalf("check migration: %v", err)
	}
	if applied {
		fmt.Println("database schema is already current")
		return
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("begin migration: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, string(schema)); err != nil {
		log.Fatalf("apply schema: %v", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO regenfeed_migrations (name) VALUES ($1)`, migrationName); err != nil {
		log.Fatalf("record migration: %v", err)
	}
	if err = tx.Commit(ctx); err != nil {
		log.Fatalf("commit migration: %v", err)
	}
	fmt.Println("database migration applied")
}
