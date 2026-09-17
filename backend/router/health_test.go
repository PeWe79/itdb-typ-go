package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthRoutes(t *testing.T) {
	app := &App{cfg: Config{CORSOrigins: []string{"*"}}}
	handler := app.routes()
	for _, path := range []string{"/health", "/api/health"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, res.Code)
		}
	}
}
