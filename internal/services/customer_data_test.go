package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeCPF(t *testing.T) {
	valid := map[string]string{
		"":                 "",
		"   ":              "",
		"12345678909":      "12345678909",
		"123.456.789-09":   "12345678909",
		" 529.982.247-25 ": "52998224725",
		"529 982 247 25":   "52998224725",
		"111.444.777-35":   "11144477735",
		"000.000.001-91":   "00000000191",
	}
	for input, want := range valid {
		got, err := normalizeCPF(input)
		assert.NoError(t, err, input)
		assert.Equal(t, want, got, input)
	}

	invalid := []string{
		"12345678900",    // dígito verificador errado
		"123.456.789-19", // primeiro dígito errado
		"11111111111",    // sequência repetida (passa no cálculo)
		"000.000.000-00",
		"1234567890",     // 10 dígitos
		"123456789090",   // 12 dígitos
		"123.456.789/09", // separador não aceito
		"abc12345678909",
	}
	for _, input := range invalid {
		_, err := normalizeCPF(input)
		assert.ErrorIs(t, err, ErrInvalidCPF, input)
	}
}

func TestNormalizeCEP(t *testing.T) {
	valid := map[string]string{
		"":             "",
		"01310100":     "01310100",
		"01310-100":    "01310100",
		" 01.310-100 ": "01310100",
	}
	for input, want := range valid {
		got, err := normalizeCEP(input)
		assert.NoError(t, err, input)
		assert.Equal(t, want, got, input)
	}

	for _, input := range []string{"0131010", "013101000", "01310_100", "CEP 01310100"} {
		_, err := normalizeCEP(input)
		assert.ErrorIs(t, err, ErrInvalidCEP, input)
	}
}

func TestNormalizeWhatsApp(t *testing.T) {
	valid := map[string]string{
		"":                    "",
		"11987654321":         "11987654321",
		"(11) 98765-4321":     "11987654321",
		"+55 (11) 98765-4321": "11987654321",
		"5511987654321":       "11987654321",
		"55 11 3456-7890":     "1134567890",
		"+551134567890":       "1134567890",
		"(55) 99988-7766":     "55999887766", // DDD 55 sem código do país
		"5599887766":          "5599887766",
		"+55 55 99988-7766":   "55999887766",
		"11 3456.7890":        "1134567890",
	}
	for input, want := range valid {
		got, err := normalizeWhatsApp(input)
		assert.NoError(t, err, input)
		assert.Equal(t, want, got, input)
	}

	invalid := []string{
		"987654321",        // sem DDD
		"119876543210",     // 12 dígitos sem 55
		"+1 415 555 26710", // outro país (12 dígitos)
		"011987654321",     // DDD com zero
		"0119876543",
		"11 98765-4321 ramal 2",
		"55",
	}
	for _, input := range invalid {
		_, err := normalizeWhatsApp(input)
		assert.ErrorIs(t, err, ErrInvalidWhatsApp, input)
	}
}
