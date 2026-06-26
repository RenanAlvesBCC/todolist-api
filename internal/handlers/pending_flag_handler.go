package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
	"github.com/RenanAlvesBCC/todolist-api/internal/utils"
)

type PendingFlagProvider interface {
	AddFlag(listID, userID uint, flagType models.FlagType, note string) (*models.PendingFlag, error)
	ResolveFlag(listID, flagID, userID uint) error
	ListFlags(listID, userID uint) ([]models.PendingFlag, error)
}

type PendingFlagHandler struct {
	svc PendingFlagProvider
}

func NewPendingFlagHandler(svc PendingFlagProvider) *PendingFlagHandler {
	return &PendingFlagHandler{svc: svc}
}

type addFlagInput struct {
	FlagType models.FlagType `json:"flag_type" binding:"required"`
	Note     string          `json:"note"`
}

func (h *PendingFlagHandler) Add(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var input addFlagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	f, err := h.svc.AddFlag(uint(listID), getUserID(c), input.FlagType, input.Note)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, f)
}

func (h *PendingFlagHandler) Resolve(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}
	flagID, err := strconv.ParseUint(c.Param("flagId"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id de pendência inválido")
		return
	}

	if err := h.svc.ResolveFlag(uint(listID), uint(flagID), getUserID(c)); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PendingFlagHandler) List(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	flags, err := h.svc.ListFlags(uint(listID), getUserID(c))
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, flags)
}
