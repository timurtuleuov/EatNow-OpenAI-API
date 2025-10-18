package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func Connect() (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), "postgres://core:12345678@localhost:5432/govault")
	if err != nil {
		fmt.Fprintf(os.Stderr, "undable to connect to database", err)
		return nil, err
	}

	fmt.Println("Connected to Postgres")
	return conn, nil

}
