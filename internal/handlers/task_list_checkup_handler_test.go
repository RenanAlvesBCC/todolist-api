package handlers

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RenanAlvesBCC/oficina-api/internal/models"
	"github.com/RenanAlvesBCC/oficina-api/internal/services"
)

func TestTaskListHandler_SetCheckupVideo_ReturnsUpdatedVehicle(t *testing.T) {
	doneAt := time.Date(2026, 11, 4, 14, 30, 0, 0, time.UTC)
	doneBy := uint(3)
	name := "Ana Silva"
	var gotDone []bool
	router := setupRouter(&mockTaskListProvider{
		checkupVideoFunc: func(listID, userID uint, done bool) (*models.TaskList, error) {
			gotDone = append(gotDone, done)
			v := vehicleWithCustomerData(false)
			if done {
				v.CheckupVideoDoneAt, v.CheckupVideoDoneBy, v.CheckupVideoDoneByName = &doneAt, &doneBy, &name
			}
			return v, nil
		},
	})

	rec := jsonRequest(router, http.MethodPut, "/api/lists/7/checkup-video", map[string]bool{"done": true})
	require.Equal(t, http.StatusOK, rec.Code)
	body := decodeObject(t, rec)
	assert.Equal(t, "2026-11-04T14:30:00Z", body["checkup_video_done_at"])
	assert.Equal(t, float64(3), body["checkup_video_done_by"])
	assert.Equal(t, "Ana Silva", body["checkup_video_done_by_name"])
	assert.NotContains(t, body, "customer_cpf")

	rec = jsonRequest(router, http.MethodPut, "/api/lists/7/checkup-video", map[string]bool{"done": false})
	require.Equal(t, http.StatusOK, rec.Code)
	body = decodeObject(t, rec)
	for _, key := range []string{"checkup_video_done_at", "checkup_video_done_by", "checkup_video_done_by_name"} {
		value, present := body[key]
		assert.True(t, present, key+" sempre presente")
		assert.Nil(t, value, key+" null quando desmarcado")
	}

	assert.Equal(t, []bool{true, false}, gotDone)
}

func TestTaskListHandler_SetCheckupVideo_Errors(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		body   string
		svcErr error
		want   int
	}{
		{"id inválido", "/api/lists/abc/checkup-video", `{"done":true}`, nil, http.StatusBadRequest},
		{"body sem done", "/api/lists/7/checkup-video", `{}`, nil, http.StatusBadRequest},
		{"done não booleano", "/api/lists/7/checkup-video", `{"done":"sim"}`, nil, http.StatusBadRequest},
		{"sem acesso", "/api/lists/7/checkup-video", `{"done":true}`, services.ErrNotFound, http.StatusNotFound},
		{"erro genérico", "/api/lists/7/checkup-video", `{"done":true}`, errors.New("db"), http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := setupRouter(&mockTaskListProvider{
				checkupVideoFunc: func(listID, userID uint, done bool) (*models.TaskList, error) {
					return nil, tc.svcErr
				},
			})
			assert.Equal(t, tc.want, rawRequest(router, http.MethodPut, tc.path, tc.body).Code)
		})
	}
}
