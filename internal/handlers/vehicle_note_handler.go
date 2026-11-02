package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
	"github.com/RenanAlvesBCC/oficina-api/internal/utils"
)

type VehicleNoteProvider interface {
	AddNote(listID, userID uint, text string) (*models.VehicleNote, error)
	ListNotes(listID, userID uint) ([]models.VehicleNote, error)
	DeleteNote(listID, noteID, userID uint) error
}

type VehicleNoteHandler struct {
	svc VehicleNoteProvider
}

func NewVehicleNoteHandler(svc VehicleNoteProvider) *VehicleNoteHandler {
	return &VehicleNoteHandler{svc: svc}
}

type addNoteInput struct {
	Text string `json:"text"`
}

func respondNoteError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNoteTextRequired), errors.Is(err, services.ErrNoteTextTooLong):
		utils.RespondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, services.ErrNotFound), errors.Is(err, services.ErrNoteNotFound):
		utils.RespondError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrNoteForbidden):
		utils.RespondError(c, http.StatusForbidden, err.Error())
	default:
		utils.RespondError(c, http.StatusInternalServerError, "erro ao processar informações importantes")
	}
}

func (h *VehicleNoteHandler) Add(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var input addNoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	n, err := h.svc.AddNote(uint(listID), getUserID(c), input.Text)
	if err != nil {
		respondNoteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, n)
}

func (h *VehicleNoteHandler) List(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	notes, err := h.svc.ListNotes(uint(listID), getUserID(c))
	if err != nil {
		respondNoteError(c, err)
		return
	}
	c.JSON(http.StatusOK, notes)
}

func (h *VehicleNoteHandler) Delete(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}
	noteID, err := strconv.ParseUint(c.Param("noteId"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id de informação inválido")
		return
	}

	if err := h.svc.DeleteNote(uint(listID), uint(noteID), getUserID(c)); err != nil {
		respondNoteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
