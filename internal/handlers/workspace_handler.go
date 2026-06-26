package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
	"github.com/RenanAlvesBCC/todolist-api/internal/utils"
)

type WorkspaceProvider interface {
	CreateWorkspace(ownerID uint, name, description string) (*models.Workspace, error)
	GetMyWorkspace(userID uint) (*models.Workspace, error)
	UpdateWorkspace(ownerID uint, name, description string) (*models.Workspace, error)
	GenerateInvite(requesterID uint, role models.WorkspaceRole) (string, error)
	AcceptInvite(userID uint, code string) (*models.Workspace, error)
	ListMembers(requesterID uint) ([]models.WorkspaceMember, error)
	ListInvites(ownerID uint) ([]models.WorkspaceInvite, error)
	RemoveMember(ownerID, targetUserID uint) error
	GetInvitePreview(code string) (*models.WorkspaceInvite, *models.Workspace, error)
}

type WorkspaceHandler struct {
	svc WorkspaceProvider
}

func NewWorkspaceHandler(svc WorkspaceProvider) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc}
}

type createWorkspaceInput struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type generateInviteInput struct {
	Role models.WorkspaceRole `json:"role" binding:"required"`
}

func (h *WorkspaceHandler) Create(c *gin.Context) {
	var input createWorkspaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	ws, err := h.svc.CreateWorkspace(getUserID(c), input.Name, input.Description)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, ws)
}

func (h *WorkspaceHandler) Get(c *gin.Context) {
	ws, err := h.svc.GetMyWorkspace(getUserID(c))
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, ws)
}

func (h *WorkspaceHandler) Update(c *gin.Context) {
	var input createWorkspaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	ws, err := h.svc.UpdateWorkspace(getUserID(c), input.Name, input.Description)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, ws)
}

func (h *WorkspaceHandler) GenerateInvite(c *gin.Context) {
	var input generateInviteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}

	code, err := h.svc.GenerateInvite(getUserID(c), input.Role)
	if err != nil {
		utils.RespondError(c, http.StatusForbidden, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": code})
}

func (h *WorkspaceHandler) InvitePreview(c *gin.Context) {
	code := c.Param("code")
	invite, ws, err := h.svc.GetInvitePreview(code)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"workspace_name": ws.Name,
		"role":           invite.Role,
		"expires_at":     invite.ExpiresAt,
	})
}

func (h *WorkspaceHandler) AcceptInvite(c *gin.Context) {
	code := c.Param("code")
	ws, err := h.svc.AcceptInvite(getUserID(c), code)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, ws)
}

func (h *WorkspaceHandler) ListMembers(c *gin.Context) {
	members, err := h.svc.ListMembers(getUserID(c))
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, members)
}

func (h *WorkspaceHandler) ListInvites(c *gin.Context) {
	invites, err := h.svc.ListInvites(getUserID(c))
	if err != nil {
		utils.RespondError(c, http.StatusForbidden, err.Error())
		return
	}
	c.JSON(http.StatusOK, invites)
}

func (h *WorkspaceHandler) RemoveMember(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return
	}

	if err := h.svc.RemoveMember(getUserID(c), uint(targetID)); err != nil {
		utils.RespondError(c, http.StatusForbidden, err.Error())
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
