package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlexString_Unmarshal(t *testing.T) {
	var rec struct {
		A FlexString `json:"a"`
		B FlexString `json:"b"`
		C FlexString `json:"c"`
		D FlexString `json:"d"`
		E FlexString `json:"e"`
		F FlexString `json:"f"`
		G FlexString `json:"g"`
	}
	require.NoError(t, json.Unmarshal([]byte(`{"a":"texto","b":12345678909,"c":null,"d":true,"e":{"x":1},"f":[1]}`), &rec))
	assert.Equal(t, Flex("texto"), rec.A)
	assert.Equal(t, Flex("12345678909"), rec.B, "número vira texto")
	assert.False(t, rec.C.Set, "null = ausente")
	assert.Equal(t, Flex("true"), rec.D)
	assert.False(t, rec.E.Set, "objeto é ignorado")
	assert.False(t, rec.F.Set)
	assert.False(t, rec.G.Set, "chave ausente")

	var bad FlexString
	assert.Error(t, bad.UnmarshalJSON([]byte(`"sem fim`)))
	require.NoError(t, bad.UnmarshalJSON(nil))
	assert.False(t, bad.Set)
}

func TestParseImportInt(t *testing.T) {
	for in, want := range map[string]int{"87500": 87500, " 87.500 ": 87500, "87500 km": 87500, "2015.0": 2015, "87500,9": 87500, "1.234.567": 1234567} {
		got, err := parseImportInt(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, *got, in)
	}
	got, err := parseImportInt("  ")
	require.NoError(t, err)
	assert.Nil(t, got)
	for _, in := range []string{"abc", "NaN", "1e20", "12-3"} {
		_, err := parseImportInt(in)
		assert.Error(t, err, in)
	}
}

func TestParseImportDecimal(t *testing.T) {
	for in, want := range map[string]float64{"1234.56": 1234.56, "1.234,56": 1234.56, "R$ 99,9": 99.9, "10.005": 10.01, "-5": -5} {
		got, err := parseImportDecimal(in)
		require.NoError(t, err, in)
		assert.InDelta(t, want, *got, 0.0001, in)
	}
	got, err := parseImportDecimal("")
	require.NoError(t, err)
	assert.Nil(t, got)
	for _, in := range []string{"abc", "Inf", "1,2,3"} {
		_, err := parseImportDecimal(in)
		assert.Error(t, err, in)
	}
}

func TestParseImportTime(t *testing.T) {
	want := time.Date(2026, 3, 10, 12, 30, 0, 0, time.UTC)
	for _, in := range []string{"2026-03-10T12:30:00Z", "2026-03-10T09:30:00-03:00", "2026-03-10 09:30:00", "2026-03-10T09:30", "10/03/2026 09:30", "10/03/2026 09:30:00"} {
		got, err := parseImportTime(in)
		require.NoError(t, err, in)
		assert.True(t, want.Equal(*got), in)
		assert.Equal(t, time.UTC, got.Location())
	}
	day, err := parseImportTime("10/03/2026")
	require.NoError(t, err)
	assert.True(t, time.Date(2026, 3, 10, 3, 0, 0, 0, time.UTC).Equal(*day), "data sem hora = meia-noite de Brasília")
	got, err := parseImportTime("")
	require.NoError(t, err)
	assert.Nil(t, got)
	_, err = parseImportTime("ontem")
	assert.Error(t, err)
}

func TestNormalizeItems(t *testing.T) {
	got, ok := normalizeItems(json.RawMessage(`[ {"kind": "peca", "description": "Filtro"} ]`))
	assert.True(t, ok)
	assert.Equal(t, `[{"kind":"peca","description":"Filtro"}]`, got)
	got, ok = normalizeItems(json.RawMessage(`[]`))
	assert.True(t, ok)
	assert.Equal(t, "[]", got)
	for _, raw := range []string{`{"a":1}`, `"texto"`, `null`, `[1,`} {
		got, ok = normalizeItems(json.RawMessage(raw))
		assert.False(t, ok, raw)
		assert.Equal(t, "[]", got)
	}
	assert.False(t, itemsSent(nil))
	assert.False(t, itemsSent(json.RawMessage(" null ")))
	assert.True(t, itemsSent(json.RawMessage("[]")))
}
