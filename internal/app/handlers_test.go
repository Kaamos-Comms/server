package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateRoom(t *testing.T) {
	app := Initialize()
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/create", nil)
	rec := httptest.NewRecorder()

	app.e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "ok", response["status"])
}

func TestGetRoom(t *testing.T) {
	app := Initialize()
	req := httptest.NewRequest(http.MethodGet, "/api/rooms/test-room", nil)
	rec := httptest.NewRecorder()

	app.e.ServeHTTP(rec, req)

	// Should return 200 with "ok" status as per current implementation stub
	require.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Equal(t, "ok", response["status"])
}
