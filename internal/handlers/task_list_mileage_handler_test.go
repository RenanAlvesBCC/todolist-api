package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
)

func TestTaskListHandler_Update_ForwardsAbsentNullAndPresentMileage(t *testing.T) {
	var got services.VehicleInput
	router := setupRouter(&mockTaskListProvider{
		updateListFunc: func(listID, userID uint, in services.VehicleInput) (*models.TaskList, error) {
			got = in
			return &models.TaskList{Title: in.Title}, nil
		},
	})

	rec := rawRequest(router, http.MethodPut, "/api/lists/3", `{"title":"Gol","plate":"ABC1D23","customer":"Maria"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, got.Mileage, "chave ausente mantém")

	rec = rawRequest(router, http.MethodPut, "/api/lists/3", `{"title":"Gol","mileage":null}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, got.Mileage, "null é tratado como ausente")

	for body, want := range map[string]int{
		`{"title":"Gol","mileage":0}`:        0,
		`{"title":"Gol","mileage": 123456 }`: 123456,
		`{"title":"Gol","mileage":-5}`:       -5,
	} {
		rec = rawRequest(router, http.MethodPut, "/api/lists/3", body)
		require.Equal(t, http.StatusOK, rec.Code, body)
		require.NotNil(t, got.Mileage, body)
		assert.Equal(t, want, *got.Mileage, body)
	}
}

func TestTaskListHandler_Create_ForwardsMileage(t *testing.T) {
	var got services.VehicleInput
	router := setupRouter(&mockTaskListProvider{
		createListFunc: func(userID uint, in services.VehicleInput) (*models.TaskList, error) {
			got = in
			return &models.TaskList{Title: in.Title, Mileage: in.Mileage}, nil
		},
	})

	rec := rawRequest(router, http.MethodPost, "/api/lists", `{"title":"Gol","mileage":87000}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.NotNil(t, got.Mileage)
	assert.Equal(t, 87000, *got.Mileage)
	assert.Equal(t, float64(87000), decodeObject(t, rec)["mileage"])

	rec = rawRequest(router, http.MethodPost, "/api/lists", `{"title":"Gol"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Nil(t, got.Mileage)
	body := decodeObject(t, rec)
	assert.Contains(t, body, "mileage")
	assert.Nil(t, body["mileage"])
}

func TestTaskListHandler_NonIntegerMileageReturns400(t *testing.T) {
	called := false
	router := setupRouter(&mockTaskListProvider{
		createListFunc: func(userID uint, in services.VehicleInput) (*models.TaskList, error) {
			called = true
			return &models.TaskList{}, nil
		},
		updateListFunc: func(listID, userID uint, in services.VehicleInput) (*models.TaskList, error) {
			called = true
			return &models.TaskList{}, nil
		},
	})

	for _, value := range []string{`"123"`, `""`, `12.5`, `1e3`, `true`, `{}`, `[1]`, `99999999999999999999999`} {
		body := `{"title":"Gol","mileage":` + value + `}`
		for _, rec := range []*httptest.ResponseRecorder{
			rawRequest(router, http.MethodPost, "/api/lists", body),
			rawRequest(router, http.MethodPut, "/api/lists/3", body),
		} {
			assert.Equal(t, http.StatusBadRequest, rec.Code, value)
			assert.Equal(t, "quilometragem inválida", decodeObject(t, rec)["error"], value)
		}
	}
	assert.False(t, called, "service não é chamado com mileage malformado")
}

func TestTaskListHandler_InvalidMileageFromServiceReturns400(t *testing.T) {
	router := setupRouter(&mockTaskListProvider{
		createListFunc: func(userID uint, in services.VehicleInput) (*models.TaskList, error) {
			return nil, services.ErrInvalidMileage
		},
		updateListFunc: func(listID, userID uint, in services.VehicleInput) (*models.TaskList, error) {
			return nil, services.ErrInvalidMileage
		},
	})

	for _, method := range []string{http.MethodPost, http.MethodPut} {
		path := "/api/lists"
		if method == http.MethodPut {
			path = "/api/lists/3"
		}
		rec := rawRequest(router, method, path, `{"title":"Gol","mileage":-1}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "quilometragem inválida", decodeObject(t, rec)["error"])
	}
}
