package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect() (*pgxpool.Pool, error) {
	connStr := "postgres://core:12345678@localhost:5432/eatnow"

	// dbUser := os.Getenv("POSTGRES_USER")
	// dbPass := os.Getenv("POSTGRES_PASSWORD")
	// dbHost := os.Getenv("POSTGRES_HOST") // будет "db" — имя сервиса в docker-compose
	// dbPort := os.Getenv("POSTGRES_PORT") // обычно 5432
	// dbName := os.Getenv("POSTGRES_DB")

	// connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
	// 	dbUser, dbPass, dbHost, dbPort, dbName)

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to connect to database: %v\n", err)
		return nil, err
	}

	fmt.Println("✅ Connected to Postgres (pool)")
	return pool, nil
}
