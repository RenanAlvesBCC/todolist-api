package services

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

const (
	MaxImportRecords   = 1000
	maxSyncWarnings    = 200
	maxSyncErrorLength = 500
	defaultRunsLimit   = 10
	maxRunsLimit       = 50
)

var (
	ErrIntegrationForbidden = errors.New("apenas dono ou gerente podem acessar a integração")
	ErrSyncRunNotFound      = errors.New("sincronização não encontrada")
	ErrSyncRunFinished      = errors.New("sincronização já finalizada")
	ErrImportBatchTooLarge  = errors.New("lote muito grande: máximo de 1000 registros por lista")
	ErrInvalidTrigger       = errors.New("trigger inválido")
)

// OnMotorStore é o que a importação precisa do cadastro. Os Find*By* devolvem nil, nil quando não encontram.
type OnMotorStore interface {
	CreateCustomer(c *models.Customer) error
	SaveCustomer(c *models.Customer) error
	TouchCustomerSynced(id uint, at time.Time) error
	FindCustomerByCPF(workspaceID uint, cpf string) (*models.Customer, error)
	FindCustomerBySource(workspaceID uint, source, sourceID string) (*models.Customer, error)
	CreateVehicle(v *models.CustomerVehicle) error
	SaveVehicle(v *models.CustomerVehicle) error
	TouchVehicleSynced(id uint, at time.Time) error
	FindVehicleByPlate(workspaceID uint, plate string) (*models.CustomerVehicle, error)
	FindVehicleBySource(workspaceID uint, source, sourceID string) (*models.CustomerVehicle, error)
	ClearVehiclePlate(id uint) error
	RaiseVehicleMileage(id uint, km int) error
	CreateServiceRecord(sr *models.ServiceRecord) error
	SaveServiceRecord(sr *models.ServiceRecord) error
	TouchServiceRecordSynced(id uint, at time.Time) error
	FindServiceRecordBySource(workspaceID uint, source, sourceID string) (*models.ServiceRecord, error)
}

type SyncRunStore interface {
	CreateRun(run *models.SyncRun) error
	SaveRun(run *models.SyncRun) error
	FindRun(workspaceID, id uint) (*models.SyncRun, error)
	ListRuns(workspaceID uint, limit int) ([]models.SyncRun, error)
}

// OnMotorSyncService recebe os dados do OnMotor (só leitura lá) e faz upsert aqui.
// Nunca apaga nada: o que sumir do OnMotor continua no cadastro.
type OnMotorSyncService struct {
	store   OnMotorStore
	runs    SyncRunStore
	wsStore WorkspaceRoleStore
	now     func() time.Time
}

func NewOnMotorSyncService(store OnMotorStore, runs SyncRunStore, wsStore WorkspaceRoleStore) *OnMotorSyncService {
	return &OnMotorSyncService{store: store, runs: runs, wsStore: wsStore, now: time.Now}
}

type SyncCounters struct {
	CustomersCreated int `json:"customers_created"`
	CustomersUpdated int `json:"customers_updated"`
	VehiclesCreated  int `json:"vehicles_created"`
	VehiclesUpdated  int `json:"vehicles_updated"`
	ServicesCreated  int `json:"services_created"`
	ServicesUpdated  int `json:"services_updated"`
	Skipped          int `json:"skipped"`
}

type ImportResult struct {
	RunID    uint              `json:"run_id"`
	Status   string            `json:"status"`
	Counters SyncCounters      `json:"counters"`
	Warnings models.StringList `json:"warnings"`
}

// ImportWithToken: lote do robô; trigger schedule (padrão) ou manual.
func (s *OnMotorSyncService) ImportWithToken(token *models.IntegrationToken, in ImportInput) (*ImportResult, error) {
	trigger := strings.TrimSpace(in.Trigger)
	if trigger == "" {
		trigger = models.SyncTriggerSchedule
	}
	if trigger != models.SyncTriggerSchedule && trigger != models.SyncTriggerManual {
		return nil, ErrInvalidTrigger
	}
	return s.importBatch(token.WorkspaceID, trigger, in)
}

// ImportManual: planilha enviada pelo painel (dono/gerente); trigger sempre upload.
func (s *OnMotorSyncService) ImportManual(userID uint, in ImportInput) (*ImportResult, error) {
	wsID, err := managerWorkspace(s.wsStore, userID, ErrIntegrationForbidden)
	if err != nil {
		return nil, err
	}
	return s.importBatch(wsID, models.SyncTriggerUpload, in)
}

// FailRun marca a execução como erro (falha do robô no meio da sincronização).
func (s *OnMotorSyncService) FailRun(token *models.IntegrationToken, runID uint, message string) (*models.SyncRun, error) {
	run, err := s.openRun(token.WorkspaceID, runID)
	if err != nil {
		return nil, err
	}
	message = strings.TrimSpace(message)
	if message == "" {
		message = "falha informada pelo robô"
	}
	if utf8.RuneCountInString(message) > maxSyncErrorLength {
		message = string([]rune(message)[:maxSyncErrorLength])
	}
	now := s.now().UTC()
	run.Status = models.SyncStatusError
	run.Error = message
	run.FinishedAt = &now
	if err := s.runs.SaveRun(run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *OnMotorSyncService) ListRuns(userID uint, limit int) ([]models.SyncRun, error) {
	wsID, err := managerWorkspace(s.wsStore, userID, ErrIntegrationForbidden)
	if err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = defaultRunsLimit
	}
	if limit > maxRunsLimit {
		limit = maxRunsLimit
	}
	return s.runs.ListRuns(wsID, limit)
}

func (s *OnMotorSyncService) openRun(wsID, runID uint) (*models.SyncRun, error) {
	run, err := s.runs.FindRun(wsID, runID)
	if err != nil {
		return nil, ErrSyncRunNotFound
	}
	if run.Status != models.SyncStatusRunning {
		return nil, ErrSyncRunFinished
	}
	return run, nil
}

func (s *OnMotorSyncService) importBatch(wsID uint, trigger string, in ImportInput) (*ImportResult, error) {
	if len(in.Customers) > MaxImportRecords || len(in.Vehicles) > MaxImportRecords || len(in.Services) > MaxImportRecords {
		return nil, ErrImportBatchTooLarge
	}
	now := s.now().UTC()

	var run *models.SyncRun
	if in.RunID == nil {
		run = &models.SyncRun{
			WorkspaceID: wsID, Source: models.SourceOnMotor, StartedAt: now,
			Status: models.SyncStatusRunning, Trigger: trigger, Warnings: models.StringList{},
		}
		if err := s.runs.CreateRun(run); err != nil {
			return nil, err
		}
	} else {
		var err error
		if run, err = s.openRun(wsID, *in.RunID); err != nil {
			return nil, err
		}
	}

	imp := &importer{store: s.store, wsID: wsID, run: run, now: now}
	if err := imp.importAll(in); err != nil {
		// Guarda o que já foi contado; o lote pode ser reenviado (upsert idempotente).
		_ = s.runs.SaveRun(run)
		return nil, err
	}

	if in.Final {
		run.Status = models.SyncStatusSuccess
		if run.Skipped > 0 || len(run.Warnings) > 0 {
			run.Status = models.SyncStatusPartial
		}
		run.FinishedAt = &now
	}
	if err := s.runs.SaveRun(run); err != nil {
		return nil, err
	}
	return resultOf(run), nil
}

func resultOf(run *models.SyncRun) *ImportResult {
	return &ImportResult{
		RunID:  run.ID,
		Status: run.Status,
		Counters: SyncCounters{
			CustomersCreated: run.CustomersCreated, CustomersUpdated: run.CustomersUpdated,
			VehiclesCreated: run.VehiclesCreated, VehiclesUpdated: run.VehiclesUpdated,
			ServicesCreated: run.ServicesCreated, ServicesUpdated: run.ServicesUpdated,
			Skipped: run.Skipped,
		},
		Warnings: run.Warnings,
	}
}

// importer processa um lote. Avisos nunca levam dados pessoais: só ids, posições e nomes de campo.
type importer struct {
	store OnMotorStore
	wsID  uint
	run   *models.SyncRun
	now   time.Time
}

const warningsTruncated = "demais avisos omitidos (limite de 200)"

func (imp *importer) warn(format string, args ...any) {
	switch n := len(imp.run.Warnings); {
	case n < maxSyncWarnings:
		imp.run.Warnings = append(imp.run.Warnings, fmt.Sprintf(format, args...))
	case n == maxSyncWarnings:
		imp.run.Warnings = append(imp.run.Warnings, warningsTruncated)
	}
}

func (imp *importer) skip(format string, args ...any) {
	imp.run.Skipped++
	imp.warn(format, args...)
}

func (imp *importer) importAll(in ImportInput) error {
	for i, rec := range in.Customers {
		if err := imp.customer(i+1, rec); err != nil {
			return err
		}
	}
	for i, rec := range in.Vehicles {
		if err := imp.vehicle(i+1, rec); err != nil {
			return err
		}
	}
	for i, rec := range in.Services {
		if err := imp.service(i+1, rec); err != nil {
			return err
		}
	}
	return nil
}

// normalized grava o campo normalizado; inválido vira "" com aviso. Ausente/null mantém.
func (imp *importer) normalized(label, field string, in FlexString, normalize func(string) (string, error), target *string) {
	if !in.Set {
		return
	}
	v, err := normalize(in.Value)
	if err != nil {
		imp.warn("%s: %s inválido (campo ignorado)", label, field)
		v = ""
	}
	*target = v
}

func plainField(in FlexString, target *string) {
	if in.Set {
		*target = in.trimmed()
	}
}

func (imp *importer) intField(label, field string, in FlexString, valid func(*int) bool, target **int) {
	if !in.Set {
		return
	}
	n, err := parseImportInt(in.Value)
	if err != nil || !valid(n) {
		imp.warn("%s: %s inválido (campo ignorado)", label, field)
		n = nil
	}
	*target = n
}

func (imp *importer) timeField(label, field string, in FlexString, target **time.Time) {
	if !in.Set {
		return
	}
	t, err := parseImportTime(in.Value)
	if err != nil {
		imp.warn("%s: %s inválido (campo ignorado)", label, field)
	}
	*target = t
}

func describeCustomer(c *models.Customer) string {
	if c.SourceID != nil {
		return "OnMotor " + *c.SourceID
	}
	return fmt.Sprintf("local #%d", c.ID)
}

func describeVehicle(v *models.CustomerVehicle) string {
	if v.SourceID != nil {
		return "OnMotor " + *v.SourceID
	}
	return fmt.Sprintf("local #%d", v.ID)
}
