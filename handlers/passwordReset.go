package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func generateResetToken() (string, string) {
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)
	hash := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(hash[:])
}

func ForgotPassword(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Email string `json:"email"`
		}

		if err := c.ShouldBindJSON(&body); err != nil || body.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
			return
		}

		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id::text FROM users WHERE email = $1", body.Email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"})
			return
		}

		_, err = db.Exec(context.Background(),
			`DELETE FROM password_reset_tokens WHERE user_id = $1::uuid`, userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		token, tokenHash := generateResetToken()

		_, err = db.Exec(context.Background(), `
			INSERT INTO password_reset_tokens (token_hash, user_id, expires_at)
			VALUES ($1, $2::uuid, $3)
		`, tokenHash, userID, time.Now().Add(1*time.Hour))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		sender := NewEmailSender()
		if sender.Username != "" {
			subject := "Восстановление пароля EatNow"
			resetLink := fmt.Sprintf("http://localhost:8080/reset-password?token=%s", token)
			emailBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0; padding:0; background:#f4f4f4; font-family:Arial, Helvetica, sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f4; padding:40px 0;">
    <tr>
      <td align="center">
        <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff; border-radius:12px; overflow:hidden; box-shadow:0 2px 12px rgba(0,0,0,0.08);">
          <tr>
            <td style="background:#e74c3c; padding:30px; text-align:center;">
              <span style="font-size:48px;">🍳</span>
              <h1 style="color:#fff; margin:8px 0 0; font-size:28px; font-weight:700;">EatNow</h1>
            </td>
          </tr>
          <tr>
            <td style="padding:40px 30px; text-align:center;">
              <h2 style="color:#333; margin:0 0 12px; font-size:22px;">Восстановление пароля</h2>
              <p style="color:#666; font-size:15px; line-height:1.5; margin:0 0 28px;">
                Вы получили это письмо, потому что запросили сброс пароля.<br>
                Нажмите кнопку ниже, чтобы задать новый пароль:
              </p>
              <a href="%s" style="display:inline-block; background:#e74c3c; color:#fff; padding:14px 36px; border-radius:8px; text-decoration:none; font-size:16px; font-weight:600;">Сбросить пароль</a>
              <p style="color:#999; font-size:13px; margin-top:30px; line-height:1.4;">
                Если кнопка не работает, скопируйте ссылку в браузер:<br>
                <span style="color:#666; word-break:break-all;">%s</span>
              </p>
            </td>
          </tr>
          <tr>
            <td style="background:#f8f9fa; padding:20px 30px; text-align:center;">
              <p style="color:#aaa; font-size:12px; margin:0; line-height:1.4;">
                Ссылка действительна 1 час.<br>
                Если вы не запрашивали восстановление пароля, просто проигнорируйте это письмо.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, resetLink, resetLink)
			_ = sender.Send(body.Email, subject, emailBody)
		}

		c.JSON(http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent"})
	}
}

func ResetPassword(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Token       string `json:"token"`
			NewPassword string `json:"new_password"`
		}

		if err := c.ShouldBindJSON(&body); err != nil || body.Token == "" || body.NewPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "token and new_password are required"})
			return
		}

		if len(body.NewPassword) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 6 characters"})
			return
		}

		tokenHash := sha256.Sum256([]byte(body.Token))
		hashHex := hex.EncodeToString(tokenHash[:])

		var userID string
		err := db.QueryRow(context.Background(), `
			DELETE FROM password_reset_tokens
			WHERE token_hash = $1 AND expires_at > NOW()
			RETURNING user_id
		`, hashHex).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		_, err = db.Exec(context.Background(),
			`UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2::uuid`,
			string(hashedPassword), userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "password has been reset successfully"})
	}
}
