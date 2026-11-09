package models

import "time"

// Origem do cadastro: importado do OnMotor (somente leitura lá) ou criado aqui.
const (
	SourceOnMotor = "onmotor"
	SourceLocal   = "local"
)

const (
	PersonTypePF = "pf"
	PersonTypePJ = "pj"
)

// Customer é o cliente da oficina. Documentos, telefones e CEP ficam só com dígitos.
// Só dono/gerente acessam (LGPD); o mecânico nunca recebe estes dados.
type Customer struct {
	ID          uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	WorkspaceID uint    `gorm:"not null;index;uniqueIndex:idx_customers_source;index:idx_customers_cpf,unique,where:cpf <> ''" json:"workspace_id"`
	Source      string  `gorm:"not null;default:'local';uniqueIndex:idx_customers_source" json:"source"`
	SourceID    *string `gorm:"uniqueIndex:idx_customers_source" json:"source_id"`
	Name        string  `gorm:"not null" json:"name"`
	PersonType  string  `gorm:"not null;default:''" json:"person_type"`
	CPF         string  `gorm:"column:cpf;not null;default:'';index:idx_customers_cpf,unique,where:cpf <> ''" json:"cpf"`
	CNPJ        string  `gorm:"column:cnpj;not null;default:''" json:"cnpj"`
	RG          string  `gorm:"column:rg;not null;default:''" json:"rg"`
	Phone       string  `gorm:"not null;default:''" json:"phone"`
	// Celular / WhatsApp.
	Mobile         string     `gorm:"not null;default:''" json:"mobile"`
	Email          string     `gorm:"not null;default:''" json:"email"`
	EmailSecondary string     `gorm:"column:email_secondary;not null;default:''" json:"email_secondary"`
	CEP            string     `gorm:"column:cep;not null;default:''" json:"cep"`
	Street         string     `gorm:"not null;default:''" json:"street"`
	Number         string     `gorm:"not null;default:''" json:"number"`
	Complement     string     `gorm:"not null;default:''" json:"complement"`
	District       string     `gorm:"not null;default:''" json:"district"`
	City           string     `gorm:"not null;default:''" json:"city"`
	UF             string     `gorm:"column:uf;not null;default:''" json:"uf"`
	BirthDate      *Date      `json:"birth_date"`
	Notes          string     `gorm:"not null;default:''" json:"notes"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	SyncedAt       *time.Time `json:"synced_at"`
}

// CustomerVehicle é o carro do cliente (cadastro permanente, diferente do
// atendimento em /api/lists). Placa normalizada: maiúsculas, sem separadores.
type CustomerVehicle struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	WorkspaceID  uint       `gorm:"not null;index;uniqueIndex:idx_customer_vehicles_source;index:idx_customer_vehicles_plate,unique,where:plate <> ''" json:"workspace_id"`
	CustomerID   *uint      `gorm:"index" json:"customer_id"`
	Source       string     `gorm:"not null;default:'local';uniqueIndex:idx_customer_vehicles_source" json:"source"`
	SourceID     *string    `gorm:"uniqueIndex:idx_customer_vehicles_source" json:"source_id"`
	Plate        string     `gorm:"not null;default:'';index:idx_customer_vehicles_plate,unique,where:plate <> ''" json:"plate"`
	Manufacturer string     `gorm:"not null;default:''" json:"manufacturer"`
	Model        string     `gorm:"not null;default:''" json:"model"`
	Transmission string     `gorm:"not null;default:''" json:"transmission"`
	YearFab      *int       `json:"year_fab"`
	YearModel    *int       `json:"year_model"`
	Fuel         string     `gorm:"not null;default:''" json:"fuel"`
	Engine       string     `gorm:"not null;default:''" json:"engine"`
	Doors        *int       `json:"doors"`
	Color        string     `gorm:"not null;default:''" json:"color"`
	Chassis      string     `gorm:"not null;default:''" json:"chassis"`
	Renavam      string     `gorm:"not null;default:''" json:"renavam"`
	LastMileage  *int       `json:"last_mileage"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	SyncedAt     *time.Time `json:"synced_at"`
}
