package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
	"github.com/RenanAlvesBCC/todolist-api/internal/utils"
)

type AssignmentProvider interface {
	Assign(requesterID, taskListID, targetUserID uint) error
	Unassign(requesterID, taskListID, targetUserID uint) error
	List(taskListID uint) ([]models.ListAssignment, error)
}

type AssignmentHandler struct {
	svc AssignmentProvider
}

func NewAssignmentHandler(svc AssignmentProvider) *AssignmentHandler {
	return &AssignmentHandler{svc: svc}
}

type assignInput struct {
	UserID uint `json:"user_id" binding:"required"`
}

func (h *AssignmentHandler) Assign(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	var input assignInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	if err := h.svc.Assign(getUserID(c), uint(listID), input.UserID); err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "mecânico atribuído com sucesso"})
}

func (h *AssignmentHandler) Unassign(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	userID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "userId inválido")
		return
	}

	if err := h.svc.Unassign(getUserID(c), uint(listID), uint(userID)); err != nil {
		utils.RespondError(c, http.StatusForbidden, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AssignmentHandler) List(c *gin.Context) {
	listID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	assignments, err := h.svc.List(uint(listID))
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "erro ao buscar atribuições")
		return
	}
	c.JSON(http.StatusOK, assignments)
}
