package services

import (
	"errors"
	"strings"
)

var (
	ErrInvalidCPF      = errors.New("CPF inválido")
	ErrInvalidCEP      = errors.New("CEP inválido")
	ErrInvalidWhatsApp = errors.New("WhatsApp inválido")
)

// onlyDigits tira a máscara (só os separadores permitidos em allowed) e devolve
// os dígitos; ok=false se aparecer qualquer outro caractere.
func onlyDigits(value, allowed string) (string, bool) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(value) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case strings.ContainsRune(allowed, r):
		default:
			return "", false
		}
	}
	return b.String(), true
}

// normalizeCPF aceita vazio ou CPF com/sem máscara (123.456.789-09) e devolve só os 11 dígitos.
func normalizeCPF(value string) (string, error) {
	digits, ok := onlyDigits(value, ".- ")
	if !ok {
		return "", ErrInvalidCPF
	}
	if digits == "" {
		return "", nil
	}
	if len(digits) != 11 || strings.Count(digits, digits[:1]) == 11 {
		return "", ErrInvalidCPF
	}
	if cpfCheckDigit(digits[:9]) != digits[9] || cpfCheckDigit(digits[:10]) != digits[10] {
		return "", ErrInvalidCPF
	}
	return digits, nil
}

func cpfCheckDigit(base string) byte {
	sum := 0
	weight := len(base) + 1
	for i := 0; i < len(base); i++ {
		sum += int(base[i]-'0') * weight
		weight--
	}
	rest := sum % 11
	if rest < 2 {
		return '0'
	}
	return byte('0' + 11 - rest)
}

// normalizeCEP aceita vazio ou 8 dígitos (com ou sem hífen: 12345-678).
func normalizeCEP(value string) (string, error) {
	digits, ok := onlyDigits(value, ".- ")
	if !ok || (digits != "" && len(digits) != 8) {
		return "", ErrInvalidCEP
	}
	return digits, nil
}

// normalizeWhatsApp aceita vazio ou telefone brasileiro (DDD + número, 10 ou 11 dígitos),
// com máscara, espaços e prefixo +55/55 opcional. Devolve só DDD + número.
func normalizeWhatsApp(value string) (string, error) {
	digits, ok := onlyDigits(value, "+()-. ")
	if !ok {
		return "", ErrInvalidWhatsApp
	}
	if digits == "" {
		return "", nil
	}
	if rest := strings.TrimPrefix(digits, "55"); rest != digits && (len(rest) == 10 || len(rest) == 11) {
		digits = rest
	}
	if (len(digits) != 10 && len(digits) != 11) || digits[0] == '0' {
		return "", ErrInvalidWhatsApp
	}
	return digits, nil
}
