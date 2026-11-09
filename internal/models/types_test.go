package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDate_JSONAndScan(t *testing.T) {
	d := NewDate(time.Date(1980, 5, 3, 15, 30, 0, 0, time.FixedZone("x", -3*3600)))
	raw, err := json.Marshal(d)
	require.NoError(t, err)
	assert.Equal(t, `"1980-05-03"`, string(raw))

	var back Date
	require.NoError(t, json.Unmarshal(raw, &back))
	assert.Equal(t, "1980-05-03", back.String())
	assert.Error(t, json.Unmarshal([]byte(`"03/05/1980"`), &back))
	assert.Error(t, json.Unmarshal([]byte(`12`), &back))

	v, err := d.Value()
	require.NoError(t, err)
	assert.Equal(t, "1980-05-03", v)

	var scanned Date
	for _, in := range []any{time.Date(1980, 5, 3, 0, 0, 0, 0, time.UTC), "1980-05-03", []byte("1980-05-03T00:00:00Z")} {
		require.NoError(t, scanned.Scan(in))
		assert.Equal(t, "1980-05-03", scanned.String())
	}
	assert.Error(t, scanned.Scan(42))
	assert.Error(t, scanned.Scan("80"))
	assert.Error(t, scanned.Scan("1980-13-45"))
	assert.Equal(t, "date", Date{}.GormDataType())
}

func TestJSONText(t *testing.T) {
	raw, err := json.Marshal(struct {
		Items JSONText `json:"items"`
	}{JSONText(`[{"kind":"peca"}]`)})
	require.NoError(t, err)
	assert.JSONEq(t, `{"items":[{"kind":"peca"}]}`, string(raw))

	raw, err = json.Marshal(JSONText("não é json"))
	require.NoError(t, err)
	assert.Equal(t, "null", string(raw))

	var j JSONText
	require.NoError(t, j.Scan("[1]"))
	assert.Equal(t, JSONText("[1]"), j)
	require.NoError(t, j.Scan([]byte("[2]")))
	assert.Equal(t, JSONText("[2]"), j)
	require.NoError(t, j.Scan(nil))
	assert.Equal(t, JSONText(""), j)
	assert.Error(t, j.Scan(3))
	v, _ := JSONText("[]").Value()
	assert.Equal(t, "[]", v)
	assert.Equal(t, "text", JSONText("").GormDataType())
}

func TestStringList(t *testing.T) {
	raw, err := json.Marshal(StringList(nil))
	require.NoError(t, err)
	assert.Equal(t, "[]", string(raw))
	raw, err = json.Marshal(StringList{"a"})
	require.NoError(t, err)
	assert.Equal(t, `["a"]`, string(raw))

	v, _ := StringList(nil).Value()
	assert.Equal(t, "[]", v)
	v, _ = StringList{"x", "y"}.Value()
	assert.Equal(t, `["x","y"]`, v)

	var l StringList
	require.NoError(t, l.Scan(`["x"]`))
	assert.Equal(t, StringList{"x"}, l)
	require.NoError(t, l.Scan([]byte(`["y"]`)))
	assert.Equal(t, StringList{"y"}, l)
	require.NoError(t, l.Scan(nil))
	assert.Equal(t, StringList{}, l)
	require.NoError(t, l.Scan(" "))
	assert.Equal(t, StringList{}, l)
	assert.Error(t, l.Scan("{"))
	assert.Error(t, l.Scan(1))
	assert.Equal(t, "text", StringList{}.GormDataType())
}

func TestTaskListMarshalJSON_CustomerLinksAreVisibleIDs(t *testing.T) {
	customerID, vehicleID := uint(4), uint(9)
	body := marshalToMap(t, TaskList{Title: "Gol", CustomerID: &customerID, CustomerVehicleID: &vehicleID})
	assert.Equal(t, float64(4), body["customer_id"])
	assert.Equal(t, float64(9), body["customer_vehicle_id"])

	body = marshalToMap(t, TaskList{Title: "Gol"})
	assert.Contains(t, body, "customer_id")
	assert.Nil(t, body["customer_id"])
	assert.Nil(t, body["customer_vehicle_id"])
}
