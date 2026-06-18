package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/services"
)

// AuthHandler só traduz HTTP <-> service. Não sabe nada sobre hash, JWT ou banco.
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler cria o handler recebendo o service que ele vai usar.
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// credentials é o formato esperado no corpo JSON de /register e /login.
type credentials struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input credentials
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	if err := h.authService.Register(input.Username, input.Password); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "usuário já existe"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "usuário criado com sucesso"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input credentials
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	token, err := h.authService.Login(input.Username, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
