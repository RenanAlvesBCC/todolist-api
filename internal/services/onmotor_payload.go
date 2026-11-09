package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FlexString aceita texto, número ou booleano do robô/planilha (CPF numérico, ano como texto...).
// Chave ausente ou null: Set = false (no update mantém o valor atual).
type FlexString struct {
	Value string
	Set   bool
}

func Flex(v string) FlexString { return FlexString{Value: v, Set: true} }

func (f *FlexString) UnmarshalJSON(data []byte) error {
	raw := bytes.TrimSpace(data)
	switch {
	case len(raw) == 0, string(raw) == "null", raw[0] == '{', raw[0] == '[':
		return nil
	case raw[0] == '"':
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return err
		}
		f.Value, f.Set = s, true
	default:
		f.Value, f.Set = string(raw), true
	}
	return nil
}

func (f FlexString) trimmed() string { return strings.TrimSpace(f.Value) }

// ImportInput é um lote da importação OnMotor. O primeiro lote vem sem run_id.
type ImportInput struct {
	RunID     *uint            `json:"run_id"`
	Trigger   string           `json:"trigger"`
	Customers []ImportCustomer `json:"customers"`
	Vehicles  []ImportVehicle  `json:"vehicles"`
	Services  []ImportService  `json:"services"`
	Final     bool             `json:"final"`
}

type ImportCustomer struct {
	SourceID        FlexString `json:"source_id"`
	Name            FlexString `json:"name"`
	PersonType      FlexString `json:"person_type"`
	CPF             FlexString `json:"cpf"`
	CNPJ            FlexString `json:"cnpj"`
	RG              FlexString `json:"rg"`
	Phone           FlexString `json:"phone"`
	Mobile          FlexString `json:"mobile"`
	Email           FlexString `json:"email"`
	EmailSecondary  FlexString `json:"email_secondary"`
	CEP             FlexString `json:"cep"`
	Street          FlexString `json:"street"`
	Number          FlexString `json:"number"`
	Complement      FlexString `json:"complement"`
	District        FlexString `json:"district"`
	City            FlexString `json:"city"`
	UF              FlexString `json:"uf"`
	BirthDate       FlexString `json:"birth_date"`
	Notes           FlexString `json:"notes"`
	LastVisitAt     FlexString `json:"last_visit_at"`
	SourceUpdatedAt FlexString `json:"source_updated_at"`
}

type ImportVehicle struct {
	SourceID         FlexString `json:"source_id"`
	CustomerSourceID FlexString `json:"customer_source_id"`
	Plate            FlexString `json:"plate"`
	Manufacturer     FlexString `json:"manufacturer"`
	Model            FlexString `json:"model"`
	Transmission     FlexString `json:"transmission"`
	YearFab          FlexString `json:"year_fab"`
	YearModel        FlexString `json:"year_model"`
	Fuel             FlexString `json:"fuel"`
	Engine           FlexString `json:"engine"`
	Doors            FlexString `json:"doors"`
	Color            FlexString `json:"color"`
	Chassis          FlexString `json:"chassis"`
	Renavam          FlexString `json:"renavam"`
	LastMileage      FlexString `json:"last_mileage"`
}

type ImportService struct {
	SourceID         FlexString      `json:"source_id"`
	CustomerSourceID FlexString      `json:"customer_source_id"`
	VehicleSourceID  FlexString      `json:"vehicle_source_id"`
	Plate            FlexString      `json:"plate"`
	Mileage          FlexString      `json:"mileage"`
	Type             FlexString      `json:"type"`
	Status           FlexString      `json:"status"`
	OpenedAt         FlexString      `json:"opened_at"`
	DiagnosedAt      FlexString      `json:"diagnosed_at"`
	QuotedAt         FlexString      `json:"quoted_at"`
	StartedAt        FlexString      `json:"started_at"`
	TestedAt         FlexString      `json:"tested_at"`
	FinishedAt       FlexString      `json:"finished_at"`
	NotifiedAt       FlexString      `json:"notified_at"`
	DeliveredAt      FlexString      `json:"delivered_at"`
	Notes            FlexString      `json:"notes"`
	Items            json.RawMessage `json:"items"`
	Total            FlexString      `json:"total"`
}

var (
	errInvalidNumber = errors.New("número inválido")
	errInvalidTime   = errors.New("data/hora inválida")
	thousandsPattern = regexp.MustCompile(`^\d{1,3}(\.\d{3})+$`)
)

// parseImportInt aceita "87500", 87500, "87.500" (milhar), "87500,0" e "2015.0"; vazio = nil.
func parseImportInt(value string) (*int, error) {
	s := strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	s = strings.TrimSuffix(strings.ToLower(s), "km")
	if s == "" {
		return nil, nil
	}
	if n, err := strconv.Atoi(s); err == nil {
		return &n, nil
	}
	if thousandsPattern.MatchString(s) {
		n, err := strconv.Atoi(strings.ReplaceAll(s, ".", ""))
		if err != nil {
			return nil, errInvalidNumber
		}
		return &n, nil
	}
	f, err := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || math.Abs(f) > math.MaxInt32 {
		return nil, errInvalidNumber
	}
	n := int(math.Floor(f))
	return &n, nil
}

// parseImportDecimal aceita 1234.56, "1.234,56", "R$ 1234,56"; arredonda em centavos; vazio = nil.
func parseImportDecimal(value string) (*float64, error) {
	s := strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	s = strings.TrimPrefix(strings.ToUpper(s), "R$")
	if s == "" {
		return nil, nil
	}
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.Replace(s, ",", ".", 1)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return nil, errInvalidNumber
	}
	f = math.Round(f*100) / 100
	return &f, nil
}

var importTimeLayouts = []string{
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04",
	"2006-01-02",
	"02/01/2006 15:04:05",
	"02/01/2006 15:04",
	"02/01/2006",
}

// parseImportTime: RFC3339 com fuso, ou data/hora sem fuso no horário de Brasília. Devolve em UTC.
func parseImportTime(value string) (*time.Time, error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		u := t.UTC()
		return &u, nil
	}
	for _, layout := range importTimeLayouts {
		if t, err := time.ParseInLocation(layout, s, brazilTime); err == nil {
			u := t.UTC()
			return &u, nil
		}
	}
	return nil, errInvalidTime
}

// normalizeItems devolve o array JSON compactado; ok = false se não for array.
func normalizeItems(raw json.RawMessage) (string, bool) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil || items == nil {
		return "[]", false
	}
	var buf bytes.Buffer
	_ = json.Compact(&buf, raw)
	return buf.String(), true
}

func itemsSent(raw json.RawMessage) bool {
	t := bytes.TrimSpace(raw)
	return len(t) > 0 && string(t) != "null"
}
