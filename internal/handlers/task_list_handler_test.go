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

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
)

type mockTaskListProvider struct {
	createListFunc   func(userID uint, title, plate, customer string) (*models.TaskList, error)
	listAllFunc      func(userID uint, search string, page, limit int, status string, mine bool) (*services.PaginatedTaskLists, error)
	getListFunc      func(listID, userID uint) (*models.TaskList, error)
	updateListFunc   func(listID, userID uint, title, plate, customer string) (*models.TaskList, error)
	deleteListFunc   func(listID, userID uint) error
	addItemFunc      func(listID, userID uint, text string) (*models.TaskItem, error)
	updateItemFunc   func(listID, itemID, userID uint, text string, completed bool) (*models.TaskItem, error)
	deleteItemFunc   func(listID, itemID, userID uint) error
	reorderLists       func(userID uint, orderedIDs []uint) error
	reorderItems       func(listID, userID uint, orderedIDs []uint) error
	reorderListsFunc   func(userID uint, orderedIDs []uint) error
	reorderItemsFunc   func(listID, userID uint, orderedIDs []uint) error
	changeStatusFunc   func(listID, userID uint, newStatus models.TaskListStatus) error
}

func (m *mockTaskListProvider) CreateList(userID uint, title, plate, customer string) (*models.TaskList, error) {
	if m.createListFunc == nil {
		return nil, nil
	}
	return m.createListFunc(userID, title, plate, customer)
}
func (m *mockTaskListProvider) ListAll(userID uint, search string, page, limit int, status string, mine bool) (*services.PaginatedTaskLists, error) {
	if m.listAllFunc == nil {
		return &services.PaginatedTaskLists{}, nil
	}
	return m.listAllFunc(userID, search, page, limit, status, mine)
}
func (m *mockTaskListProvider) GetList(listID, userID uint) (*models.TaskList, error) {
	if m.getListFunc == nil {
		return nil, nil
	}
	return m.getListFunc(listID, userID)
}
func (m *mockTaskListProvider) UpdateList(listID, userID uint, title, plate, customer string) (*models.TaskList, error) {
	if m.updateListFunc == nil {
		return nil, nil
	}
	return m.updateListFunc(listID, userID, title, plate, customer)
}
func (m *mockTaskListProvider) DeleteList(listID, userID uint) error {
	if m.deleteListFunc == nil {
		return nil
	}
	return m.deleteListFunc(listID, userID)
}
func (m *mockTaskListProvider) AddItem(listID, userID uint, text string) (*models.TaskItem, error) {
	if m.addItemFunc == nil {
		return nil, nil
	}
	return m.addItemFunc(listID, userID, text)
}
func (m *mockTaskListProvider) UpdateItem(listID, itemID, userID uint, text string, completed bool) (*models.TaskItem, error) {
	if m.updateItemFunc == nil {
		return nil, nil
	}
	return m.updateItemFunc(listID, itemID, userID, text, completed)
}
func (m *mockTaskListProvider) DeleteItem(listID, itemID, userID uint) error {
	if m.deleteItemFunc == nil {
		return nil
	}
	return m.deleteItemFunc(listID, itemID, userID)
}

func (m *mockTaskListProvider) ReorderLists(userID uint, orderedIDs []uint) error {
	if m.reorderListsFunc == nil {
		return nil
	}
	return m.reorderListsFunc(userID, orderedIDs)
}
func (m *mockTaskListProvider) ReorderItems(listID, userID uint, orderedIDs []uint) error {
	if m.reorderItemsFunc == nil {
		return nil
	}
	return m.reorderItemsFunc(listID, userID, orderedIDs)
}
func (m *mockTaskListProvider) ChangeStatus(listID, userID uint, newStatus models.TaskListStatus) error {
	if m.changeStatusFunc == nil {
		return nil
	}
	return m.changeStatusFunc(listID, userID, newStatus)
}
func (m *mockTaskListProvider) AssignMember(listID, requesterID, targetUserID uint) error {
	return nil
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
		group.PUT("/lists/reorder", handler.ReorderLists)
		group.GET("/lists/:id", handler.Get)
		group.PUT("/lists/:id", handler.Update)
		group.DELETE("/lists/:id", handler.Delete)
		group.POST("/lists/:id/items", handler.AddItem)
		group.PUT("/lists/:id/items/reorder", handler.ReorderItems)
		group.PUT("/lists/:id/items/:itemId", handler.UpdateItem)
		group.DELETE("/lists/:id/items/:itemId", handler.DeleteItem)
		group.PUT("/lists/:id/status", handler.ChangeStatus)
	}
	return router
}

func TestTaskListHandler_Create_Success(t *testing.T) {
	provider := &mockTaskListProvider{
		createListFunc: func(userID uint, title, plate, customer string) (*models.TaskList, error) {
			return &models.TaskList{Title: title, Plate: plate, Customer: customer, UserID: userID}, nil
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

func TestTaskListHandler_Create_ForwardsPlateAndCustomer(t *testing.T) {
	var gotPlate, gotCustomer string
	provider := &mockTaskListProvider{
		createListFunc: func(userID uint, title, plate, customer string) (*models.TaskList, error) {
			gotPlate, gotCustomer = plate, customer
			return &models.TaskList{Title: title, Plate: plate, Customer: customer, UserID: userID}, nil
		},
	}
	router := setupRouter(provider)

	body, _ := json.Marshal(map[string]string{
		"title":    "Fusca",
		"plate":    "abc-1234",
		"customer": "Ana",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/lists", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "abc-1234", gotPlate)
	assert.Equal(t, "Ana", gotCustomer)
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

func TestTaskListHandler_ReorderLists_Success(t *testing.T) {
	var receivedIDs []uint
	provider := &mockTaskListProvider{
		reorderListsFunc: func(userID uint, orderedIDs []uint) error {
			receivedIDs = orderedIDs
			return nil
		},
	}
	router := setupRouter(provider)

	body, _ := json.Marshal(map[string][]uint{"ids": {3, 1, 2}})
	req := httptest.NewRequest(http.MethodPut, "/api/lists/reorder", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, []uint{3, 1, 2}, receivedIDs)
}

func jsonRequest(router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestTaskListHandler_Create_ErrNotManagerReturns403(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		createListFunc: func(userID uint, title, plate, customer string) (*models.TaskList, error) {
			return nil, services.ErrNotManager
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/lists", map[string]string{"title": "Fusca"})
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTaskListHandler_Create_ServiceErrorReturns400(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		createListFunc: func(userID uint, title, plate, customer string) (*models.TaskList, error) {
			return nil, errors.New("título é obrigatório")
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/lists", map[string]string{"title": "x"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_List_Success(t *testing.T) {
	var gotSearch, gotStatus string
	var gotPage, gotLimit int
	var gotMine bool
	router := setupRouter(&mockTaskListProvider{
		listAllFunc: func(userID uint, search string, page, limit int, status string, mine bool) (*services.PaginatedTaskLists, error) {
			gotSearch, gotPage, gotLimit, gotStatus, gotMine = search, page, limit, status, mine
			return &services.PaginatedTaskLists{
				Lists: []models.TaskList{{Title: "Fusca", UserID: userID}},
				Page:  page, Limit: limit, Total: 1, TotalPages: 1,
			}, nil
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/lists?search=fusca&page=2&limit=10&status=em_andamento&mine=true", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "fusca", gotSearch)
	assert.Equal(t, 2, gotPage)
	assert.Equal(t, 10, gotLimit)
	assert.Equal(t, "em_andamento", gotStatus)
	assert.True(t, gotMine)
}

func TestTaskListHandler_List_ServiceErrorReturns500(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		listAllFunc: func(userID uint, search string, page, limit int, status string, mine bool) (*services.PaginatedTaskLists, error) {
			return nil, errors.New("db")
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/lists", nil)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestTaskListHandler_Get_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodGet, "/api/lists/abc", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_Get_Success(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		getListFunc: func(listID, userID uint) (*models.TaskList, error) {
			return &models.TaskList{Title: "Gol", UserID: userID}, nil
		},
	})
	rec := jsonRequest(router, http.MethodGet, "/api/lists/3", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskListHandler_Update_Success(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		updateListFunc: func(listID, userID uint, title, plate, customer string) (*models.TaskList, error) {
			return &models.TaskList{Title: title, Plate: plate, Customer: customer, UserID: userID}, nil
		},
	})
	rec := jsonRequest(router, http.MethodPut, "/api/lists/3", map[string]string{
		"title":    "Novo título",
		"plate":    "ABC1D23",
		"customer": "Maria",
	})
	assert.Equal(t, http.StatusOK, rec.Code)

	var response models.TaskList
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, "Novo título", response.Title)
	assert.Equal(t, "ABC1D23", response.Plate)
	assert.Equal(t, "Maria", response.Customer)
}

func TestTaskListHandler_Update_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPut, "/api/lists/abc", map[string]string{"title": "x"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_Update_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPut, "/api/lists/3", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_Update_ErrNotManagerReturns403(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		updateListFunc: func(listID, userID uint, title, plate, customer string) (*models.TaskList, error) {
			return nil, services.ErrNotManager
		},
	})
	rec := jsonRequest(router, http.MethodPut, "/api/lists/3", map[string]string{"title": "x"})
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTaskListHandler_Update_NotFoundReturns404(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		updateListFunc: func(listID, userID uint, title, plate, customer string) (*models.TaskList, error) {
			return nil, errors.New("lista não encontrada")
		},
	})
	rec := jsonRequest(router, http.MethodPut, "/api/lists/3", map[string]string{"title": "x"})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTaskListHandler_Delete_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodDelete, "/api/lists/abc", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_Delete_ErrNotManagerReturns403(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		deleteListFunc: func(listID, userID uint) error { return services.ErrNotManager },
	})
	rec := jsonRequest(router, http.MethodDelete, "/api/lists/3", nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTaskListHandler_Delete_NotFoundReturns404(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		deleteListFunc: func(listID, userID uint) error { return errors.New("não encontrada") },
	})
	rec := jsonRequest(router, http.MethodDelete, "/api/lists/3", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTaskListHandler_AddItem_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPost, "/api/lists/abc/items", map[string]string{"text": "x"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_AddItem_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPost, "/api/lists/1/items", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_AddItem_NotFoundReturns404(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		addItemFunc: func(listID, userID uint, text string) (*models.TaskItem, error) {
			return nil, errors.New("lista não encontrada")
		},
	})
	rec := jsonRequest(router, http.MethodPost, "/api/lists/1/items", map[string]string{"text": "x"})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTaskListHandler_UpdateItem_Success(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		updateItemFunc: func(listID, itemID, userID uint, text string, completed bool) (*models.TaskItem, error) {
			return &models.TaskItem{Text: text, Completed: completed, TaskListID: listID}, nil
		},
	})
	rec := jsonRequest(router, http.MethodPut, "/api/lists/1/items/2", map[string]any{"text": "Leite", "completed": true})
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskListHandler_UpdateItem_InvalidListIDReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPut, "/api/lists/abc/items/2", map[string]any{"text": "x"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_UpdateItem_InvalidItemIDReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPut, "/api/lists/1/items/abc", map[string]any{"text": "x"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_UpdateItem_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPut, "/api/lists/1/items/2", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_UpdateItem_NotFoundReturns404(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		updateItemFunc: func(listID, itemID, userID uint, text string, completed bool) (*models.TaskItem, error) {
			return nil, errors.New("item não encontrado")
		},
	})
	rec := jsonRequest(router, http.MethodPut, "/api/lists/1/items/2", map[string]any{"text": "x"})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTaskListHandler_DeleteItem_Success(t *testing.T) {
	var gotList, gotItem uint
	router := setupRouter(&mockTaskListProvider{
		deleteItemFunc: func(listID, itemID, userID uint) error {
			gotList, gotItem = listID, itemID
			return nil
		},
	})
	rec := jsonRequest(router, http.MethodDelete, "/api/lists/4/items/9", nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, uint(4), gotList)
	assert.Equal(t, uint(9), gotItem)
}

func TestTaskListHandler_DeleteItem_InvalidListIDReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodDelete, "/api/lists/abc/items/1", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_DeleteItem_InvalidItemIDReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodDelete, "/api/lists/1/items/abc", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_DeleteItem_NotFoundReturns404(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		deleteItemFunc: func(listID, itemID, userID uint) error { return errors.New("item não encontrado") },
	})
	rec := jsonRequest(router, http.MethodDelete, "/api/lists/1/items/2", nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTaskListHandler_ReorderLists_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPut, "/api/lists/reorder", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_ReorderLists_ErrNotManagerReturns403(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		reorderListsFunc: func(userID uint, orderedIDs []uint) error { return services.ErrNotManager },
	})
	rec := jsonRequest(router, http.MethodPut, "/api/lists/reorder", map[string][]uint{"ids": {1, 2}})
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTaskListHandler_ReorderLists_ServiceErrorReturns400(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		reorderListsFunc: func(userID uint, orderedIDs []uint) error { return errors.New("lista de ids vazia") },
	})
	rec := jsonRequest(router, http.MethodPut, "/api/lists/reorder", map[string][]uint{"ids": {1}})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_ReorderItems_Success(t *testing.T) {
	var received []uint
	router := setupRouter(&mockTaskListProvider{
		reorderItemsFunc: func(listID, userID uint, orderedIDs []uint) error {
			received = orderedIDs
			return nil
		},
	})
	rec := jsonRequest(router, http.MethodPut, "/api/lists/1/items/reorder", map[string][]uint{"ids": {11, 10}})
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, []uint{11, 10}, received)
}

func TestTaskListHandler_ReorderItems_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPut, "/api/lists/abc/items/reorder", map[string][]uint{"ids": {1}})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_ReorderItems_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPut, "/api/lists/1/items/reorder", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_ReorderItems_NotFoundReturns404(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		reorderItemsFunc: func(listID, userID uint, orderedIDs []uint) error {
			return errors.New("lista não encontrada")
		},
	})
	rec := jsonRequest(router, http.MethodPut, "/api/lists/1/items/reorder", map[string][]uint{"ids": {1}})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTaskListHandler_ChangeStatus_Success(t *testing.T) {
	var got models.TaskListStatus
	router := setupRouter(&mockTaskListProvider{
		changeStatusFunc: func(listID, userID uint, newStatus models.TaskListStatus) error {
			got = newStatus
			return nil
		},
	})
	rec := jsonRequest(router, http.MethodPut, "/api/lists/1/status", map[string]string{"status": "aguardando_peca"})
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, models.StatusAguardandoPeca, got)
}

func TestTaskListHandler_ChangeStatus_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPut, "/api/lists/abc/status", map[string]string{"status": "em_andamento"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_ChangeStatus_InvalidBodyReturns400(t *testing.T) {
	rec := jsonRequest(setupRouter(&mockTaskListProvider{}), http.MethodPut, "/api/lists/1/status", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskListHandler_ChangeStatus_ForbiddenReturns403(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		changeStatusFunc: func(listID, userID uint, newStatus models.TaskListStatus) error {
			return errors.New("transição de status não permitida para seu papel")
		},
	})
	rec := jsonRequest(router, http.MethodPut, "/api/lists/1/status", map[string]string{"status": "entregue"})
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
