package repository

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

// CustomerRepository cuida de clientes, veículos do cliente e histórico de OS,
// sempre filtrando pela oficina (workspace_id).
type CustomerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

type CustomerFilter struct {
	Search string
	Page   int
	Limit  int
}

func (r *CustomerRepository) CreateCustomer(c *models.Customer) error {
	return r.db.Create(c).Error
}

func (r *CustomerRepository) SaveCustomer(c *models.Customer) error {
	return r.db.Save(c).Error
}

func (r *CustomerRepository) FindCustomer(workspaceID, id uint) (*models.Customer, error) {
	var c models.Customer
	if err := r.db.Where("workspace_id = ?", workspaceID).First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// FindCustomerByCPF devolve nil, nil quando não há cliente com o CPF na oficina.
func (r *CustomerRepository) FindCustomerByCPF(workspaceID uint, cpf string) (*models.Customer, error) {
	return firstOrNil[models.Customer](r.db.Where("workspace_id = ? AND cpf = ?", workspaceID, cpf))
}

// FindCustomerBySource devolve nil, nil quando o registro de origem ainda não foi importado.
func (r *CustomerRepository) FindCustomerBySource(workspaceID uint, source, sourceID string) (*models.Customer, error) {
	return firstOrNil[models.Customer](r.db.Where("workspace_id = ? AND source = ? AND source_id = ?", workspaceID, source, sourceID))
}

func firstOrNil[T any](query *gorm.DB) (*T, error) {
	var out T
	err := query.First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchCustomers pagina os clientes da oficina: última visita (OnMotor) mais recente primeiro,
// clientes sem visita no fim; desempate por updated_at e id, decrescentes.
// search (sem diferenciar maiúsculas) casa com nome, e-mails, documentos/telefones
// (só dígitos, quando o termo não tem letras) e placas dos veículos do cliente.
func (r *CustomerRepository) SearchCustomers(workspaceID uint, filter CustomerFilter) ([]models.Customer, int64, error) {
	query := func() *gorm.DB {
		q := r.db.Model(&models.Customer{}).Where("customers.workspace_id = ?", workspaceID)
		if cond, args := customerSearchCondition(filter.Search); cond != "" {
			q = q.Where(cond, args...)
		}
		return q
	}

	var total int64
	if err := query().Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var customers []models.Customer
	if err := query().
		// CASE em vez de NULLS LAST: o Postgres põe NULL primeiro no DESC; o SQLite, por último.
		Order("CASE WHEN customers.last_visit_at IS NULL THEN 1 ELSE 0 END").
		Order("customers.last_visit_at desc").
		Order("customers.updated_at desc").
		Order("customers.id desc").
		Offset((filter.Page - 1) * filter.Limit).Limit(filter.Limit).
		Find(&customers).Error; err != nil {
		return nil, 0, err
	}
	return customers, total, nil
}

func customerSearchCondition(search string) (string, []any) {
	term := strings.TrimSpace(search)
	if term == "" {
		return "", nil
	}
	// LIKE no Postgres diferencia maiúsculas (no SQLite não); comparamos os dois lados em minúsculas.
	like := "%" + strings.ToLower(term) + "%"
	conds := []string{
		"LOWER(customers.name) LIKE ?",
		"LOWER(customers.email) LIKE ?",
		"LOWER(customers.email_secondary) LIKE ?",
	}
	args := []any{like, like, like}

	if digits, hasLetters := searchDigits(term); digits != "" && !hasLetters {
		d := "%" + digits + "%"
		conds = append(conds, "customers.cpf LIKE ?", "customers.cnpj LIKE ?", "customers.phone LIKE ?", "customers.mobile LIKE ?")
		args = append(args, d, d, d, d)
	}
	if plate := plateSearchTerm(term); plate != "" {
		conds = append(conds, "EXISTS (SELECT 1 FROM customer_vehicles cv WHERE cv.customer_id = customers.id"+
			" AND cv.workspace_id = customers.workspace_id AND cv.plate LIKE ?)")
		args = append(args, "%"+plate+"%")
	}
	return "(" + strings.Join(conds, " OR ") + ")", args
}

func searchDigits(term string) (digits string, hasLetters bool) {
	var b strings.Builder
	for _, r := range term {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r != '.' && r != '-' && r != '/' && r != '(' && r != ')' && r != '+' && r != ' ':
			hasLetters = true
		}
	}
	return b.String(), hasLetters
}

func plateSearchTerm(term string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(term) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// LastServiceAt: por cliente, o mais recente entre a abertura das OS importadas
// e a criação dos nossos atendimentos (task_lists) vinculados a ele.
func (r *CustomerRepository) LastServiceAt(workspaceID uint, customerIDs []uint) (map[uint]time.Time, error) {
	out := make(map[uint]time.Time, len(customerIDs))
	if len(customerIDs) == 0 {
		return out, nil
	}
	queries := []*gorm.DB{
		r.db.Model(&models.ServiceRecord{}).
			Select("customer_id, MAX(opened_at)").
			Where("workspace_id = ? AND customer_id IN ? AND opened_at IS NOT NULL", workspaceID, customerIDs).
			Group("customer_id"),
		r.db.Model(&models.TaskList{}).
			Select("customer_id, MAX(created_at)").
			Where("workspace_id = ? AND customer_id IN ?", workspaceID, customerIDs).
			Group("customer_id"),
	}
	for _, q := range queries {
		if err := scanMaxTimes(q, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func scanMaxTimes(query *gorm.DB, out map[uint]time.Time) error {
	rows, err := query.Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id uint
		var at scannedTime
		if err := rows.Scan(&id, &at); err != nil {
			return err
		}
		if at.Valid && at.Time.After(out[id]) {
			out[id] = at.Time
		}
	}
	return rows.Err()
}

// scannedTime lê MAX(timestamp): o Postgres devolve time.Time; o SQLite, texto.
type scannedTime struct {
	Time  time.Time
	Valid bool
}

var sqliteTimeLayouts = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999",
}

func (s *scannedTime) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		return nil
	case time.Time:
		s.Time, s.Valid = v, true
		return nil
	case []byte:
		return s.parse(string(v))
	case string:
		return s.parse(v)
	}
	return fmt.Errorf("tipo inesperado para data/hora: %T", value)
}

func (s *scannedTime) parse(raw string) error {
	for _, layout := range sqliteTimeLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			s.Time, s.Valid = t, true
			return nil
		}
	}
	return fmt.Errorf("data/hora inválida: %q", raw)
}

// VehiclesByCustomers devolve os veículos dos clientes, por placa.
func (r *CustomerRepository) VehiclesByCustomers(workspaceID uint, customerIDs []uint) ([]models.CustomerVehicle, error) {
	vehicles := []models.CustomerVehicle{}
	if len(customerIDs) == 0 {
		return vehicles, nil
	}
	err := r.db.Where("workspace_id = ? AND customer_id IN ?", workspaceID, customerIDs).
		Order("plate asc").Order("id asc").
		Find(&vehicles).Error
	return vehicles, err
}
