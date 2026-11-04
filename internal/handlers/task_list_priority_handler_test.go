package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
)

func TestTaskListHandler_ChangePriority_ReturnsUpdatedVehicle(t *testing.T) {
	var gotID uint
	var gotPriority string
	router := setupRouter(&mockTaskListProvider{
		changePriorityFunc: func(listID, userID uint, priority string) (*models.TaskList, error) {
			gotID, gotPriority = listID, priority
			v := vehicleWithCustomerData(false)
			v.Priority = models.VehiclePriority(priority)
			return v, nil
		},
	})

	rec := jsonRequest(router, http.MethodPut, "/api/lists/7/priority", map[string]string{"priority": "alta"})
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uint(7), gotID)
	assert.Equal(t, "alta", gotPriority)
	body := decodeObject(t, rec)
	assert.Equal(t, "alta", body["priority"])
	assert.NotContains(t, body, "customer_cpf")
}

func TestTaskListHandler_ChangePriority_Errors(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		body   any
		svcErr error
		want   int
	}{
		{"id inválido", "/api/lists/abc/priority", map[string]string{"priority": "alta"}, nil, http.StatusBadRequest},
		{"body sem priority", "/api/lists/7/priority", map[string]string{}, nil, http.StatusBadRequest},
		{"prioridade inválida", "/api/lists/7/priority", map[string]string{"priority": "baixa"}, services.ErrInvalidPriority, http.StatusBadRequest},
		{"mecânico", "/api/lists/7/priority", map[string]string{"priority": "alta"}, services.ErrNotManager, http.StatusForbidden},
		{"sem acesso", "/api/lists/7/priority", map[string]string{"priority": "alta"}, services.ErrNotFound, http.StatusNotFound},
		{"erro genérico", "/api/lists/7/priority", map[string]string{"priority": "alta"}, errors.New("db"), http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := setupRouter(&mockTaskListProvider{
				changePriorityFunc: func(listID, userID uint, priority string) (*models.TaskList, error) {
					return nil, tc.svcErr
				},
			})
			rec := jsonRequest(router, http.MethodPut, tc.path, tc.body)
			assert.Equal(t, tc.want, rec.Code)
		})
	}
}

func TestTaskListHandler_CreateAndUpdate_ForwardPriority(t *testing.T) {
	var created, updated services.VehicleInput
	router := setupRouter(&mockTaskListProvider{
		createListFunc: func(userID uint, in services.VehicleInput) (*models.TaskList, error) {
			created = in
			return &models.TaskList{}, nil
		},
		updateListFunc: func(listID, userID uint, in services.VehicleInput) (*models.TaskList, error) {
			updated = in
			return &models.TaskList{}, nil
		},
	})

	require.Equal(t, http.StatusCreated, rawRequest(router, http.MethodPost, "/api/lists", `{"title":"Gol","priority":"urgente"}`).Code)
	require.NotNil(t, created.Priority)
	assert.Equal(t, "urgente", *created.Priority)

	require.Equal(t, http.StatusOK, rawRequest(router, http.MethodPut, "/api/lists/7", `{"title":"Gol"}`).Code)
	assert.Nil(t, updated.Priority, "PUT sem priority mantém a atual")
}
