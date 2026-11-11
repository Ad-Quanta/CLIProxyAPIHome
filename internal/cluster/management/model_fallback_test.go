package management

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPIHome/internal/cluster"
	appconfig "github.com/router-for-me/CLIProxyAPIHome/internal/config"
)

// TestConfigV8ModelFallbackRoundTrip verifies that the Management API persists
// the CPA model-fallback section and forwards it in the runtime payload.
func TestConfigV8ModelFallbackRoundTrip(t *testing.T) {
	db, cleanup := openManagementLogTestDB(t)
	defer cleanup()
	repo := cluster.NewRepository(db)
	h := NewHandler(repo, nil, "127.0.0.1", 0)
	router := gin.New()
	router.PUT("/v8/management/config/*path", h.ConfigV8)
	router.GET("/v8/management/config/*path", h.ConfigV8)
	request := func(method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		if rec.Code != status {
			t.Fatalf("%s %s: status %d, want %d: %s", method, path, rec.Code, status, rec.Body.String())
		}
		return rec
	}
	body := `{"enabled":true,"preserve-requested-model":false,"trigger-status-codes":[429,503],"no-fallback-status-codes":[400],"rules":[{"from":"primary","to":["secondary","tertiary"]}]}`
	request(http.MethodPut, "/v8/management/config/model-fallback", body, http.StatusOK)

	read := request(http.MethodGet, "/v8/management/config/model-fallback", "", http.StatusOK)
	var got appconfig.ModelFallback
	if errDecode := json.Unmarshal(read.Body.Bytes(), &got); errDecode != nil {
		t.Fatalf("decode model-fallback: %v", errDecode)
	}
	if !got.Enabled || got.PreserveRequestedModel == nil || *got.PreserveRequestedModel {
		t.Fatalf("model-fallback flags = %#v", got)
	}
	if len(got.Rules) != 1 || got.Rules[0].From != "primary" || len(got.Rules[0].To) != 2 {
		t.Fatalf("model-fallback rules = %#v", got.Rules)
	}

	cfg, payload, errLoad := repo.LoadConfigAsRuntimeConfig(context.Background())
	if errLoad != nil {
		t.Fatalf("LoadConfigAsRuntimeConfig() error = %v", errLoad)
	}
	if !cfg.ModelFallback.Enabled || len(cfg.ModelFallback.Rules) != 1 {
		t.Fatalf("runtime config lost model-fallback: %#v", cfg.ModelFallback)
	}
	if !strings.Contains(string(payload), "model-fallback:") || !strings.Contains(string(payload), "from: primary") {
		t.Fatalf("runtime payload lost model-fallback:\n%s", payload)
	}

	request(http.MethodPut, "/v8/management/config/model-fallback", `{"enabled":"not-a-bool"}`, http.StatusBadRequest)
}
