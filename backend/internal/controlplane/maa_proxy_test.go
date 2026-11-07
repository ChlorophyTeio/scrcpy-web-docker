package controlplane

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMAAProxyAuthAndScopedSession(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("credentials leaked upstream")
		}
		switch r.URL.Path {
		case "/api/GetRunningTask":
			io.WriteString(w, `[]`)
		case "/api/check":
			io.WriteString(w, `{"ok":true}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer upstream.Close()
	t.Setenv("SCRCPYCAT_MAA_URL", upstream.URL)
	s := newTestServer(t)
	admin, _ := s.store.Authenticate("admin", "password")
	token, _ := issueUserToken(s.config.JWTSecret, admin)
	request := func(method, path, auth string, cookie *http.Cookie, site string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		if auth != "" {
			r.Header.Set("Authorization", "Bearer "+auth)
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if site != "" {
			r.Header.Set("Sec-Fetch-Site", site)
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/maa/api/check", "", nil, ""); w.Code != 401 {
		t.Fatalf("anonymous API: %d", w.Code)
	}
	if w := request("GET", "/maa/", "", nil, ""); w.Code != 303 || w.Header().Get("Location") != "/maa-console" {
		t.Fatalf("anonymous UI: %d", w.Code)
	}
	user, _ := s.store.CreateUser("maa-user", "password", RoleUser, nil)
	userToken, _ := issueUserToken(s.config.JWTSecret, user)
	for _, path := range []string{"/api/maa/session", "/maa/api/check"} {
		if w := request("POST", path, userToken, nil, ""); w.Code != 403 {
			t.Fatalf("user %s: %d", path, w.Code)
		}
	}
	w := request("POST", "/api/maa/session", token, nil, "")
	if w.Code != 200 {
		t.Fatalf("session: %d %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing cookie")
	}
	cookie := cookies[0]
	if cookie.Name != maaCookie || cookie.Path != "/maa" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("unsafe session scope")
	}
	for _, path := range []string{"/maa", "/maa/"} {
		if w := request("GET", path, "", cookie, "same-origin"); w.Code != 303 || w.Header().Get("Location") != "/maa-console" {
			t.Fatalf("legacy UI did not redirect: %d", w.Code)
		}
	}
	if w := request("GET", "/maa/api/check", "", cookie, "same-origin"); w.Code != 200 || !strings.Contains(w.Body.String(), "true") {
		t.Fatalf("proxy: %d", w.Code)
	}
	if w := request("GET", "/maa/api/check", "", cookie, "same-site"); w.Code != 403 {
		t.Fatalf("cross-origin cookie: %d", w.Code)
	}
	s.store.RevokeUserToken(tokenDigest(token), time.Now().Add(time.Hour))
	if w := request("GET", "/maa/api/check", "", cookie, "same-origin"); w.Code != 401 {
		t.Fatalf("revoked session: %d", w.Code)
	}
}
