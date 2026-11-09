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

type mockCustomerProvider struct {
	listFunc   func(userID uint, search string, page, limit int) (*services.PaginatedCustomers, error)
	getFunc    func(userID, id uint) (*services.CustomerDetail, error)
	createFunc func(userID uint, in services.CustomerInput) (*services.CustomerDetail, error)
	updateFunc func(userID, id uint, in services.CustomerInput) (*services.CustomerDetail, error)
}

func (m *mockCustomerProvider) ListCustomers(userID uint, search string, page, limit int) (*services.PaginatedCustomers, error) {
	return m.listFunc(userID, search, page, limit)
}
func (m *mockCustomerProvider) GetCustomer(userID, id uint) (*services.CustomerDetail, error) {
	return m.getFunc(userID, id)
}
func (m *mockCustomerProvider) CreateCustomer(userID uint, in services.CustomerInput) (*services.CustomerDetail, error) {
	return m.createFunc(userID, in)
}
func (m *mockCustomerProvider) UpdateCustomer(userID, id uint, in services.CustomerInput) (*services.CustomerDetail, error) {
	return m.updateFunc(userID, id, in)
}

func setupCustomerRouter(p *mockCustomerProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewCustomerHandler(p)
	g := router.Group("/api")
	g.Use(withUserContext(1))
	{
		g.GET("/customers", h.List)
		g.POST("/customers", h.Create)
		g.GET("/customers/:id", h.Get)
		g.PUT("/customers/:id", h.Update)
	}
	return router
}

func customerDetail(id uint) *services.CustomerDetail {
	return &services.CustomerDetail{Customer: models.Customer{ID: id, Name: "Maria", CPF: "12345678909"}, Vehicles: []models.CustomerVehicle{}}
}

func TestCustomerHandler_List_ForwardsQueryAndReturnsPage(t *testing.T) {
	var gotSearch string
	var gotPage, gotLimit int
	router := setupCustomerRouter(&mockCustomerProvider{
		listFunc: func(userID uint, search string, page, limit int) (*services.PaginatedCustomers, error) {
			gotSearch, gotPage, gotLimit = search, page, limit
			return &services.PaginatedCustomers{
				Customers: []services.CustomerListItem{{Customer: models.Customer{ID: 3, Name: "Maria"}, Vehicles: []services.CustomerVehicleSummary{{ID: 9, Plate: "ABC1D23"}}}},
				Page:      2, Limit: 10, Total: 11, TotalPages: 2,
			}, nil
		},
	})

	rec := jsonRequest(router, http.MethodGet, "/api/customers?search=abc&page=2&limit=10", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "abc", gotSearch)
	assert.Equal(t, 2, gotPage)
	assert.Equal(t, 10, gotLimit)
	body := decodeObject(t, rec)
	assert.Equal(t, float64(11), body["total"])
	customers := body["customers"].([]any)
	require.Len(t, customers, 1)
	first := customers[0].(map[string]any)
	assert.Equal(t, "Maria", first["name"])
	assert.Contains(t, first, "last_service_at")
	assert.Equal(t, "ABC1D23", first["vehicles"].([]any)[0].(map[string]any)["plate"])
}

func TestCustomerHandler_ErrorMapping(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{services.ErrCustomerForbidden, http.StatusForbidden},
		{services.ErrNoWorkspace, http.StatusNotFound},
		{services.ErrCustomerNotFound, http.StatusNotFound},
		{services.ErrInvalidCPF, http.StatusBadRequest},
		{services.ErrCustomerNameRequired, http.StatusBadRequest},
		{assert.AnError, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		router := setupCustomerRouter(&mockCustomerProvider{
			listFunc: func(uint, string, int, int) (*services.PaginatedCustomers, error) { return nil, tc.err },
			getFunc:  func(uint, uint) (*services.CustomerDetail, error) { return nil, tc.err },
			createFunc: func(uint, services.CustomerInput) (*services.CustomerDetail, error) {
				return nil, tc.err
			},
			updateFunc: func(uint, uint, services.CustomerInput) (*services.CustomerDetail, error) {
				return nil, tc.err
			},
		})
		for _, rec := range []*httptest.ResponseRecorder{
			jsonRequest(router, http.MethodGet, "/api/customers", nil),
			jsonRequest(router, http.MethodGet, "/api/customers/3", nil),
			rawRequest(router, http.MethodPost, "/api/customers", `{"name":"Maria"}`),
			rawRequest(router, http.MethodPut, "/api/customers/3", `{"name":"Maria"}`),
		} {
			assert.Equal(t, tc.code, rec.Code, tc.err.Error())
		}
	}
}

func TestCustomerHandler_DuplicateReturns409WithExistingID(t *testing.T) {
	dup := &services.DuplicateError{Err: services.ErrCustomerCPFTaken, ExistingID: 42}
	router := setupCustomerRouter(&mockCustomerProvider{
		createFunc: func(uint, services.CustomerInput) (*services.CustomerDetail, error) { return nil, dup },
		updateFunc: func(uint, uint, services.CustomerInput) (*services.CustomerDetail, error) { return nil, dup },
	})
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		path := "/api/customers"
		if method == http.MethodPut {
			path += "/1"
		}
		rec := rawRequest(router, method, path, `{"name":"Maria","cpf":"123.456.789-09"}`)
		require.Equal(t, http.StatusConflict, rec.Code)
		body := decodeObject(t, rec)
		assert.Equal(t, "CPF já cadastrado em outro cliente", body["error"])
		assert.Equal(t, float64(42), body["existing_id"])
	}
}

func TestCustomerHandler_CreateAndUpdateForwardInput(t *testing.T) {
	var created, updated services.CustomerInput
	var updatedID uint
	router := setupCustomerRouter(&mockCustomerProvider{
		createFunc: func(userID uint, in services.CustomerInput) (*services.CustomerDetail, error) {
			created = in
			return customerDetail(5), nil
		},
		updateFunc: func(userID, id uint, in services.CustomerInput) (*services.CustomerDetail, error) {
			updated, updatedID = in, id
			return customerDetail(id), nil
		},
		getFunc: func(userID, id uint) (*services.CustomerDetail, error) { return customerDetail(id), nil },
	})

	rec := rawRequest(router, http.MethodPost, "/api/customers", `{"name":"Maria","cpf":"123.456.789-09","birth_date":"1980-05-03"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "Maria", *created.Name)
	assert.Equal(t, "123.456.789-09", *created.CPF)
	assert.Equal(t, "1980-05-03", *created.BirthDate)
	assert.Nil(t, created.Email)
	body := decodeObject(t, rec)
	assert.Equal(t, float64(5), body["id"])
	assert.Equal(t, []any{}, body["vehicles"])

	rec = rawRequest(router, http.MethodPut, "/api/customers/8", `{"city":"Campinas","email":null}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, uint(8), updatedID)
	assert.Nil(t, updated.Name, "ausente mantém")
	assert.Nil(t, updated.Email, "null mantém")
	assert.Equal(t, "Campinas", *updated.City)

	rec = jsonRequest(router, http.MethodGet, "/api/customers/8", nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "12345678909", decodeObject(t, rec)["cpf"])
}

func TestCustomerHandler_BadRequests(t *testing.T) {
	router := setupCustomerRouter(&mockCustomerProvider{})
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/customers/abc", ""},
		{http.MethodPut, "/api/customers/abc", `{}`},
		{http.MethodPost, "/api/customers", `{"name": 12}`},
		{http.MethodPut, "/api/customers/1", `{`},
	} {
		rec := rawRequest(router, tc.method, tc.path, tc.body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, tc.method+" "+tc.path)
	}
}
