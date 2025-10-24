package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect() (*pgxpool.Pool, error) {
	connStr := "postgres://core:12345678@localhost:5432/eatnow"
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to connect to database: %v\n", err)
		return nil, err
	}

	fmt.Println("✅ Connected to Postgres (pool)")
	return pool, nil
}
