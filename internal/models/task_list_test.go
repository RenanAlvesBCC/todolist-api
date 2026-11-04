package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func marshalToMap(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	return body
}

func TestTaskListMarshalJSON_CustomerDataHiddenByDefault(t *testing.T) {
	list := TaskList{Base: Base{ID: 3}, Title: "Gol", Customer: "Maria", CustomerCPF: "12345678909", CustomerCEP: "01310100", CustomerWhatsApp: "11987654321"}

	for _, v := range []any{list, &list, []TaskList{list}} {
		raw, err := json.Marshal(v)
		require.NoError(t, err)
		for _, secret := range []string{"customer_cpf", "customer_cep", "customer_whatsapp", "12345678909", "01310100", "11987654321", "CustomerDataVisible"} {
			assert.NotContains(t, string(raw), secret)
		}
	}

	body := marshalToMap(t, list)
	assert.Equal(t, float64(3), body["id"])
	assert.Equal(t, "Gol", body["title"])
	assert.Equal(t, "Maria", body["customer"])
}

func TestTaskListMarshalJSON_CustomerDataWhenVisible(t *testing.T) {
	list := &TaskList{Title: "Gol", CustomerCPF: "12345678909", CustomerDataVisible: true}

	body := marshalToMap(t, list)
	assert.Equal(t, "12345678909", body["customer_cpf"])
	assert.Equal(t, "", body["customer_cep"], "campo vazio vem como \"\" para quem pode ver")
	assert.Equal(t, "", body["customer_whatsapp"])
}
