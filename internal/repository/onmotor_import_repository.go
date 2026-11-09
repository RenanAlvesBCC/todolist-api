package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

// importWriteBatch: linhas por INSERT (cabe nos limites de parâmetros do SQLite e do Postgres).
const importWriteBatch = 200

// Transaction roda fn com um repositório preso à mesma transação (commit só se fn não falhar).
func (r *CustomerRepository) Transaction(fn func(tx *CustomerRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(&CustomerRepository{db: tx})
	})
}

// CustomerImportLookup: chaves de um lote da importação para pré-carregar os clientes de uma vez.
type CustomerImportLookup struct {
	SourceIDs []string
	CPFs      []string
	CNPJs     []string
	// Celulares para casar clientes do OnMotor sem CPF/CNPJ por nome + celular.
	Mobiles []string
}

// CustomersForImport traz, numa consulta só, os clientes da oficina que o lote pode tocar.
func (r *CustomerRepository) CustomersForImport(workspaceID uint, l CustomerImportLookup) ([]models.Customer, error) {
	rows := []models.Customer{}
	cond := anyOf(r.db, []orCondition{
		{"source = ? AND source_id IN ?", l.SourceIDs, true},
		{"cpf IN ?", l.CPFs, false},
		{"cnpj IN ?", l.CNPJs, false},
		{"source = ? AND cpf = '' AND cnpj = '' AND mobile IN ?", l.Mobiles, true},
	})
	if cond == nil {
		return rows, nil
	}
	err := r.db.Where("workspace_id = ?", workspaceID).Where(cond).Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *CustomerRepository) VehiclesForImport(workspaceID uint, sourceIDs, plates []string) ([]models.CustomerVehicle, error) {
	rows := []models.CustomerVehicle{}
	cond := anyOf(r.db, []orCondition{
		{"source = ? AND source_id IN ?", sourceIDs, true},
		{"plate IN ?", plates, false},
	})
	if cond == nil {
		return rows, nil
	}
	err := r.db.Where("workspace_id = ?", workspaceID).Where(cond).Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *CustomerRepository) ServiceRecordsForImport(workspaceID uint, sourceIDs []string) ([]models.ServiceRecord, error) {
	rows := []models.ServiceRecord{}
	if len(sourceIDs) == 0 {
		return rows, nil
	}
	err := r.db.Where("workspace_id = ? AND source = ? AND source_id IN ?", workspaceID, models.SourceOnMotor, sourceIDs).
		Find(&rows).Error
	return rows, err
}

type orCondition struct {
	sql      string
	values   []string
	byOrigin bool // o primeiro "?" é a origem (onmotor)
}

// anyOf junta com OR só as condições com valores (lista vazia não pode virar "IN ()" nem casar vazios).
func anyOf(db *gorm.DB, conds []orCondition) *gorm.DB {
	var out *gorm.DB
	for _, c := range conds {
		if len(c.values) == 0 {
			continue
		}
		args := []any{c.values}
		if c.byOrigin {
			args = []any{models.SourceOnMotor, c.values}
		}
		if out == nil {
			out = db.Where(c.sql, args...)
		} else {
			out = out.Or(c.sql, args...)
		}
	}
	return out
}

func createRows[T any](db *gorm.DB, rows []*T) error {
	if len(rows) == 0 {
		return nil
	}
	return db.CreateInBatches(rows, importWriteBatch).Error
}

// upsertRows regrava linhas existentes (por id) em lote; created_at é preservado.
func upsertRows[T any](db *gorm.DB, rows []*T) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, UpdateAll: true}).
		CreateInBatches(rows, importWriteBatch).Error
}

func updateByIDs(db *gorm.DB, model any, ids []uint, columns map[string]any) error {
	if len(ids) == 0 {
		return nil
	}
	return db.Model(model).Where("id IN ?", ids).UpdateColumns(columns).Error
}

func (r *CustomerRepository) CreateCustomers(rows []*models.Customer) error {
	return createRows(r.db, rows)
}
func (r *CustomerRepository) UpsertCustomers(rows []*models.Customer) error {
	return upsertRows(r.db, rows)
}

// ReleaseCustomerKeys solta CPF e source_id antes de regravar, para que trocas dentro do
// mesmo lote não esbarrem nos índices únicos.
func (r *CustomerRepository) ReleaseCustomerKeys(ids []uint) error {
	return updateByIDs(r.db, &models.Customer{}, ids, map[string]any{"cpf": "", "source_id": nil})
}

func (r *CustomerRepository) TouchCustomers(ids []uint, at time.Time) error {
	return updateByIDs(r.db, &models.Customer{}, ids, map[string]any{"synced_at": at})
}

func (r *CustomerRepository) CreateVehicles(rows []*models.CustomerVehicle) error {
	return createRows(r.db, rows)
}
func (r *CustomerRepository) UpsertVehicles(rows []*models.CustomerVehicle) error {
	return upsertRows(r.db, rows)
}

// ReleaseVehicleKeys solta placa e source_id antes de regravar (mesmo motivo dos clientes).
func (r *CustomerRepository) ReleaseVehicleKeys(ids []uint) error {
	return updateByIDs(r.db, &models.CustomerVehicle{}, ids, map[string]any{"plate": "", "source_id": nil})
}

func (r *CustomerRepository) TouchVehicles(ids []uint, at time.Time) error {
	return updateByIDs(r.db, &models.CustomerVehicle{}, ids, map[string]any{"synced_at": at})
}

func (r *CustomerRepository) CreateServiceRecords(rows []*models.ServiceRecord) error {
	return createRows(r.db, rows)
}
func (r *CustomerRepository) UpsertServiceRecords(rows []*models.ServiceRecord) error {
	return upsertRows(r.db, rows)
}

func (r *CustomerRepository) TouchServiceRecords(ids []uint, at time.Time) error {
	return updateByIDs(r.db, &models.ServiceRecord{}, ids, map[string]any{"synced_at": at})
}
