package controlplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUnlockCurrentUserLockState(t *testing.T) {
	for _, tc := range []struct {
		input         string
		locked, known bool
	}{
		{`User "owner" (id=0) (current): deviceLocked=1, flag=0`, true, true},
		{`User "owner" (id=0) (current): deviceLocked=0, flag=0`, false, true},
		{"User (id=10): deviceLocked=0\nUser (id=0) (current): deviceLocked=1", true, true},
		{`User (id=10): deviceLocked=0`, false, false},
		{`deviceLocked=0`, false, false},
	} {
		locked, known := parseDeviceLocked(tc.input)
		if locked != tc.locked || known != tc.known {
			t.Fatalf("wrong state for %q", tc.input)
		}
	}
}
func TestUnlockPermissionAndInputBoundaries(t *testing.T) {
	s := newTestServer(t)
	s.store.UpsertAgentDevice("phone", DeviceInfo{})
	admin, _ := s.store.Authenticate("admin", "password")
	for _, tc := range []struct {
		name, method, body string
		user               User
		status             int
	}{
		{"method", http.MethodGet, `{}`, admin, 405},
		{"unassigned", http.MethodPost, `{"pin":"1234"}`, User{Role: RoleUser, Permissions: OperationPermissions{Shell: true}}, 403},
		{"no-shell", http.MethodPost, `{"pin":"1234"}`, User{Role: RoleUser, AssignedDevices: []string{"phone"}}, 403},
		{"short", http.MethodPost, `{"pin":"123"}`, admin, 400},
		{"shell-input", http.MethodPost, `{"pin":"1234; echo unsafe"}`, admin, 400},
		{"too-large", http.MethodPost, `{"pin":"` + strings.Repeat("1", 600) + `"}`, admin, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/api/devices/phone/unlock", strings.NewReader(tc.body))
			r = r.WithContext(context.WithValue(r.Context(), userContextKey, tc.user))
			w := httptest.NewRecorder()
			s.deviceUnlock(w, r, "phone")
			if w.Code != tc.status {
				t.Fatalf("status %d, want %d", w.Code, tc.status)
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("response must not be cached")
			}
		})
	}
	s.unlockLastAttempt.Store("phone", time.Now())
	r := httptest.NewRequest("POST", "/api/devices/phone/unlock", strings.NewReader(`{"pin":"1234"}`))
	r = r.WithContext(context.WithValue(r.Context(), userContextKey, admin))
	w := httptest.NewRecorder()
	s.deviceUnlock(w, r, "phone")
	if w.Code != 429 {
		t.Fatalf("duplicate attempt: %d", w.Code)
	}
}
func TestUnlockRouteRequiresAuthentication(t *testing.T) {
	s := newTestServer(t)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest("POST", "/api/devices/phone/unlock", strings.NewReader(`{"pin":"1234"}`)))
	if w.Code != 401 {
		t.Fatalf("unauthenticated status %d", w.Code)
	}
}
