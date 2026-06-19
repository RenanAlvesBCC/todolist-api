package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/todolist-api/internal/models"
	"github.com/RenanAlvesBCC/todolist-api/internal/services"
)

type mockTaskListProvider struct {
	createListFunc func(userID uint, title string) (*models.TaskList, error)
	listAllFunc    func(userID uint, search string, page, limit int) (*services.PaginatedTaskLists, error)
	getListFunc    func(listID, userID uint) (*models.TaskList, error)
	updateListFunc func(listID, userID uint, title string) (*models.TaskList, error)
	deleteListFunc func(listID, userID uint) error
	addItemFunc    func(listID, userID uint, text string) (*models.TaskItem, error)
	updateItemFunc func(listID, itemID, userID uint, text string, completed bool) (*models.TaskItem, error)
	deleteItemFunc func(listID, itemID, userID uint) error
}

func (m *mockTaskListProvider) CreateList(userID uint, title string) (*models.TaskList, error) {
	return m.createListFunc(userID, title)
}
func (m *mockTaskListProvider) ListAll(userID uint, search string, page, limit int) (*services.PaginatedTaskLists, error) {
	return m.listAllFunc(userID, search, page, limit)
}
func (m *mockTaskListProvider) GetList(listID, userID uint) (*models.TaskList, error) {
	return m.getListFunc(listID, userID)
}
func (m *mockTaskListProvider) UpdateList(listID, userID uint, title string) (*models.TaskList, error) {
	return m.updateListFunc(listID, userID, title)
}
func (m *mockTaskListProvider) DeleteList(listID, userID uint) error {
	return m.deleteListFunc(listID, userID)
}
func (m *mockTaskListProvider) AddItem(listID, userID uint, text string) (*models.TaskItem, error) {
	return m.addItemFunc(listID, userID, text)
}
func (m *mockTaskListProvider) UpdateItem(listID, itemID, userID uint, text string, completed bool) (*models.TaskItem, error) {
	return m.updateItemFunc(listID, itemID, userID, text, completed)
}
func (m *mockTaskListProvider) DeleteItem(listID, itemID, userID uint) error {
	return m.deleteItemFunc(listID, itemID, userID)
}

// withUserContext simula o que o middleware de autenticação faria: injeta
// um user_id no contexto antes do handler rodar, sem precisar de um JWT real.
func withUserContext(userID float64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func setupRouter(provider *mockTaskListProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewTaskListHandler(provider)

	group := router.Group("/api")
	group.Use(withUserContext(1))
	{
		group.POST("/lists", handler.Create)
		group.GET("/lists", handler.List)
		group.GET("/lists/:id", handler.Get)
		group.PUT("/lists/:id", handler.Update)
		group.DELETE("/lists/:id", handler.Delete)
		group.POST("/lists/:id/items", handler.AddItem)
		group.PUT("/lists/:id/items/:itemId", handler.UpdateItem)
		group.DELETE("/lists/:id/items/:itemId", handler.DeleteItem)
	}
	return router
}

func TestTaskListHandler_Create_Success(t *testing.T) {
	provider := &mockTaskListProvider{
		createListFunc: func(userID uint, title string) (*models.TaskList, error) {
			return &models.TaskList{Title: title, UserID: userID}, nil
		},
	}
	router := setupRouter(provider)

	body, _ := json.Marshal(map[string]string{"title": "Compras da semana"})
	req := httptest.NewRequest(http.MethodPost, "/api/lists", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response models.TaskList
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "Compras da semana", response.Title)
}

func TestTaskListHandler_Create_MissingTitleReturns400(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{})

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/lists", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_Get_NotFoundReturns404(t *testing.T) {
	provider := &mockTaskListProvider{
		getListFunc: func(listID, userID uint) (*models.TaskList, error) {
			return nil, errors.New("lista não encontrada")
		},
	}
	router := setupRouter(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/lists/999", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTaskListHandler_Delete_Success(t *testing.T) {
	var deletedID uint
	provider := &mockTaskListProvider{
		deleteListFunc: func(listID, userID uint) error {
			deletedID = listID
			return nil
		},
	}
	router := setupRouter(provider)

	req := httptest.NewRequest(http.MethodDelete, "/api/lists/7", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, uint(7), deletedID)
}

func TestTaskListHandler_AddItem_Success(t *testing.T) {
	provider := &mockTaskListProvider{
		addItemFunc: func(listID, userID uint, text string) (*models.TaskItem, error) {
			return &models.TaskItem{Text: text, TaskListID: listID}, nil
		},
	}
	router := setupRouter(provider)

	body, _ := json.Marshal(map[string]string{"text": "Leite"})
	req := httptest.NewRequest(http.MethodPost, "/api/lists/1/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var response models.TaskItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "Leite", response.Text)
}
