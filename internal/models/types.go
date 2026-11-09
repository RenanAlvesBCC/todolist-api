package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

// Date é uma data sem hora: "AAAA-MM-DD" no JSON e coluna date no banco.
type Date struct{ time.Time }

func NewDate(t time.Time) Date {
	return Date{time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
}

func (Date) GormDataType() string { return "date" }

func (d Date) String() string { return d.Format(dateLayout) }

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Date) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return err
	}
	*d = NewDate(t)
	return nil
}

func (d Date) Value() (driver.Value, error) {
	return d.String(), nil
}

func (d *Date) Scan(value any) error {
	switch v := value.(type) {
	case time.Time:
		*d = NewDate(v)
		return nil
	case string:
		return d.parse(v)
	case []byte:
		return d.parse(string(v))
	}
	return fmt.Errorf("tipo inesperado para data: %T", value)
}

func (d *Date) parse(s string) error {
	if len(s) < len(dateLayout) {
		return errors.New("data inválida: " + s)
	}
	t, err := time.Parse(dateLayout, s[:len(dateLayout)])
	if err != nil {
		return err
	}
	*d = NewDate(t)
	return nil
}

// JSONText guarda JSON cru numa coluna text e o devolve como JSON (não como string).
type JSONText string

func (JSONText) GormDataType() string { return "text" }

func (j JSONText) MarshalJSON() ([]byte, error) {
	if !json.Valid([]byte(j)) {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

func (j JSONText) Value() (driver.Value, error) { return string(j), nil }

func (j *JSONText) Scan(value any) error {
	switch v := value.(type) {
	case string:
		*j = JSONText(v)
	case []byte:
		*j = JSONText(v)
	case nil:
		*j = ""
	default:
		return fmt.Errorf("tipo inesperado para JSON: %T", value)
	}
	return nil
}

// StringList é uma lista de textos guardada como array JSON numa coluna text.
type StringList []string

func (StringList) GormDataType() string { return "text" }

func (l StringList) MarshalJSON() ([]byte, error) {
	if l == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(l))
}

func (l StringList) Value() (driver.Value, error) {
	if l == nil {
		return "[]", nil
	}
	raw, err := json.Marshal([]string(l))
	return string(raw), err
}

func (l *StringList) Scan(value any) error {
	var raw string
	switch v := value.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	case nil:
		*l = StringList{}
		return nil
	default:
		return fmt.Errorf("tipo inesperado para lista: %T", value)
	}
	if strings.TrimSpace(raw) == "" {
		*l = StringList{}
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return err
	}
	*l = out
	return nil
}
