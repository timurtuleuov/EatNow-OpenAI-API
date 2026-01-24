package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	// Устанавливаем Gin в тестовый режим
	gin.SetMode(gin.TestMode)

	t.Run("Missing Authorization Header", func(t *testing.T) {
		resp := httptest.NewRecorder()
		c, r := gin.CreateTestContext(resp)

		// Настраиваем маршрут с middleware
		r.Use(AuthMiddleware())
		r.GET("/home", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		c.Request, _ = http.NewRequest(http.MethodGet, "/home", nil)
		r.ServeHTTP(resp, c.Request)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
	})

	t.Run("Invalid Token Format", func(t *testing.T) {
		resp := httptest.NewRecorder()
		c, r := gin.CreateTestContext(resp)

		r.Use(AuthMiddleware())
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "WrongFormat token123")

		r.ServeHTTP(resp, c.Request)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
	})
}
