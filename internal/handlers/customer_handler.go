package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/RenanAlvesBCC/oficina-api/internal/services"
	"github.com/RenanAlvesBCC/oficina-api/internal/utils"
)

type CustomerProvider interface {
	ListCustomers(userID uint, search string, page, limit int) (*services.PaginatedCustomers, error)
	GetCustomer(userID, id uint) (*services.CustomerDetail, error)
	CreateCustomer(userID uint, in services.CustomerInput) (*services.CustomerDetail, error)
	UpdateCustomer(userID, id uint, in services.CustomerInput) (*services.CustomerDetail, error)
}

type CustomerHandler struct {
	svc CustomerProvider
}

func NewCustomerHandler(svc CustomerProvider) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

// Ponteiros: chave ausente ou null = nil (no PUT mantém o valor atual).
type customerInput struct {
	Name           *string `json:"name"`
	PersonType     *string `json:"person_type"`
	CPF            *string `json:"cpf"`
	CNPJ           *string `json:"cnpj"`
	RG             *string `json:"rg"`
	Phone          *string `json:"phone"`
	Mobile         *string `json:"mobile"`
	Email          *string `json:"email"`
	EmailSecondary *string `json:"email_secondary"`
	CEP            *string `json:"cep"`
	Street         *string `json:"street"`
	Number         *string `json:"number"`
	Complement     *string `json:"complement"`
	District       *string `json:"district"`
	City           *string `json:"city"`
	UF             *string `json:"uf"`
	BirthDate      *string `json:"birth_date"`
	Notes          *string `json:"notes"`
}

func (in customerInput) toService() services.CustomerInput {
	return services.CustomerInput(in)
}

var customerValidationErrors = []error{
	services.ErrCustomerNameRequired,
	services.ErrInvalidCPF,
	services.ErrInvalidCNPJ,
	services.ErrInvalidCEP,
	services.ErrInvalidPhone,
	services.ErrInvalidMobile,
	services.ErrInvalidEmail,
	services.ErrInvalidBirthDate,
	services.ErrInvalidPersonType,
	services.ErrInvalidUF,
}

// respondCustomerError: 403 mecânico, 404 sem oficina/inexistente/outra oficina,
// 409 duplicado (com existing_id), 400 validação.
func respondCustomerError(c *gin.Context, err error) {
	var dup *services.DuplicateError
	switch {
	case errors.As(err, &dup):
		c.JSON(http.StatusConflict, gin.H{"error": dup.Error(), "existing_id": dup.ExistingID})
		return
	case errors.Is(err, services.ErrCustomerForbidden):
		utils.RespondError(c, http.StatusForbidden, err.Error())
		return
	case errors.Is(err, services.ErrNoWorkspace),
		errors.Is(err, services.ErrCustomerNotFound):
		utils.RespondError(c, http.StatusNotFound, err.Error())
		return
	}
	for _, target := range customerValidationErrors {
		if errors.Is(err, target) {
			utils.RespondError(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	utils.RespondError(c, http.StatusInternalServerError, "erro ao processar clientes")
}

func parseIDParam(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "id inválido")
		return 0, false
	}
	return uint(id), true
}

func (h *CustomerHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	result, err := h.svc.ListCustomers(getUserID(c), c.Query("search"), page, limit)
	if err != nil {
		respondCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *CustomerHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	result, err := h.svc.GetCustomer(getUserID(c), id)
	if err != nil {
		respondCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *CustomerHandler) Create(c *gin.Context) {
	var input customerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}
	result, err := h.svc.CreateCustomer(getUserID(c), input.toService())
	if err != nil {
		respondCustomerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *CustomerHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var input customerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "dados inválidos: "+err.Error())
		return
	}
	result, err := h.svc.UpdateCustomer(getUserID(c), id, input.toService())
	if err != nil {
		respondCustomerError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
