package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
)

type mockCustomerVehicleProvider struct {
	err        error
	gotPlate   string
	gotInput   services.CustomerVehicleInput
	gotID      uint
	gotOwnerID uint
}

func (m *mockCustomerVehicleProvider) ListVehicles(userID, customerID uint) ([]models.CustomerVehicle, error) {
	m.gotOwnerID = customerID
	if m.err != nil {
		return nil, m.err
	}
	return []models.CustomerVehicle{{ID: 30, Plate: "ABC1D23"}}, nil
}
func (m *mockCustomerVehicleProvider) CreateVehicle(userID, customerID uint, in services.CustomerVehicleInput) (*models.CustomerVehicle, error) {
	m.gotOwnerID, m.gotInput = customerID, in
	if m.err != nil {
		return nil, m.err
	}
	return &models.CustomerVehicle{ID: 31, CustomerID: &customerID, Plate: "ABC1D23"}, nil
}
func (m *mockCustomerVehicleProvider) UpdateVehicle(userID, vehicleID uint, in services.CustomerVehicleInput) (*models.CustomerVehicle, error) {
	m.gotID, m.gotInput = vehicleID, in
	if m.err != nil {
		return nil, m.err
	}
	return &models.CustomerVehicle{ID: vehicleID}, nil
}
func (m *mockCustomerVehicleProvider) LookupByPlate(userID uint, plate string) (*services.VehicleLookup, error) {
	m.gotPlate = plate
	if m.err != nil {
		return nil, m.err
	}
	return &services.VehicleLookup{Vehicle: &models.CustomerVehicle{ID: 30, Plate: "ABC1D23"}}, nil
}
func (m *mockCustomerVehicleProvider) VehicleHistory(userID, vehicleID uint) (*services.VehicleHistory, error) {
	m.gotID = vehicleID
	if m.err != nil {
		return nil, m.err
	}
	listID := uint(7)
	return &services.VehicleHistory{Records: []services.HistoryRecord{{Source: "rmaxx", ID: 7, ListID: &listID, Items: "null"}}}, nil
}

func setupCustomerVehicleRouter(p *mockCustomerVehicleProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewCustomerVehicleHandler(p)
	g := router.Group("/api")
	g.Use(withUserContext(1))
	{
		g.GET("/customers/:id/vehicles", h.List)
		g.POST("/customers/:id/vehicles", h.Create)
		g.GET("/customer-vehicles/lookup", h.Lookup)
		g.PUT("/customer-vehicles/:id", h.Update)
		g.GET("/customer-vehicles/:id/history", h.History)
	}
	return router
}

func TestCustomerVehicleHandler_HappyPaths(t *testing.T) {
	p := &mockCustomerVehicleProvider{}
	router := setupCustomerVehicleRouter(p)

	rec := jsonRequest(router, http.MethodGet, "/api/customers/12/vehicles", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uint(12), p.gotOwnerID)
	assert.Contains(t, rec.Body.String(), `"plate":"ABC1D23"`)

	rec = rawRequest(router, http.MethodPost, "/api/customers/12/vehicles", `{"plate":"abc-1d23","year_model":2015,"model":"Gol"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "abc-1d23", *p.gotInput.Plate)
	assert.Equal(t, 2015, *p.gotInput.YearModel)
	assert.Nil(t, p.gotInput.Color)

	rec = rawRequest(router, http.MethodPut, "/api/customer-vehicles/30", `{"customer_id":5,"color":"Preto","last_mileage":null}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uint(30), p.gotID)
	assert.Equal(t, uint(5), *p.gotInput.CustomerID)
	assert.Nil(t, p.gotInput.LastMileage)

	rec = jsonRequest(router, http.MethodGet, "/api/customer-vehicles/lookup?plate=abc-1d23", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "abc-1d23", p.gotPlate)
	body := decodeObject(t, rec)
	assert.Contains(t, body, "customer")
	assert.Equal(t, "ABC1D23", body["vehicle"].(map[string]any)["plate"])

	rec = jsonRequest(router, http.MethodGet, "/api/customer-vehicles/30/history", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	records := decodeObject(t, rec)["records"].([]any)
	require.Len(t, records, 1)
	first := records[0].(map[string]any)
	assert.Equal(t, "rmaxx", first["source"])
	assert.Equal(t, float64(7), first["list_id"])
	assert.Nil(t, first["items"])
}

func TestCustomerVehicleHandler_Errors(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{services.ErrCustomerForbidden, http.StatusForbidden},
		{services.ErrCustomerVehicleNotFound, http.StatusNotFound},
		{services.ErrCustomerNotFound, http.StatusNotFound},
		{services.ErrPlateRequired, http.StatusBadRequest},
		{services.ErrInvalidCustomer, http.StatusBadRequest},
		{&services.DuplicateError{Err: services.ErrPlateTaken, ExistingID: 9}, http.StatusConflict},
	}
	for _, tc := range cases {
		router := setupCustomerVehicleRouter(&mockCustomerVehicleProvider{err: tc.err})
		for _, rec := range []*httptest.ResponseRecorder{
			jsonRequest(router, http.MethodGet, "/api/customers/1/vehicles", nil),
			rawRequest(router, http.MethodPost, "/api/customers/1/vehicles", `{"plate":"ABC1D23"}`),
			rawRequest(router, http.MethodPut, "/api/customer-vehicles/1", `{}`),
			jsonRequest(router, http.MethodGet, "/api/customer-vehicles/lookup?plate=x", nil),
			jsonRequest(router, http.MethodGet, "/api/customer-vehicles/1/history", nil),
		} {
			assert.Equal(t, tc.code, rec.Code, tc.err.Error())
		}
	}

	router := setupCustomerVehicleRouter(&mockCustomerVehicleProvider{err: &services.DuplicateError{Err: services.ErrPlateTaken, ExistingID: 9}})
	rec := rawRequest(router, http.MethodPost, "/api/customers/1/vehicles", `{"plate":"ABC1D23"}`)
	body := decodeObject(t, rec)
	assert.Equal(t, "placa já cadastrada em outro veículo", body["error"])
	assert.Equal(t, float64(9), body["existing_id"])
}

func TestCustomerVehicleHandler_BadRequests(t *testing.T) {
	router := setupCustomerVehicleRouter(&mockCustomerVehicleProvider{})
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/customers/x/vehicles", ""},
		{http.MethodPost, "/api/customers/x/vehicles", `{}`},
		{http.MethodPost, "/api/customers/1/vehicles", `{"year_model":"2015"}`},
		{http.MethodPut, "/api/customer-vehicles/x", `{}`},
		{http.MethodPut, "/api/customer-vehicles/1", `{"plate":1}`},
		{http.MethodGet, "/api/customer-vehicles/x/history", ""},
	} {
		rec := rawRequest(router, tc.method, tc.path, tc.body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, tc.method+" "+tc.path)
	}
}
