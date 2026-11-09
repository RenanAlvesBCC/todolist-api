package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
	"github.com/RenanAlvesBCC/oficina-api/internal/utils"
)

type CustomerVehicleProvider interface {
	ListVehicles(userID, customerID uint) ([]models.CustomerVehicle, error)
	CreateVehicle(userID, customerID uint, in services.CustomerVehicleInput) (*models.CustomerVehicle, error)
	UpdateVehicle(userID, vehicleID uint, in services.CustomerVehicleInput) (*models.CustomerVehicle, error)
	LookupByPlate(userID uint, plate string) (*services.VehicleLookup, error)
	VehicleHistory(userID, vehicleID uint) (*services.VehicleHistory, error)
}

type CustomerVehicleHandler struct {
	svc CustomerVehicleProvider
}

func NewCustomerVehicleHandler(svc CustomerVehicleProvider) *CustomerVehicleHandler {
	return &CustomerVehicleHandler{svc: svc}
}

type customerVehicleInput struct {
	CustomerID   *uint   `json:"customer_id"`
	Plate        *string `json:"plate"`
	Manufacturer *string `json:"manufacturer"`
	Model        *string `json:"model"`
	Transmission *string `json:"transmission"`
	YearFab      *int    `json:"year_fab"`
	YearModel    *int    `json:"year_model"`
	Fuel         *string `json:"fuel"`
	Engine       *string `json:"engine"`
	Doors        *int    `json:"doors"`
	Color        *string `json:"color"`
	Chassis      *string `json:"chassis"`
	Renavam      *string `json:"renavam"`
	LastMileage  *int    `json:"last_mileage"`
}

func bindVehicleInput(c *gin.Context) (services.CustomerVehicleInput, bool) {
	var input customerVehicleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return services.CustomerVehicleInput{}, false
	}
	return services.CustomerVehicleInput(input), true
}

func (h *CustomerVehicleHandler) List(c *gin.Context) {
	customerID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	vehicles, err := h.svc.ListVehicles(getUserID(c), customerID)
	if err != nil {
		respondCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, vehicles)
}

func (h *CustomerVehicleHandler) Create(c *gin.Context) {
	customerID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	in, ok := bindVehicleInput(c)
	if !ok {
		return
	}
	v, err := h.svc.CreateVehicle(getUserID(c), customerID, in)
	if err != nil {
		respondCustomerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *CustomerVehicleHandler) Update(c *gin.Context) {
	vehicleID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	in, ok := bindVehicleInput(c)
	if !ok {
		return
	}
	v, err := h.svc.UpdateVehicle(getUserID(c), vehicleID, in)
	if err != nil {
		respondCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *CustomerVehicleHandler) Lookup(c *gin.Context) {
	result, err := h.svc.LookupByPlate(getUserID(c), c.Query("plate"))
	if err != nil {
		respondCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *CustomerVehicleHandler) History(c *gin.Context) {
	vehicleID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	result, err := h.svc.VehicleHistory(getUserID(c), vehicleID)
	if err != nil {
		respondCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
