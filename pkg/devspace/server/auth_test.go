package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loft-sh/devspace/pkg/devspace/config"
	"github.com/loft-sh/devspace/pkg/devspace/config/localcache"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	devspacecontext "github.com/loft-sh/devspace/pkg/devspace/context"
	"github.com/loft-sh/devspace/pkg/util/log"
	"gotest.tools/assert"
)

func TestAuthMiddleware(t *testing.T) {
	h := &handler{authToken: "test-token"}

	protected := authMiddleware(h, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("rejects unauthenticated callers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
		recorder := httptest.NewRecorder()

		protected.ServeHTTP(recorder, req)

		assert.Equal(t, recorder.Code, http.StatusUnauthorized)
	})

	t.Run("accepts cookie auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
		req.AddCookie(&http.Cookie{Name: authCookieName, Value: "test-token"})
		recorder := httptest.NewRecorder()

		protected.ServeHTTP(recorder, req)

		assert.Equal(t, recorder.Code, http.StatusNoContent)
	})

	t.Run("accepts bearer auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		recorder := httptest.NewRecorder()

		protected.ServeHTTP(recorder, req)

		assert.Equal(t, recorder.Code, http.StatusNoContent)
	})

	t.Run("accepts ui token header auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
		req.Header.Set(authHeaderName, "test-token")
		recorder := httptest.NewRecorder()

		protected.ServeHTTP(recorder, req)

		assert.Equal(t, recorder.Code, http.StatusNoContent)
	})
}

func TestProtectedRoutesRequireAuth(t *testing.T) {
	h := &handler{
		authToken: "test-token",
		protectUI: true,
		mux:       http.NewServeMux(),
		path:      ".",
	}
	h.registerRoutes()

	paths := []string{
		"/api/command?name=test",
		"/api/resource?resource=pods",
		"/api/config",
		"/api/forward?name=test&port=8080",
		"/api/enter?name=test&container=app",
		"/api/resize?resize_id=test&width=80&height=24",
		"/api/logs?name=test&container=app",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()

			h.ServeHTTP(recorder, req)

			assert.Equal(t, recorder.Code, http.StatusUnauthorized)
		})
	}
}

func TestInternalAndDiscoveryRoutesRemainUnauthenticated(t *testing.T) {
	h := &handler{
		protectUI: true,
		mux:       http.NewServeMux(),
		path:      ".",
	}
	h.registerRoutes()

	t.Run("api version stays public", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)

		assert.Equal(t, recorder.Code, http.StatusOK)
	})

	t.Run("api ping is still reachable without auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/ping", nil)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)

		assert.Equal(t, recorder.Code, http.StatusBadRequest)
	})

	t.Run("api exclude-dependency is still reachable without auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/exclude-dependency", nil)
		recorder := httptest.NewRecorder()

		h.ServeHTTP(recorder, req)

		assert.Equal(t, recorder.Code, http.StatusBadRequest)
	})
}

func TestIndexAuthorizesAndSetsCookie(t *testing.T) {
	h := &handler{authToken: "test-token", protectUI: true}
	req := httptest.NewRequest(http.MethodGet, "/?devspace-ui-token=test-token", nil)
	recorder := httptest.NewRecorder()

	ok := h.authorizeIndexRequest(recorder, req)

	assert.Equal(t, ok, false)
	assert.Equal(t, recorder.Code, http.StatusTemporaryRedirect)
	assert.Equal(t, recorder.Header().Get("Location"), "/")

	cookies := recorder.Result().Cookies()
	assert.Equal(t, len(cookies), 1)
	assert.Equal(t, cookies[0].Name, authCookieName)
	assert.Equal(t, cookies[0].Value, "test-token")
}

func TestHandleAuthenticatedSkipsAuthWhenUIProtectionDisabled(t *testing.T) {
	h := &handler{
		mux: http.NewServeMux(),
	}
	h.handleAuthenticated("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	recorder := httptest.NewRecorder()

	h.mux.ServeHTTP(recorder, req)

	assert.Equal(t, recorder.Code, http.StatusNoContent)
}

func TestAuthorizeIndexRequestSkipsAuthWhenUIProtectionDisabled(t *testing.T) {
	h := &handler{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	ok := h.authorizeIndexRequest(recorder, req)

	assert.Equal(t, ok, true)
	assert.Equal(t, len(recorder.Result().Cookies()), 0)
}

func TestRedactSensitiveVars(t *testing.T) {
	vars := map[string]interface{}{
		"dbPassword":         "secret",
		"serviceCredentials": "credentials",
		"sshKey":             "private-key",
		"APITOKEN":           "api-token",
		"DBPASSWORD":         "db-password",
		"SSHKEY":             "ssh-private-key",
		"monkey":             "banana",
	}

	redacted := redactSensitiveVars(vars)

	assert.Equal(t, redacted["dbPassword"], "***")
	assert.Equal(t, redacted["serviceCredentials"], "***")
	assert.Equal(t, redacted["sshKey"], "***")
	assert.Equal(t, redacted["APITOKEN"], "***")
	assert.Equal(t, redacted["DBPASSWORD"], "***")
	assert.Equal(t, redacted["SSHKEY"], "***")
	assert.Equal(t, redacted["monkey"], "banana")
	assert.Equal(t, vars["dbPassword"], "secret")
}

func TestReturnConfigRespectsUIProtectionSetting(t *testing.T) {
	vars := map[string]interface{}{
		"dbPassword": "secret",
		"monkey":     "banana",
	}

	t.Run("protect ui redacts sensitive vars", func(t *testing.T) {
		parsedVars := returnConfigVars(t, true, vars)
		assert.Equal(t, parsedVars["dbPassword"], "***")
		assert.Equal(t, parsedVars["monkey"], "banana")
	})

	t.Run("default ui returns original vars", func(t *testing.T) {
		parsedVars := returnConfigVars(t, false, vars)
		assert.Equal(t, parsedVars["dbPassword"], "secret")
		assert.Equal(t, parsedVars["monkey"], "banana")
	})
}

func returnConfigVars(t *testing.T, protectUI bool, vars map[string]interface{}) map[string]interface{} {
	t.Helper()

	conf := config.NewConfig(
		map[string]interface{}{},
		map[string]interface{}{},
		latest.NewRaw(),
		localcache.New(""),
		nil,
		vars,
		"",
	)
	ctx := devspacecontext.NewContext(context.Background(), nil, log.Discard).WithConfig(conf)
	h := &handler{
		ctx:       ctx,
		protectUI: protectUI,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	recorder := httptest.NewRecorder()

	h.returnConfig(recorder, req)

	assert.Equal(t, recorder.Code, http.StatusOK)

	payload := map[string]interface{}{}
	err := json.Unmarshal(recorder.Body.Bytes(), &payload)
	assert.NilError(t, err)

	generatedConfig := payload["generatedConfig"].(map[string]interface{})
	return generatedConfig["vars"].(map[string]interface{})
}
