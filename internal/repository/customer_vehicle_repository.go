package repository

import (
	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

func (r *CustomerRepository) CreateVehicle(v *models.CustomerVehicle) error {
	return r.db.Create(v).Error
}

func (r *CustomerRepository) SaveVehicle(v *models.CustomerVehicle) error {
	return r.db.Save(v).Error
}

func (r *CustomerRepository) FindVehicle(workspaceID, id uint) (*models.CustomerVehicle, error) {
	var v models.CustomerVehicle
	if err := r.db.Where("workspace_id = ?", workspaceID).First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// FindVehicleByPlate recebe a placa já normalizada; nil, nil quando não existe na oficina.
func (r *CustomerRepository) FindVehicleByPlate(workspaceID uint, plate string) (*models.CustomerVehicle, error) {
	return firstOrNil[models.CustomerVehicle](r.db.Where("workspace_id = ? AND plate = ?", workspaceID, plate))
}

func (r *CustomerRepository) FindVehicleBySource(workspaceID uint, source, sourceID string) (*models.CustomerVehicle, error) {
	return firstOrNil[models.CustomerVehicle](r.db.Where("workspace_id = ? AND source = ? AND source_id = ?", workspaceID, source, sourceID))
}

// ServiceRecordsByVehicle: OS importadas do veículo, mais recentes primeiro.
func (r *CustomerRepository) ServiceRecordsByVehicle(workspaceID, vehicleID uint) ([]models.ServiceRecord, error) {
	records := []models.ServiceRecord{}
	err := r.db.Where("workspace_id = ? AND customer_vehicle_id = ?", workspaceID, vehicleID).
		Order("opened_at desc").Order("id desc").
		Find(&records).Error
	return records, err
}

// TaskListsByCustomerVehicle: nossos atendimentos vinculados ao veículo do cliente, com os serviços.
func (r *CustomerRepository) TaskListsByCustomerVehicle(workspaceID, vehicleID uint) ([]models.TaskList, error) {
	lists := []models.TaskList{}
	err := r.db.
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Where("workspace_id = ? AND customer_vehicle_id = ?", workspaceID, vehicleID).
		Order("created_at desc").Order("id desc").
		Find(&lists).Error
	return lists, err
}
