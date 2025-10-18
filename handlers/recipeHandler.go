package handlers

import (
	"fmt"
	model "openai/models"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetRecipeByPrompt(c *gin.Context) (model.Recipe, error) {
	prompt := c.Param("prompt")
	mockData := model.MockRecipes()

	if strings.Contains(prompt, "spagetti") {
		return mockData[1], nil
	}

	if strings.Contains(prompt, "chicken") {
		return mockData[1], nil
	}
	if strings.Contains(prompt, "avocado") {
		return mockData[2], nil
	}
	return model.Recipe{}, fmt.Errorf("рецепт по запросу '%s' не найден", prompt)
}
