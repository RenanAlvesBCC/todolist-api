package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/repository"
	"github.com/RenanAlvesBCC/todolist-api/internal/services"
	"github.com/RenanAlvesBCC/todolist-api/internal/utils"
)

type AuthHandler struct {
	authService  *services.AuthService
	securityRepo *repository.SecurityRepository
}

func NewAuthHandler(authService *services.AuthService, securityRepo *repository.SecurityRepository) *AuthHandler {
	return &AuthHandler{authService: authService, securityRepo: securityRepo}
}

type credentials struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input credentials
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos: " + err.Error()})
		return
	}

	if err := h.authService.Register(input.Username, input.Password); err != nil {
		h.securityRepo.LogAction(nil, "register_failed", c.ClientIP(), c.GetHeader("User-Agent"), input.Username, false)
		c.JSON(http.StatusConflict, gin.H{"error": "usuário já existe"})
		return
	}

	h.securityRepo.LogAction(nil, "register_success", c.ClientIP(), c.GetHeader("User-Agent"), input.Username, true)
	c.JSON(http.StatusCreated, gin.H{"message": "usuário criado com sucesso"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input credentials
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos: " + err.Error()})
		return
	}

	token, err := h.authService.Login(input.Username, input.Password)
	if err != nil {
		// Loga tentativa falha sem revelar se o usuário existe ou não
		h.securityRepo.LogAction(nil, "login_failed", c.ClientIP(), c.GetHeader("User-Agent"), input.Username, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário ou senha incorretos"})
		return
	}

	// Extrai o userID do token pra logar com contexto
	if claims, err := utils.ValidateToken(token); err == nil {
		if uid, ok := claims["user_id"].(float64); ok {
			userID := uint(uid)
			h.securityRepo.LogAction(&userID, "login_success", c.ClientIP(), c.GetHeader("User-Agent"), "", true)
		}
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	header := c.GetHeader("Authorization")
	parts := strings.Split(header, " ")
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token não fornecido"})
		return
	}

	token := parts[1]
	expiresAt, err := utils.TokenExpiration(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token inválido"})
		return
	}

	if err := h.securityRepo.BlacklistToken(token, expiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao revogar token"})
		return
	}

	userID := uint(c.MustGet("user_id").(float64))
	h.securityRepo.LogAction(&userID, "logout", c.ClientIP(), c.GetHeader("User-Agent"), "", true)

	c.JSON(http.StatusOK, gin.H{"message": "logout realizado com sucesso"})
}
