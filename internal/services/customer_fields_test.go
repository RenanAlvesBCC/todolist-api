package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeCNPJ(t *testing.T) {
	for in, want := range map[string]string{
		"":                   "",
		"11.222.333/0001-81": "11222333000181",
		"11444777000161":     "11444777000161",
	} {
		got, err := normalizeCNPJ(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got)
	}
	for _, in := range []string{"11.222.333/0001-80", "1122233300018", "11111111111111", "11.222.333/0001-8a"} {
		_, err := normalizeCNPJ(in)
		assert.ErrorIs(t, err, ErrInvalidCNPJ, in)
	}
}

func TestNormalizePhoneAndMobile(t *testing.T) {
	got, err := normalizePhone("(11) 3333-4444")
	require.NoError(t, err)
	assert.Equal(t, "1133334444", got)
	got, err = normalizeMobile("+55 11 98765-4321")
	require.NoError(t, err)
	assert.Equal(t, "11987654321", got)

	_, err = normalizePhone("3333-4444")
	assert.ErrorIs(t, err, ErrInvalidPhone)
	_, err = normalizeMobile("abc")
	assert.ErrorIs(t, err, ErrInvalidMobile)
}

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail("  Maria@Exemplo.com.BR ")
	require.NoError(t, err)
	assert.Equal(t, "maria@exemplo.com.br", got)
	got, err = normalizeEmail("")
	require.NoError(t, err)
	assert.Equal(t, "", got)
	for _, in := range []string{"maria", "@x.com", "a@b", "a@b.", "a@@b.com", "a b@c.com", "a@.com"} {
		_, err := normalizeEmail(in)
		assert.ErrorIs(t, err, ErrInvalidEmail, in)
	}
}

func TestNormalizePersonTypeAndUF(t *testing.T) {
	for in, want := range map[string]string{"": "", "PF": "pf", "F": "pf", "Física": "pf", "pj": "pj", "Jurídica": "pj"} {
		got, err := normalizePersonType(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got)
	}
	_, err := normalizePersonType("x")
	assert.ErrorIs(t, err, ErrInvalidPersonType)

	uf, err := normalizeUF(" sp ")
	require.NoError(t, err)
	assert.Equal(t, "SP", uf)
	uf, err = normalizeUF("")
	require.NoError(t, err)
	assert.Equal(t, "", uf)
	for _, in := range []string{"S", "SPX", "1A"} {
		_, err := normalizeUF(in)
		assert.ErrorIs(t, err, ErrInvalidUF, in)
	}
}

func TestParseBirthDate(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	for _, in := range []string{"1980-05-03", "03/05/1980", "1980-05-03T10:00:00-03:00"} {
		d, err := parseBirthDate(in, now)
		require.NoError(t, err, in)
		assert.Equal(t, "1980-05-03", d.String())
	}
	d, err := parseBirthDate(" ", now)
	require.NoError(t, err)
	assert.Nil(t, d)
	for _, in := range []string{"1980-13-01", "ontem", "1800-01-01", "2030-01-01"} {
		_, err := parseBirthDate(in, now)
		assert.ErrorIs(t, err, ErrInvalidBirthDate, in)
	}
}

func TestCustomerVehicleFieldHelpers(t *testing.T) {
	assert.Equal(t, "ABC1D23", normalizeCustomerPlate(" abc-1d23 "))
	assert.Equal(t, "ABC1234", normalizeCustomerPlate("ABC 1234"))
	assert.Equal(t, "", normalizeCustomerPlate(" - "))

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.True(t, validYear(nil, now))
	assert.True(t, validYear(ptr(2028), now))
	assert.False(t, validYear(ptr(2029), now))
	assert.False(t, validYear(ptr(1899), now))
	assert.True(t, validDoors(ptr(4)))
	assert.False(t, validDoors(ptr(11)))
	assert.True(t, validMileage(nil))
	assert.False(t, validMileage(ptr(-1)))
}
