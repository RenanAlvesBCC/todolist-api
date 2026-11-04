package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
)

var customerDataKeys = []string{"customer_cpf", "customer_cep", "customer_whatsapp"}

func vehicleWithCustomerData(visible bool) *models.TaskList {
	return &models.TaskList{
		Base:                models.Base{ID: 7},
		Title:               "Gol",
		Customer:            "Maria",
		CustomerCPF:         "12345678909",
		CustomerCEP:         "01310100",
		CustomerWhatsApp:    "11987654321",
		CustomerDataVisible: visible,
	}
}

func decodeObject(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

func rawRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestTaskListHandler_CustomerDataFollowsServiceVisibility(t *testing.T) {
	for _, visible := range []bool{true, false} {
		router := setupRouter(&mockTaskListProvider{
			getListFunc: func(listID, userID uint) (*models.TaskList, error) {
				return vehicleWithCustomerData(visible), nil
			},
			listAllFunc: func(userID uint, search string, page, limit int, status string, mine bool) (*services.PaginatedTaskLists, error) {
				return &services.PaginatedTaskLists{Lists: []models.TaskList{*vehicleWithCustomerData(visible)}, Page: 1, Limit: 20, Total: 1, TotalPages: 1}, nil
			},
			updateListFunc: func(listID, userID uint, in services.VehicleInput) (*models.TaskList, error) {
				return vehicleWithCustomerData(visible), nil
			},
			createListFunc: func(userID uint, in services.VehicleInput) (*models.TaskList, error) {
				return vehicleWithCustomerData(visible), nil
			},
		})

		detail := decodeObject(t, jsonRequest(router, http.MethodGet, "/api/lists/7", nil))
		updated := decodeObject(t, jsonRequest(router, http.MethodPut, "/api/lists/7", map[string]string{"title": "Gol"}))
		created := decodeObject(t, jsonRequest(router, http.MethodPost, "/api/lists", map[string]string{"title": "Gol"}))
		var page struct {
			Lists []map[string]any `json:"lists"`
		}
		require.NoError(t, json.Unmarshal(jsonRequest(router, http.MethodGet, "/api/lists", nil).Body.Bytes(), &page))
		require.Len(t, page.Lists, 1)

		for _, body := range []map[string]any{detail, updated, created, page.Lists[0]} {
			assert.Equal(t, "Maria", body["customer"])
			for _, key := range customerDataKeys {
				if visible {
					assert.Contains(t, body, key)
				} else {
					assert.NotContains(t, body, key, "mecânico não recebe "+key)
				}
			}
			if visible {
				assert.Equal(t, "12345678909", body["customer_cpf"])
				assert.Equal(t, "01310100", body["customer_cep"])
				assert.Equal(t, "11987654321", body["customer_whatsapp"])
			}
		}
	}
}

func TestTaskListHandler_Update_ForwardsAbsentAndPresentCustomerKeys(t *testing.T) {
	var got services.VehicleInput
	router := setupRouter(&mockTaskListProvider{
		updateListFunc: func(listID, userID uint, in services.VehicleInput) (*models.TaskList, error) {
			got = in
			return &models.TaskList{Title: in.Title}, nil
		},
	})

	rec := rawRequest(router, http.MethodPut, "/api/lists/3", `{"title":"Gol","plate":"ABC1D23","customer":"Maria"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, got.CustomerCPF)
	assert.Nil(t, got.CustomerCEP)
	assert.Nil(t, got.CustomerWhatsApp)

	rec = rawRequest(router, http.MethodPut, "/api/lists/3", `{"title":"Gol","customer_cpf":"","customer_cep":"01310-100","customer_whatsapp":null}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, got.CustomerCPF)
	assert.Equal(t, "", *got.CustomerCPF)
	require.NotNil(t, got.CustomerCEP)
	assert.Equal(t, "01310-100", *got.CustomerCEP)
	assert.Nil(t, got.CustomerWhatsApp, "null é tratado como ausente")
}

func TestTaskListHandler_Create_ForwardsCustomerData(t *testing.T) {
	var got services.VehicleInput
	router := setupRouter(&mockTaskListProvider{
		createListFunc: func(userID uint, in services.VehicleInput) (*models.TaskList, error) {
			got = in
			return &models.TaskList{Title: in.Title}, nil
		},
	})

	rec := rawRequest(router, http.MethodPost, "/api/lists", `{"title":"Gol","customer_cpf":"123.456.789-09","customer_whatsapp":"(11) 98765-4321"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.NotNil(t, got.CustomerCPF)
	assert.Equal(t, "123.456.789-09", *got.CustomerCPF)
	assert.Nil(t, got.CustomerCEP)
	require.NotNil(t, got.CustomerWhatsApp)
}

func TestTaskListHandler_InvalidCustomerDataReturns400(t *testing.T) {
	for _, svcErr := range []error{services.ErrInvalidCPF, services.ErrInvalidCEP, services.ErrInvalidWhatsApp, services.ErrTitleRequired} {
		router := setupRouter(&mockTaskListProvider{
			createListFunc: func(userID uint, in services.VehicleInput) (*models.TaskList, error) { return nil, svcErr },
			updateListFunc: func(listID, userID uint, in services.VehicleInput) (*models.TaskList, error) { return nil, svcErr },
		})

		for _, rec := range []*httptest.ResponseRecorder{
			jsonRequest(router, http.MethodPost, "/api/lists", map[string]string{"title": "Gol"}),
			jsonRequest(router, http.MethodPut, "/api/lists/3", map[string]string{"title": "Gol"}),
		} {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Equal(t, svcErr.Error(), decodeObject(t, rec)["error"])
		}
	}
}
