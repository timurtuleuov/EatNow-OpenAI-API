package handlers

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func createUser(db *sql.DB, username string, email string, password string, platform string) (bool, error) {

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false, fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO users (name, password, email, platform)
        VALUES ($1, $2, $3, $4)
	`, username, string(hashedPassword), email, platform)
	if err != nil {
		return false, fmt.Errorf("failed to insert user: %w", err)
	}
	return true, nil

}
