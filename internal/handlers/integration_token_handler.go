package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
	"github.com/RenanAlvesBCC/oficina-api/internal/utils"
)

type IntegrationTokenProvider interface {
	ListTokens(userID uint) ([]models.IntegrationToken, error)
	CreateToken(userID uint, name string) (*services.CreatedIntegrationToken, error)
	RevokeToken(userID, id uint) error
}

type IntegrationTokenHandler struct {
	svc IntegrationTokenProvider
}

func NewIntegrationTokenHandler(svc IntegrationTokenProvider) *IntegrationTokenHandler {
	return &IntegrationTokenHandler{svc: svc}
}

func respondTokenError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrTokenOwnerOnly):
		utils.RespondError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, services.ErrNoWorkspace), errors.Is(err, services.ErrTokenNotFound):
		utils.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrTokenNameRequired), errors.Is(err, services.ErrTokenNameTooLong):
		utils.RespondError(c, http.StatusBadRequest, err.Error())
	default:
		utils.RespondError(c, http.StatusInternalServerError, "erro ao processar tokens de integração")
	}
}

func (h *IntegrationTokenHandler) List(c *gin.Context) {
	tokens, err := h.svc.ListTokens(getUserID(c))
	if err != nil {
		respondTokenError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

type createTokenInput struct {
	Name string `json:"name"`
}

func (h *IntegrationTokenHandler) Create(c *gin.Context) {
	var input createTokenInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}
	created, err := h.svc.CreateToken(getUserID(c), input.Name)
	if err != nil {
		respondTokenError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *IntegrationTokenHandler) Revoke(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.svc.RevokeToken(getUserID(c), id); err != nil {
		respondTokenError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
