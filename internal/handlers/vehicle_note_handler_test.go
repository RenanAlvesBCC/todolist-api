package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
)

type mockNoteProvider struct {
	addFunc    func(listID, userID uint, text string) (*models.VehicleNote, error)
	listFunc   func(listID, userID uint) ([]models.VehicleNote, error)
	deleteFunc func(listID, noteID, userID uint) error
}

func (m *mockNoteProvider) AddNote(listID, userID uint, text string) (*models.VehicleNote, error) {
	if m.addFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.addFunc(listID, userID, text)
}
func (m *mockNoteProvider) ListNotes(listID, userID uint) ([]models.VehicleNote, error) {
	if m.listFunc == nil {
		return nil, errors.New("not implemented")
	}
	return m.listFunc(listID, userID)
}
func (m *mockNoteProvider) DeleteNote(listID, noteID, userID uint) error {
	if m.deleteFunc == nil {
		return errors.New("not implemented")
	}
	return m.deleteFunc(listID, noteID, userID)
}

func setupNoteRouter(p *mockNoteProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewVehicleNoteHandler(p)
	g := router.Group("/api")
	g.Use(withUserContext(1))
	{
		g.GET("/lists/:id/notes", h.List)
		g.POST("/lists/:id/notes", h.Add)
		g.DELETE("/lists/:id/notes/:noteId", h.Delete)
	}
	return router
}

func TestVehicleNoteHandler_Add_Success(t *testing.T) {
	var gotText string
	router := setupNoteRouter(&mockNoteProvider{
		addFunc: func(listID, userID uint, text string) (*models.VehicleNote, error) {
			gotText = text
			return &models.VehicleNote{ID: 9, TaskListID: listID, AuthorID: userID, AuthorName: "Ana Silva", Text: text}, nil
		},
	})

	rec := jsonRequest(router, http.MethodPost, "/api/lists/3/notes", map[string]string{"text": "Linha 1\nLinha 2"})

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "Linha 1\nLinha 2", gotText)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "Ana Silva", body["author_name"])
	assert.Equal(t, float64(3), body["task_list_id"])
	assert.Contains(t, body, "created_at")
	assert.NotContains(t, body, "deleted_at")
	assert.NotContains(t, body, "deleted_by")
}

func TestVehicleNoteHandler_Add_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupNoteRouter(&mockNoteProvider{}), http.MethodPost, "/api/lists/abc/notes", map[string]string{"text": "x"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVehicleNoteHandler_Add_InvalidJSONReturns400(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/lists/3/notes", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	setupNoteRouter(&mockNoteProvider{}).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVehicleNoteHandler_ErrorMapping(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{services.ErrNoteTextRequired, http.StatusBadRequest},
		{services.ErrNoteTextTooLong, http.StatusBadRequest},
		{services.ErrNotFound, http.StatusNotFound},
		{services.ErrNoteNotFound, http.StatusNotFound},
		{services.ErrNoteForbidden, http.StatusForbidden},
		{errors.New("db down"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		router := setupNoteRouter(&mockNoteProvider{
			addFunc:    func(listID, userID uint, text string) (*models.VehicleNote, error) { return nil, tc.err },
			listFunc:   func(listID, userID uint) ([]models.VehicleNote, error) { return nil, tc.err },
			deleteFunc: func(listID, noteID, userID uint) error { return tc.err },
		})

		rec := jsonRequest(router, http.MethodPost, "/api/lists/3/notes", map[string]string{"text": ""})
		assert.Equal(t, tc.code, rec.Code, "POST %v", tc.err)
		rec = jsonRequest(router, http.MethodGet, "/api/lists/3/notes", nil)
		assert.Equal(t, tc.code, rec.Code, "GET %v", tc.err)
		rec = jsonRequest(router, http.MethodDelete, "/api/lists/3/notes/4", nil)
		assert.Equal(t, tc.code, rec.Code, "DELETE %v", tc.err)
	}
}

func TestVehicleNoteHandler_Add_ErrorMessageInPortuguese(t *testing.T) {
	router := setupNoteRouter(&mockNoteProvider{
		addFunc: func(listID, userID uint, text string) (*models.VehicleNote, error) {
			return nil, services.ErrNoteTextTooLong
		},
	})

	rec := jsonRequest(router, http.MethodPost, "/api/lists/3/notes", map[string]string{"text": "x"})

	assert.JSONEq(t, `{"error":"texto deve ter no máximo 1000 caracteres"}`, rec.Body.String())
}

func TestVehicleNoteHandler_List_Success(t *testing.T) {
	router := setupNoteRouter(&mockNoteProvider{
		listFunc: func(listID, userID uint) ([]models.VehicleNote, error) {
			return []models.VehicleNote{{ID: 2, Text: "nova"}, {ID: 1, Text: "antiga"}}, nil
		},
	})

	rec := jsonRequest(router, http.MethodGet, "/api/lists/3/notes", nil)

	require.Equal(t, http.StatusOK, rec.Code)
	var body []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body, 2)
	assert.Equal(t, "nova", body[0]["text"])
}

func TestVehicleNoteHandler_List_InvalidIDReturns400(t *testing.T) {
	rec := jsonRequest(setupNoteRouter(&mockNoteProvider{}), http.MethodGet, "/api/lists/abc/notes", nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVehicleNoteHandler_Delete_Success(t *testing.T) {
	var gotList, gotNote, gotUser uint
	router := setupNoteRouter(&mockNoteProvider{
		deleteFunc: func(listID, noteID, userID uint) error {
			gotList, gotNote, gotUser = listID, noteID, userID
			return nil
		},
	})

	rec := jsonRequest(router, http.MethodDelete, "/api/lists/3/notes/8", nil)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, []uint{3, 8, 1}, []uint{gotList, gotNote, gotUser})
}

func TestVehicleNoteHandler_Delete_InvalidIDsReturn400(t *testing.T) {
	router := setupNoteRouter(&mockNoteProvider{})
	assert.Equal(t, http.StatusBadRequest, jsonRequest(router, http.MethodDelete, "/api/lists/abc/notes/8", nil).Code)
	assert.Equal(t, http.StatusBadRequest, jsonRequest(router, http.MethodDelete, "/api/lists/3/notes/abc", nil).Code)
}
