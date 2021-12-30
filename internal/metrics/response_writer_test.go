package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gotest.tools/v3/assert"
)

// TODO: needs improvement
func TestResponseWriter(t *testing.T) {
	sResponseWriter := statusResponseWriter{}

	handler := func(w http.ResponseWriter, r *http.Request) {
		sResponseWriter.ResponseWriter = w
		sResponseWriter.WriteHeader(http.StatusBadGateway)
	}

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, sResponseWriter.status, http.StatusBadGateway)
}

