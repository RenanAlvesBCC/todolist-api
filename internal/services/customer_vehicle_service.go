package services

import (
	"errors"
	"time"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

var (
	ErrCustomerVehicleNotFound = errors.New("veículo do cliente não encontrado")
	ErrPlateTaken              = errors.New("placa já cadastrada em outro veículo")
	ErrInvalidCustomer         = errors.New("cliente inválido")
	ErrInvalidCustomerVehicle  = errors.New("veículo do cliente inválido")
)

// CustomerVehicleInput: nil = chave ausente (no PUT mantém o valor atual).
type CustomerVehicleInput struct {
	// Só no PUT: transfere o veículo para outro cliente da oficina.
	CustomerID   *uint
	Plate        *string
	Manufacturer *string
	Model        *string
	Transmission *string
	YearFab      *int
	YearModel    *int
	Fuel         *string
	Engine       *string
	Doors        *int
	Color        *string
	Chassis      *string
	Renavam      *string
	LastMileage  *int
}

type VehicleLookup struct {
	Vehicle  *models.CustomerVehicle `json:"vehicle"`
	Customer *models.Customer        `json:"customer"`
}

func (s *CustomerService) findVehicle(wsID, id uint) (*models.CustomerVehicle, error) {
	v, err := s.store.FindVehicle(wsID, id)
	if err != nil {
		return nil, ErrCustomerVehicleNotFound
	}
	return v, nil
}

func (s *CustomerService) ListVehicles(userID, customerID uint) ([]models.CustomerVehicle, error) {
	wsID, err := s.workspace(userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.findCustomer(wsID, customerID); err != nil {
		return nil, err
	}
	return s.store.VehiclesByCustomers(wsID, []uint{customerID})
}

func (s *CustomerService) CreateVehicle(userID, customerID uint, in CustomerVehicleInput) (*models.CustomerVehicle, error) {
	wsID, err := s.workspace(userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.findCustomer(wsID, customerID); err != nil {
		return nil, err
	}
	if in.Plate == nil {
		return nil, ErrPlateRequired
	}
	in.CustomerID = nil
	v := &models.CustomerVehicle{WorkspaceID: wsID, CustomerID: &customerID, Source: models.SourceLocal}
	if err := applyVehicleInput(v, in, s.now()); err != nil {
		return nil, err
	}
	if err := s.ensurePlateAvailable(v); err != nil {
		return nil, err
	}
	if err := s.store.CreateVehicle(v); err != nil {
		return nil, err
	}
	return v, nil
}

// UpdateVehicle edita veículo local ou importado (a sincronização pode sobrescrever o importado).
func (s *CustomerService) UpdateVehicle(userID, vehicleID uint, in CustomerVehicleInput) (*models.CustomerVehicle, error) {
	wsID, err := s.workspace(userID)
	if err != nil {
		return nil, err
	}
	v, err := s.findVehicle(wsID, vehicleID)
	if err != nil {
		return nil, err
	}
	if in.CustomerID != nil {
		if _, err := s.store.FindCustomer(wsID, *in.CustomerID); err != nil {
			return nil, ErrInvalidCustomer
		}
	}
	if err := applyVehicleInput(v, in, s.now()); err != nil {
		return nil, err
	}
	if err := s.ensurePlateAvailable(v); err != nil {
		return nil, err
	}
	if err := s.store.SaveVehicle(v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *CustomerService) ensurePlateAvailable(v *models.CustomerVehicle) error {
	if v.Plate == "" {
		return nil
	}
	holder, err := s.store.FindVehicleByPlate(v.WorkspaceID, v.Plate)
	if err != nil {
		return err
	}
	if holder != nil && holder.ID != v.ID {
		return &DuplicateError{Err: ErrPlateTaken, ExistingID: holder.ID}
	}
	return nil
}

// LookupByPlate reconhece um carro conhecido pela placa (com ou sem máscara).
func (s *CustomerService) LookupByPlate(userID uint, plate string) (*VehicleLookup, error) {
	wsID, err := s.workspace(userID)
	if err != nil {
		return nil, err
	}
	normalized := normalizeCustomerPlate(plate)
	if normalized == "" {
		return nil, ErrPlateRequired
	}
	v, err := s.store.FindVehicleByPlate(wsID, normalized)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrCustomerVehicleNotFound
	}
	out := &VehicleLookup{Vehicle: v}
	if v.CustomerID != nil {
		if c, err := s.store.FindCustomer(wsID, *v.CustomerID); err == nil {
			out.Customer = c
		}
	}
	return out, nil
}

// applyVehicleInput valida tudo antes de alterar o veículo.
func applyVehicleInput(v *models.CustomerVehicle, in CustomerVehicleInput, now time.Time) error {
	next := *v
	if in.Plate != nil {
		next.Plate = normalizeCustomerPlate(*in.Plate)
		if next.Plate == "" {
			return ErrPlateRequired
		}
	}
	if !validYear(in.YearFab, now) || !validYear(in.YearModel, now) {
		return ErrInvalidYear
	}
	if !validDoors(in.Doors) {
		return ErrInvalidDoors
	}
	if !validMileage(in.LastMileage) {
		return ErrInvalidMileage
	}
	for _, f := range []struct{ value, target *string }{
		{in.Manufacturer, &next.Manufacturer}, {in.Model, &next.Model}, {in.Transmission, &next.Transmission},
		{in.Fuel, &next.Fuel}, {in.Engine, &next.Engine}, {in.Color, &next.Color},
		{in.Chassis, &next.Chassis}, {in.Renavam, &next.Renavam},
	} {
		if f.value != nil {
			*f.target = trimmed(*f.value)
		}
	}
	for _, f := range []struct{ value, target **int }{
		{&in.YearFab, &next.YearFab}, {&in.YearModel, &next.YearModel},
		{&in.Doors, &next.Doors}, {&in.LastMileage, &next.LastMileage},
	} {
		if *f.value != nil {
			n := **f.value
			*f.target = &n
		}
	}
	if in.CustomerID != nil {
		id := *in.CustomerID
		next.CustomerID = &id
	}
	*v = next
	return nil
}
