package models

import "time"

type FlagType string

const (
	FlagAguardandoML       FlagType = "aguardando_ml"
	FlagProcurandoPeca     FlagType = "procurando_peca"
	FlagAguardandoCliente  FlagType = "aguardando_cliente"
	FlagAguardandoOrcamento FlagType = "aguardando_orcamento"
	FlagAguardandoEntrega  FlagType = "aguardando_entrega"
	FlagAguardandoRetirada FlagType = "aguardando_retirada"
	FlagAguardandoTerceiro FlagType = "aguardando_terceiro"
	FlagOutro              FlagType = "outro"
)

type PendingFlag struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskListID  uint       `gorm:"not null;index" json:"task_list_id"`
	CreatedBy   uint       `gorm:"not null" json:"created_by"`
	FlagType    FlagType   `gorm:"not null" json:"flag_type"`
	Note        string     `json:"note"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	ResolvedBy  *uint      `json:"resolved_by"`
	CreatedAt   time.Time  `json:"created_at"`
}
