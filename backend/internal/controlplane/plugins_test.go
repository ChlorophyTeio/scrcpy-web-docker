package controlplane

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOptionalPluginDiscoveryAndProxy(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("main credentials leaked to plugin")
		}
		if r.URL.Path == "/health" {
			io.WriteString(w, `{"id":"sample","api_version":1}`)
			return
		}
		io.WriteString(w, `{"ok":true}`)
	}))
	defer target.Close()
	dir := t.TempDir()
	t.Setenv("SCRCPYCAT_BUNDLED_PLUGIN_DIR", dir)
	t.Setenv("SCRCPYCAT_PLUGIN_DIR", t.TempDir())
	manifest := pluginManifest{ID: "sample", Name: "Sample", Frontend: "sample", APIVersion: 1, Target: target.URL, HealthPath: "/health", HealthMatch: map[string]any{"id": "sample", "api_version": 1}}
	raw, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(dir, "sample.json"), raw, 0600)
	s := newTestServer(t)
	admin, _ := s.store.Authenticate("admin", "password")
	token, _ := issueUserToken(s.config.JWTSecret, admin)
	user, _ := s.store.CreateUser("plugin-user", "password", RoleUser, nil)
	userToken, _ := issueUserToken(s.config.JWTSecret, user)
	request := func(method, path, auth string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		if auth != "" {
			r.Header.Set("Authorization", "Bearer "+auth)
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/api/plugins", "", nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := request("GET", "/api/plugins", userToken, nil); w.Code != 403 {
		t.Fatal(w.Code)
	}
	w := request("GET", "/api/plugins", token, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"available":true`) || strings.Contains(w.Body.String(), target.URL) {
		t.Fatalf("discovery: %d %s", w.Code, w.Body.String())
	}
	w = request("POST", "/api/plugins/sample/session", token, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Path != "/api/plugins/sample/proxy" || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("invalid scoped cookie")
	}
	if w = request("GET", "/api/plugins/sample/proxy/data", "", cookies[0]); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = request("GET", "/api/plugins/sample/proxy/data", userToken, nil); w.Code != 403 {
		t.Fatal(w.Code)
	}
	r := httptest.NewRequest("GET", "/api/plugins/sample/proxy/data", nil)
	r.AddCookie(cookies[0])
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	target.Close()
	s.plugins.checked = time.Time{}
	w = request("GET", "/api/plugins", token, nil)
	if !strings.Contains(w.Body.String(), `"available":false`) {
		t.Fatal("stopped plugin remained available", w.Body.String())
	}
	enabled := false
	manifest.Enabled = &enabled
	raw, _ = json.Marshal(manifest)
	os.WriteFile(filepath.Join(dir, "sample.json"), raw, 0600)
	w = request("POST", "/api/plugins/sample/session", token, nil)
	if w.Code != 404 {
		t.Fatal("disabled plugin accepted", w.Code)
	}
}
func TestPluginManifestValidation(t *testing.T) {
	p := pluginManifest{ID: "sample", Name: "Sample", Frontend: "sample", APIVersion: 1, Target: "http://service:8080", HealthPath: "/health"}
	if !validPlugin(p) {
		t.Fatal("valid manifest rejected")
	}
	for _, bad := range []string{"file:///etc/passwd", "http://name:password@service", "javascript:alert(1)", "http://service?redirect=1"} {
		p.Target = bad
		if validPlugin(p) {
			t.Fatal("invalid target accepted", bad)
		}
	}
	p.Target = "http://service"
	p.HealthPath = "//external/health"
	if validPlugin(p) {
		t.Fatal("invalid health path accepted")
	}
}
