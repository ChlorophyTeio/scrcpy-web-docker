package controlplane

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestPhysicalScreenState(t *testing.T) {
	for _, tc := range []struct {
		s         string
		on, known bool
	}{
		{"Display 12\n connectionType=Internal\n powerMode=On", true, true},
		{"Display 12\n connectionType=Internal\n powerMode=Off", false, true},
		{"Display 12\n connectionType=Internal\n powerMode=DozeSuspend", false, true},
		{"Display 12\n powerMode=2", true, true},
		{"Display 12\n powerMode=0", false, true},
		{"Display 12\n powerMode=1", false, true},
		{"Display 12\n connectionType=External\n powerMode=On\nDisplay 99\n connectionType=Internal\n powerMode=Off", false, true},
		{"Display 12\n powerMode=On\nDisplay 99\n powerMode=Off", false, false},
		{"mWakefulness=Awake\n mScreenState=ON", false, false},
		{"Display 12\n powerMode=Unknown", false, false},
		{"Permission Denial", false, false},
		{"Display 12\n connectionType=Internal\n powerMode=Off\nVirtual Display 99\n powerMode=On", false, true},
	} {
		on, known := parsePhysicalScreen(tc.s)
		if on != tc.on || known != tc.known {
			t.Fatalf("%q: got %v,%v", tc.s, on, known)
		}
	}
}
func TestScreenStateAuthorization(t *testing.T) {
	s := newTestServer(t)
	admin, _ := s.store.Authenticate("admin", "password")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/devices/phone/screen-state", nil))
	if w.Code != 401 {
		t.Fatal(w.Code)
	}
	for _, tc := range []struct {
		method, id string
		user       User
		code       int
	}{
		{"POST", "phone", admin, 405}, {"GET", "phone", User{Role: RoleUser}, 403}, {"GET", "missing", admin, 404},
	} {
		r := httptest.NewRequest(tc.method, "/api/devices/"+tc.id+"/screen-state", nil)
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, tc.user))
		w := httptest.NewRecorder()
		s.deviceScreenState(w, r, tc.id)
		if w.Code != tc.code {
			t.Fatal(tc, w.Code)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cached state")
		}
	}
}
