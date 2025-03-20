package handlers

import (
	"net/http"

	"StartupPCConfigurator/internal/config/usecase"
	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	service usecase.ConfigService
}

// Конструктор
func NewConfigHandler(service usecase.ConfigService) *ConfigHandler {
	return &ConfigHandler{service: service}
}

// GetComponents обрабатывает GET /config/components
// query-параметры: category, search
func (h *ConfigHandler) GetComponents(c *gin.Context) {
	category := c.Query("category")
	search := c.Query("search")

	// Обращаемся к бизнес-логике, получаем список компонентов
	components, err := h.service.FetchComponents(category, search)
	if err != nil {
		// Можно вернуть 500 или более детально обработать ошибку
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Возвращаем JSON-массив
	c.JSON(http.StatusOK, components)
}
