package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

type mockAssignmentProvider struct {
	assignFunc   func(requesterID, taskListID, targetUserID uint) error
	unassignFunc func(requesterID, taskListID, targetUserID uint) error
	listFunc     func(taskListID uint) ([]models.ListAssignment, error)
}

func (m *mockAssignmentProvider) Assign(requesterID, taskListID, targetUserID uint) error {
	if m.assignFunc == nil {
		return errors.New("not implemented")
	}
	return m.assignFunc(requesterID, taskListID, targetUserID)
}
func (m *mockAssignmentProvider) Unassign(requesterID, taskListID, targetUserID uint) error {
	if m.unassignFunc == nil {
		return errors.New("not implemented")
	}
	return m.unassignFunc(requesterID, taskListID, targetUserID)
}
func (m *mockAssignmentProvider) List(taskListID uint) ([]models.ListAssignment, error) {
	if m.listFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.listFunc(taskListID)
}

func setupAssignmentRouter(p *mockAssignmentProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewAssignmentHandler(p)
	g := router.Group("/api")
	g.Use(withUserContext(1))
	{
		g.GET("/lists/:id/assignments", h.List)
		g.POST("/lists/:id/assignments", h.Assign)
		g.DELETE("/lists/:id/assignments/:userId", h.Unassign)
	}
	return router
}

func TestAssignmentHandler_Assign_Success(t *testing.T) {
	router := setupAssignmentRouter(&mockAssignmentProvider{
		assignFunc: func(requesterID, taskListID, targetUserID uint) error { return nil },
	})
	rec := jsonRequest(router, http.MethodPost, "/api/lists/3/assignments", map[string]uint{"user_id": 5})
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestAssignmentHandler_Assign_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupAssignmentRouter(&mockAssignmentProvider{}), http.MethodPost, "/api/lists/abc/assignments", map[string]uint{"user_id": 5})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssignmentHandler_Assign_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupAssignmentRouter(&mockAssignmentProvider{}), http.MethodPost, "/api/lists/3/assignments", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssignmentHandler_Assign_ServiceErrorReturns400(t *testing.T) {
	router := setupAssignmentRouter(&mockAssignmentProvider{
		assignFunc: func(requesterID, taskListID, targetUserID uint) error {
			return errors.New("apenas gerentes podem atribuir mecânicos")
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/lists/3/assignments", map[string]uint{"user_id": 5})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssignmentHandler_Unassign_Success(t *testing.T) {
	router := setupAssignmentRouter(&mockAssignmentProvider{
		unassignFunc: func(requesterID, taskListID, targetUserID uint) error { return nil },
	})
	rec := jsonRequest(router, http.MethodDelete, "/api/lists/3/assignments/5", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestAssignmentHandler_Unassign_InvalidListIDReturns400(t *testing.T) {
	rec := jsonRequest(setupAssignmentRouter(&mockAssignmentProvider{}), http.MethodDelete, "/api/lists/abc/assignments/5", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssignmentHandler_Unassign_InvalidUserIDReturns400(t *testing.T) {
	rec := jsonRequest(setupAssignmentRouter(&mockAssignmentProvider{}), http.MethodDelete, "/api/lists/3/assignments/abc", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssignmentHandler_Unassign_ForbiddenReturns403(t *testing.T) {
	router := setupAssignmentRouter(&mockAssignmentProvider{
		unassignFunc: func(requesterID, taskListID, targetUserID uint) error {
			return errors.New("apenas gerentes podem remover atribuições")
		},
	})
	rec := jsonRequest(router, http.MethodDelete, "/api/lists/3/assignments/5", nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAssignmentHandler_List_Success(t *testing.T) {
	router := setupAssignmentRouter(&mockAssignmentProvider{
		listFunc: func(taskListID uint) ([]models.ListAssignment, error) {
			return []models.ListAssignment{{TaskListID: taskListID, UserID: 5}}, nil
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/lists/3/assignments", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAssignmentHandler_List_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupAssignmentRouter(&mockAssignmentProvider{}), http.MethodGet, "/api/lists/abc/assignments", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssignmentHandler_List_ServiceErrorReturns500(t *testing.T) {
	router := setupAssignmentRouter(&mockAssignmentProvider{
		listFunc: func(taskListID uint) ([]models.ListAssignment, error) { return nil, errors.New("db") },
	})
	rec := jsonRequest(router, http.MethodGet, "/api/lists/3/assignments", nil)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
