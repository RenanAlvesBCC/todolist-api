package services

import (
	"errors"
	"math"
	"time"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/repository"
)

var (
	ErrCustomerForbidden = errors.New("apenas dono ou gerente podem acessar clientes")
	ErrCustomerNotFound  = errors.New("cliente não encontrado")
	ErrCustomerCPFTaken  = errors.New("CPF já cadastrado em outro cliente")
)

// CustomerStore é o que os services de clientes e da importação precisam do banco.
// Os Find*By* devolvem nil, nil quando não encontram.
type CustomerStore interface {
	CreateCustomer(c *models.Customer) error
	SaveCustomer(c *models.Customer) error
	TouchCustomerSynced(id uint, at time.Time) error
	FindCustomer(workspaceID, id uint) (*models.Customer, error)
	FindCustomerByCPF(workspaceID uint, cpf string) (*models.Customer, error)
	FindCustomerBySource(workspaceID uint, source, sourceID string) (*models.Customer, error)
	SearchCustomers(workspaceID uint, filter repository.CustomerFilter) ([]models.Customer, int64, error)
	LastServiceAt(workspaceID uint, customerIDs []uint) (map[uint]time.Time, error)
	VehiclesByCustomers(workspaceID uint, customerIDs []uint) ([]models.CustomerVehicle, error)
	CreateVehicle(v *models.CustomerVehicle) error
	SaveVehicle(v *models.CustomerVehicle) error
	FindVehicle(workspaceID, id uint) (*models.CustomerVehicle, error)
	FindVehicleByPlate(workspaceID uint, plate string) (*models.CustomerVehicle, error)
	ServiceRecordsByVehicle(workspaceID, vehicleID uint) ([]models.ServiceRecord, error)
	TaskListsByCustomerVehicle(workspaceID, vehicleID uint) ([]models.TaskList, error)
}

type CustomerService struct {
	store   CustomerStore
	wsStore WorkspaceRoleStore
	now     func() time.Time
}

func NewCustomerService(store CustomerStore, wsStore WorkspaceRoleStore) *CustomerService {
	return &CustomerService{store: store, wsStore: wsStore, now: time.Now}
}

// CustomerInput: nil = chave ausente (no PUT mantém; no POST fica vazio).
type CustomerInput struct {
	Name           *string
	PersonType     *string
	CPF            *string
	CNPJ           *string
	RG             *string
	Phone          *string
	Mobile         *string
	Email          *string
	EmailSecondary *string
	CEP            *string
	Street         *string
	Number         *string
	Complement     *string
	District       *string
	City           *string
	UF             *string
	BirthDate      *string
	Notes          *string
}

type CustomerVehicleSummary struct {
	ID           uint   `json:"id"`
	Plate        string `json:"plate"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	YearModel    *int   `json:"year_model"`
	LastMileage  *int   `json:"last_mileage"`
}

type CustomerListItem struct {
	models.Customer
	Vehicles      []CustomerVehicleSummary `json:"vehicles"`
	LastServiceAt *time.Time               `json:"last_service_at"`
}

type CustomerDetail struct {
	models.Customer
	Vehicles      []models.CustomerVehicle `json:"vehicles"`
	LastServiceAt *time.Time               `json:"last_service_at"`
}

type PaginatedCustomers struct {
	Customers  []CustomerListItem `json:"customers"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	Total      int64              `json:"total"`
	TotalPages int                `json:"total_pages"`
}

func (s *CustomerService) workspace(userID uint) (uint, error) {
	return managerWorkspace(s.wsStore, userID, ErrCustomerForbidden)
}

func (s *CustomerService) ListCustomers(userID uint, search string, page, limit int) (*PaginatedCustomers, error) {
	wsID, err := s.workspace(userID)
	if err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	customers, total, err := s.store.SearchCustomers(wsID, repository.CustomerFilter{Search: search, Page: page, Limit: limit})
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(customers))
	for i, c := range customers {
		ids[i] = c.ID
	}
	vehicles, lastService, err := s.related(wsID, ids)
	if err != nil {
		return nil, err
	}

	items := make([]CustomerListItem, len(customers))
	for i, c := range customers {
		summaries := []CustomerVehicleSummary{}
		for _, v := range vehicles[c.ID] {
			summaries = append(summaries, CustomerVehicleSummary{
				ID: v.ID, Plate: v.Plate, Manufacturer: v.Manufacturer, Model: v.Model,
				YearModel: v.YearModel, LastMileage: v.LastMileage,
			})
		}
		items[i] = CustomerListItem{Customer: c, Vehicles: summaries, LastServiceAt: lastService(&c)}
	}

	return &PaginatedCustomers{
		Customers:  items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: int(math.Ceil(float64(total) / float64(limit))),
	}, nil
}

// related carrega veículos (por cliente) e a data do último atendimento: o mais recente entre
// OS importadas, nossos atendimentos vinculados e a última visita registrada no OnMotor.
func (s *CustomerService) related(wsID uint, ids []uint) (map[uint][]models.CustomerVehicle, func(*models.Customer) *time.Time, error) {
	vehicles, err := s.store.VehiclesByCustomers(wsID, ids)
	if err != nil {
		return nil, nil, err
	}
	last, err := s.store.LastServiceAt(wsID, ids)
	if err != nil {
		return nil, nil, err
	}
	byCustomer := make(map[uint][]models.CustomerVehicle, len(ids))
	for _, v := range vehicles {
		byCustomer[*v.CustomerID] = append(byCustomer[*v.CustomerID], v)
	}
	lastService := func(c *models.Customer) *time.Time {
		at, ok := last[c.ID]
		if c.LastVisitAt != nil && (!ok || c.LastVisitAt.After(at)) {
			at, ok = *c.LastVisitAt, true
		}
		if !ok {
			return nil
		}
		at = at.UTC()
		return &at
	}
	return byCustomer, lastService, nil
}

func (s *CustomerService) detail(wsID uint, c *models.Customer) (*CustomerDetail, error) {
	vehicles, lastService, err := s.related(wsID, []uint{c.ID})
	if err != nil {
		return nil, err
	}
	list := vehicles[c.ID]
	if list == nil {
		list = []models.CustomerVehicle{}
	}
	return &CustomerDetail{Customer: *c, Vehicles: list, LastServiceAt: lastService(c)}, nil
}

func (s *CustomerService) findCustomer(wsID, id uint) (*models.Customer, error) {
	c, err := s.store.FindCustomer(wsID, id)
	if err != nil {
		return nil, ErrCustomerNotFound
	}
	return c, nil
}

func (s *CustomerService) GetCustomer(userID, id uint) (*CustomerDetail, error) {
	wsID, err := s.workspace(userID)
	if err != nil {
		return nil, err
	}
	c, err := s.findCustomer(wsID, id)
	if err != nil {
		return nil, err
	}
	return s.detail(wsID, c)
}

func (s *CustomerService) CreateCustomer(userID uint, in CustomerInput) (*CustomerDetail, error) {
	wsID, err := s.workspace(userID)
	if err != nil {
		return nil, err
	}
	if in.Name == nil {
		return nil, ErrCustomerNameRequired
	}
	c := &models.Customer{WorkspaceID: wsID, Source: models.SourceLocal}
	if err := applyCustomerInput(c, in, s.now()); err != nil {
		return nil, err
	}
	if err := s.ensureCPFAvailable(c); err != nil {
		return nil, err
	}
	if err := s.store.CreateCustomer(c); err != nil {
		return nil, err
	}
	return &CustomerDetail{Customer: *c, Vehicles: []models.CustomerVehicle{}}, nil
}

// UpdateCustomer edita cliente local ou importado. Campos vindos do OnMotor podem
// ser sobrescritos na próxima sincronização.
func (s *CustomerService) UpdateCustomer(userID, id uint, in CustomerInput) (*CustomerDetail, error) {
	wsID, err := s.workspace(userID)
	if err != nil {
		return nil, err
	}
	c, err := s.findCustomer(wsID, id)
	if err != nil {
		return nil, err
	}
	if err := applyCustomerInput(c, in, s.now()); err != nil {
		return nil, err
	}
	if err := s.ensureCPFAvailable(c); err != nil {
		return nil, err
	}
	if err := s.store.SaveCustomer(c); err != nil {
		return nil, err
	}
	return s.detail(wsID, c)
}

func (s *CustomerService) ensureCPFAvailable(c *models.Customer) error {
	if c.CPF == "" {
		return nil
	}
	holder, err := s.store.FindCustomerByCPF(c.WorkspaceID, c.CPF)
	if err != nil {
		return err
	}
	if holder != nil && holder.ID != c.ID {
		return &DuplicateError{Err: ErrCustomerCPFTaken, ExistingID: holder.ID}
	}
	return nil
}

// applyCustomerInput valida tudo antes de alterar o cliente.
func applyCustomerInput(c *models.Customer, in CustomerInput, now time.Time) error {
	next := *c
	if in.Name != nil {
		next.Name = trimmed(*in.Name)
		if next.Name == "" {
			return ErrCustomerNameRequired
		}
	}
	normalized := []struct {
		value     *string
		normalize func(string) (string, error)
		target    *string
	}{
		{in.PersonType, normalizePersonType, &next.PersonType},
		{in.CPF, normalizeCPF, &next.CPF},
		{in.CNPJ, normalizeCNPJ, &next.CNPJ},
		{in.Phone, normalizePhone, &next.Phone},
		{in.Mobile, normalizeMobile, &next.Mobile},
		{in.Email, normalizeEmail, &next.Email},
		{in.EmailSecondary, normalizeEmail, &next.EmailSecondary},
		{in.CEP, normalizeCEP, &next.CEP},
		{in.UF, normalizeUF, &next.UF},
	}
	for _, f := range normalized {
		if f.value == nil {
			continue
		}
		v, err := f.normalize(*f.value)
		if err != nil {
			return err
		}
		*f.target = v
	}
	for _, f := range []struct{ value, target *string }{
		{in.RG, &next.RG}, {in.Street, &next.Street}, {in.Number, &next.Number}, {in.Complement, &next.Complement},
		{in.District, &next.District}, {in.City, &next.City}, {in.Notes, &next.Notes},
	} {
		if f.value != nil {
			*f.target = trimmed(*f.value)
		}
	}
	if in.BirthDate != nil {
		d, err := parseBirthDate(*in.BirthDate, now)
		if err != nil {
			return err
		}
		next.BirthDate = d
	}
	*c = next
	return nil
}
