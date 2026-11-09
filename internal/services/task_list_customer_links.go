package services

import (
	"errors"
	"strings"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

var ErrVehicleCustomerMismatch = errors.New("veículo não pertence ao cliente")

// CustomerLinkStore resolve os vínculos do atendimento com o cadastro de clientes.
type CustomerLinkStore interface {
	FindCustomer(workspaceID, id uint) (*models.Customer, error)
	FindVehicle(workspaceID, id uint) (*models.CustomerVehicle, error)
}

// WithCustomerLinks liga o atendimento ao cadastro de clientes (customer_id / customer_vehicle_id).
func (s *TaskListService) WithCustomerLinks(links CustomerLinkStore) *TaskListService {
	s.links = links
	return s
}

// applyCustomerLinks valida customer_id / customer_vehicle_id (mesma oficina do atendimento;
// 0 desvincula) e preenche o que estiver vazio: placa e título a partir do veículo do cliente;
// nome, CPF, CEP e WhatsApp a partir do cliente. Sem customer_id, usa o dono do veículo.
func (s *TaskListService) applyCustomerLinks(list *models.TaskList, in VehicleInput) error {
	if in.CustomerID == nil && in.CustomerVehicleID == nil {
		return nil
	}
	vehicle, err := s.linkedVehicle(list, in.CustomerVehicleID)
	if err != nil {
		return err
	}
	customer, err := s.linkedCustomer(list, in.CustomerID)
	if err != nil {
		return err
	}
	if vehicle != nil && customer != nil && (vehicle.CustomerID == nil || *vehicle.CustomerID != customer.ID) {
		return ErrVehicleCustomerMismatch
	}
	if vehicle != nil && in.CustomerID == nil && vehicle.CustomerID != nil {
		if owner, err := s.links.FindCustomer(*list.WorkspaceID, *vehicle.CustomerID); err == nil {
			customer = owner
		}
	}

	if in.CustomerVehicleID != nil {
		list.CustomerVehicleID = nil
		if vehicle != nil {
			list.CustomerVehicleID = &vehicle.ID
		}
	}
	if customer != nil {
		list.CustomerID = &customer.ID
	} else if in.CustomerID != nil {
		list.CustomerID = nil
	}

	if vehicle != nil {
		if list.Plate == "" {
			list.Plate = normalizePlate(vehicle.Plate)
		}
		if list.Title == "" {
			list.Title = strings.TrimSpace(vehicle.Manufacturer + " " + vehicle.Model)
		}
	}
	if customer != nil {
		fillEmpty(&list.Customer, customer.Name)
		fillEmpty(&list.CustomerCPF, customer.CPF)
		fillEmpty(&list.CustomerCEP, customer.CEP)
		fillEmpty(&list.CustomerWhatsApp, customer.Mobile)
	}
	return nil
}

func (s *TaskListService) linkedVehicle(list *models.TaskList, id *uint) (*models.CustomerVehicle, error) {
	if id == nil || *id == 0 {
		return nil, nil
	}
	if s.links == nil || list.WorkspaceID == nil {
		return nil, ErrInvalidCustomerVehicle
	}
	v, err := s.links.FindVehicle(*list.WorkspaceID, *id)
	if err != nil {
		return nil, ErrInvalidCustomerVehicle
	}
	return v, nil
}

func (s *TaskListService) linkedCustomer(list *models.TaskList, id *uint) (*models.Customer, error) {
	if id == nil || *id == 0 {
		return nil, nil
	}
	if s.links == nil || list.WorkspaceID == nil {
		return nil, ErrInvalidCustomer
	}
	c, err := s.links.FindCustomer(*list.WorkspaceID, *id)
	if err != nil {
		return nil, ErrInvalidCustomer
	}
	return c, nil
}

func fillEmpty(target *string, value string) {
	if *target == "" {
		*target = value
	}
}
