package handlers

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetFreePrompt(db *pgxpool.Pool, deviceID string) (bool, error) {

	err := db.Exec(context.Background(), `
	UPDATE users SET daily_used_promts=4 WHERE device_id=$1` deviceID)


}