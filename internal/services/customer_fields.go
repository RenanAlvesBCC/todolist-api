package services

import (
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
)

var (
	ErrCustomerNameRequired = errors.New("nome é obrigatório")
	ErrInvalidCNPJ          = errors.New("CNPJ inválido")
	ErrInvalidPhone         = errors.New("telefone inválido")
	ErrInvalidMobile        = errors.New("celular inválido")
	ErrInvalidEmail         = errors.New("e-mail inválido")
	ErrInvalidBirthDate     = errors.New("data de nascimento inválida")
	ErrInvalidPersonType    = errors.New("tipo de pessoa inválido")
	ErrInvalidUF            = errors.New("UF inválida")
	ErrPlateRequired        = errors.New("placa é obrigatória")
	ErrInvalidYear          = errors.New("ano inválido")
	ErrInvalidDoors         = errors.New("número de portas inválido")
)

// brazilTime: horários do OnMotor chegam sem fuso; o Brasil (Brasília) não tem mais horário de verão.
var brazilTime = time.FixedZone("BRT", -3*60*60)

// normalizeCNPJ aceita vazio ou CNPJ com/sem máscara (12.345.678/0001-95) e devolve os 14 dígitos.
func normalizeCNPJ(value string) (string, error) {
	digits, ok := onlyDigits(value, "./- ")
	if !ok {
		return "", ErrInvalidCNPJ
	}
	if digits == "" {
		return "", nil
	}
	if len(digits) != 14 || strings.Count(digits, digits[:1]) == 14 {
		return "", ErrInvalidCNPJ
	}
	if cnpjCheckDigit(digits[:12]) != digits[12] || cnpjCheckDigit(digits[:13]) != digits[13] {
		return "", ErrInvalidCNPJ
	}
	return digits, nil
}

func cnpjCheckDigit(base string) byte {
	sum := 0
	weight := len(base) - 7
	for i := 0; i < len(base); i++ {
		sum += int(base[i]-'0') * weight
		weight--
		if weight < 2 {
			weight = 9
		}
	}
	rest := sum % 11
	if rest < 2 {
		return '0'
	}
	return byte('0' + 11 - rest)
}

func normalizePhone(value string) (string, error) {
	return normalizeBRPhone(value, ErrInvalidPhone)
}

func normalizeMobile(value string) (string, error) {
	return normalizeBRPhone(value, ErrInvalidMobile)
}

// normalizeEmail: vazio ou um e-mail simples (algo@dominio.tld), sem espaços; grava em minúsculas.
func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if email == "" {
		return "", nil
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || strings.Count(email, "@") != 1 || strings.ContainsAny(email, " \t\n,;") {
		return "", ErrInvalidEmail
	}
	domain := email[at+1:]
	if dot := strings.LastIndex(domain, "."); dot < 1 || dot == len(domain)-1 {
		return "", ErrInvalidEmail
	}
	return email, nil
}

// normalizePersonType aceita "", pf/pj e as variações do OnMotor (F, J, física, jurídica).
func normalizePersonType(value string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "":
		return "", nil
	case "pf", "f", "fisica", "física", "pessoa fisica", "pessoa física":
		return models.PersonTypePF, nil
	case "pj", "j", "juridica", "jurídica", "pessoa juridica", "pessoa jurídica":
		return models.PersonTypePJ, nil
	}
	return "", ErrInvalidPersonType
}

func normalizeUF(value string) (string, error) {
	uf := strings.ToUpper(strings.TrimSpace(value))
	if uf == "" {
		return "", nil
	}
	if len(uf) != 2 || !unicode.IsLetter(rune(uf[0])) || !unicode.IsLetter(rune(uf[1])) {
		return "", ErrInvalidUF
	}
	return uf, nil
}

var dateLayouts = []string{"2006-01-02", "02/01/2006", time.RFC3339}

// parseBirthDate aceita vazio (nil), AAAA-MM-DD, DD/MM/AAAA ou RFC3339; data futura é inválida.
func parseBirthDate(value string, now time.Time) (*models.Date, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil, nil
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			if t.Year() < 1900 || t.After(now) {
				return nil, ErrInvalidBirthDate
			}
			d := models.NewDate(t)
			return &d, nil
		}
	}
	return nil, ErrInvalidBirthDate
}

// normalizeCustomerPlate deixa só letras e números, em maiúsculas (ABC-1D23 → ABC1D23).
func normalizeCustomerPlate(value string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(value)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func validYear(year *int, now time.Time) bool {
	return year == nil || (*year >= 1900 && *year <= now.Year()+2)
}

func validDoors(doors *int) bool {
	return doors == nil || (*doors >= 0 && *doors <= 10)
}

func validMileage(km *int) bool {
	return km == nil || (*km >= 0 && *km <= MaxMileage)
}

func trimmed(s string) string { return strings.TrimSpace(s) }
