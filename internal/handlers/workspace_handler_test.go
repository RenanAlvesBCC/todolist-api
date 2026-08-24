package handlers

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

type mockWorkspaceProvider struct {
	createFunc     func(ownerID uint, name, description string) (*models.Workspace, error)
	getWSFunc      func(userID uint) (*models.Workspace, error)
	getRoleFunc    func(userID uint) (models.WorkspaceRole, error)
	updateFunc     func(ownerID uint, name, description string) (*models.Workspace, error)
	inviteFunc     func(requesterID uint, role models.WorkspaceRole) (string, error)
	acceptFunc     func(userID uint, code string) (*models.Workspace, error)
	membersFunc    func(requesterID uint) ([]models.WorkspaceMember, error)
	invitesFunc    func(ownerID uint) ([]models.WorkspaceInvite, error)
	removeFunc     func(ownerID, targetUserID uint) error
	previewFunc    func(code string) (*models.WorkspaceInvite, *models.Workspace, error)
}

func (m *mockWorkspaceProvider) CreateWorkspace(ownerID uint, name, description string) (*models.Workspace, error) {
	if m.createFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.createFunc(ownerID, name, description)
}
func (m *mockWorkspaceProvider) GetMyWorkspace(userID uint) (*models.Workspace, error) {
	if m.getWSFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.getWSFunc(userID)
}
func (m *mockWorkspaceProvider) GetMyRole(userID uint) (models.WorkspaceRole, error) {
	if m.getRoleFunc == nil {
		return "", errors.New("not implemented")
	}
	return m.getRoleFunc(userID)
}
func (m *mockWorkspaceProvider) UpdateWorkspace(ownerID uint, name, description string) (*models.Workspace, error) {
	if m.updateFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.updateFunc(ownerID, name, description)
}
func (m *mockWorkspaceProvider) GenerateInvite(requesterID uint, role models.WorkspaceRole) (string, error) {
	if m.inviteFunc == nil {
		return "", errors.New("not implemented")
	}
	return m.inviteFunc(requesterID, role)
}
func (m *mockWorkspaceProvider) AcceptInvite(userID uint, code string) (*models.Workspace, error) {
	if m.acceptFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.acceptFunc(userID, code)
}
func (m *mockWorkspaceProvider) ListMembers(requesterID uint) ([]models.WorkspaceMember, error) {
	if m.membersFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.membersFunc(requesterID)
}
func (m *mockWorkspaceProvider) ListInvites(ownerID uint) ([]models.WorkspaceInvite, error) {
	if m.invitesFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.invitesFunc(ownerID)
}
func (m *mockWorkspaceProvider) RemoveMember(ownerID, targetUserID uint) error {
	if m.removeFunc == nil {
		return errors.New("not implemented")
	}
	return m.removeFunc(ownerID, targetUserID)
}
func (m *mockWorkspaceProvider) GetInvitePreview(code string) (*models.WorkspaceInvite, *models.Workspace, error) {
	if m.previewFunc == nil {
		return nil, nil, errors.New("not implemented")
	}
	return m.previewFunc(code)
}

func setupWorkspaceRouter(p *mockWorkspaceProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewWorkspaceHandler(p)
	router.GET("/invites/:code/preview", h.InvitePreview)
	g := router.Group("/api")
	g.Use(withUserContext(1))
	{
		g.POST("/workspace", h.Create)
		g.GET("/workspace", h.Get)
		g.PUT("/workspace", h.Update)
		g.POST("/workspace/invites", h.GenerateInvite)
		g.GET("/workspace/invites", h.ListInvites)
		g.POST("/invites/:code/accept", h.AcceptInvite)
		g.GET("/workspace/members", h.ListMembers)
		g.DELETE("/workspace/members/:userId", h.RemoveMember)
	}
	return router
}

func sampleWorkspace() *models.Workspace {
	return &models.Workspace{Base: models.Base{ID: 1}, Name: "Oficina Centro", OwnerID: 1}
}

func TestWorkspaceHandler_Create_Success(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		createFunc: func(ownerID uint, name, description string) (*models.Workspace, error) {
			return &models.Workspace{Name: name, Description: description, OwnerID: ownerID}, nil
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/workspace", map[string]string{"name": "Oficina Centro", "description": "x"})
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestWorkspaceHandler_Create_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupWorkspaceRouter(&mockWorkspaceProvider{}), http.MethodPost, "/api/workspace", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_Create_ServiceErrorReturns400(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		createFunc: func(ownerID uint, name, description string) (*models.Workspace, error) {
			return nil, errors.New("você já possui um workspace")
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/workspace", map[string]string{"name": "Oficina"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_Get_Success(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		getWSFunc:   func(userID uint) (*models.Workspace, error) { return sampleWorkspace(), nil },
		getRoleFunc: func(userID uint) (models.WorkspaceRole, error) { return models.RoleEditor, nil },
	})
	rec := jsonRequest(router, http.MethodGet, "/api/workspace", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"role":"editor"`)
}

func TestWorkspaceHandler_Get_WorkspaceNotFoundReturns404(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		getWSFunc: func(userID uint) (*models.Workspace, error) { return nil, errors.New("workspace não encontrado") },
	})
	rec := jsonRequest(router, http.MethodGet, "/api/workspace", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWorkspaceHandler_Get_RoleNotFoundReturns404(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		getWSFunc:   func(userID uint) (*models.Workspace, error) { return sampleWorkspace(), nil },
		getRoleFunc: func(userID uint) (models.WorkspaceRole, error) { return "", errors.New("workspace não encontrado") },
	})
	rec := jsonRequest(router, http.MethodGet, "/api/workspace", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWorkspaceHandler_Update_Success(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		updateFunc: func(ownerID uint, name, description string) (*models.Workspace, error) {
			return &models.Workspace{Name: name, Description: description, OwnerID: ownerID}, nil
		},
	})
	rec := jsonRequest(router, http.MethodPut, "/api/workspace", map[string]string{"name": "Nova", "description": ""})
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWorkspaceHandler_Update_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupWorkspaceRouter(&mockWorkspaceProvider{}), http.MethodPut, "/api/workspace", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_Update_ServiceErrorReturns400(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		updateFunc: func(ownerID uint, name, description string) (*models.Workspace, error) {
			return nil, errors.New("workspace não encontrado")
		},
	})
	rec := jsonRequest(router, http.MethodPut, "/api/workspace", map[string]string{"name": "Nova"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_GenerateInvite_Success(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		inviteFunc: func(requesterID uint, role models.WorkspaceRole) (string, error) { return "abc-code", nil },
	})
	rec := jsonRequest(router, http.MethodPost, "/api/workspace/invites", map[string]string{"role": "editor"})
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "abc-code")
}

func TestWorkspaceHandler_GenerateInvite_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupWorkspaceRouter(&mockWorkspaceProvider{}), http.MethodPost, "/api/workspace/invites", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_GenerateInvite_ForbiddenReturns403(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		inviteFunc: func(requesterID uint, role models.WorkspaceRole) (string, error) {
			return "", errors.New("apenas o dono pode gerar convites")
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/workspace/invites", map[string]string{"role": "manager"})
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWorkspaceHandler_InvitePreview_Success(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		previewFunc: func(code string) (*models.WorkspaceInvite, *models.Workspace, error) {
			return &models.WorkspaceInvite{Role: models.RoleEditor, ExpiresAt: exp}, sampleWorkspace(), nil
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/invites/xyz/preview", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Oficina Centro")
}

func TestWorkspaceHandler_InvitePreview_NotFoundReturns404(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		previewFunc: func(code string) (*models.WorkspaceInvite, *models.Workspace, error) {
			return nil, nil, errors.New("convite não encontrado")
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/invites/xyz/preview", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWorkspaceHandler_AcceptInvite_Success(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		acceptFunc: func(userID uint, code string) (*models.Workspace, error) { return sampleWorkspace(), nil },
	})
	rec := jsonRequest(router, http.MethodPost, "/api/invites/xyz/accept", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWorkspaceHandler_AcceptInvite_ServiceErrorReturns400(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		acceptFunc: func(userID uint, code string) (*models.Workspace, error) {
			return nil, errors.New("convite expirado")
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/invites/xyz/accept", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_ListMembers_Success(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		membersFunc: func(requesterID uint) ([]models.WorkspaceMember, error) {
			return []models.WorkspaceMember{
				{
					UserID: 1,
					Role:   models.RoleOwner,
					User:   &models.User{Base: models.Base{ID: 1}, Username: "dono@oficina.com"},
				},
				{
					UserID: 7,
					Role:   models.RoleEditor,
					User:   &models.User{Base: models.Base{ID: 7}, Username: "mecanico@teste.com"},
				},
			}, nil
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/workspace/members", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `"username":"dono@oficina.com"`)
	assert.Contains(t, body, `"username":"mecanico@teste.com"`)
	assert.Contains(t, body, `"role":"owner"`)
	assert.Contains(t, body, `"role":"editor"`)
	assert.NotContains(t, body, `"password"`)
}

func TestWorkspaceHandler_ListMembers_NotFoundReturns404(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		membersFunc: func(requesterID uint) ([]models.WorkspaceMember, error) {
			return nil, errors.New("workspace não encontrado")
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/workspace/members", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWorkspaceHandler_ListInvites_Success(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		invitesFunc: func(ownerID uint) ([]models.WorkspaceInvite, error) {
			return []models.WorkspaceInvite{{Code: "abc"}}, nil
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/workspace/invites", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWorkspaceHandler_ListInvites_ForbiddenReturns403(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		invitesFunc: func(ownerID uint) ([]models.WorkspaceInvite, error) {
			return nil, errors.New("apenas o dono pode ver os convites")
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/workspace/invites", nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWorkspaceHandler_RemoveMember_Success(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		removeFunc: func(ownerID, targetUserID uint) error { return nil },
	})
	rec := jsonRequest(router, http.MethodDelete, "/api/workspace/members/5", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestWorkspaceHandler_RemoveMember_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupWorkspaceRouter(&mockWorkspaceProvider{}), http.MethodDelete, "/api/workspace/members/abc", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWorkspaceHandler_RemoveMember_ForbiddenReturns403(t *testing.T) {
	router := setupWorkspaceRouter(&mockWorkspaceProvider{
		removeFunc: func(ownerID, targetUserID uint) error {
			return errors.New("apenas o dono pode remover membros")
		},
	})
	rec := jsonRequest(router, http.MethodDelete, "/api/workspace/members/5", nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
