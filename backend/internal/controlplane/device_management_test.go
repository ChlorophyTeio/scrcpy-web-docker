package controlplane

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestManagedAppsAndProcesses(t *testing.T) {
	apps := parseManagedApps("package:com.example.one versionCode:12 uid:10101\npackage:com.example.system versionCode:2 uid:1000\npackage:com.example.off versionCode:3 uid:10202\n", "package:com.example.system\n", "package:com.example.off\n")
	if len(apps) != 3 {
		t.Fatal(apps)
	}
	var user, system, off managedApp
	for _, a := range apps {
		switch a.Package {
		case "com.example.one":
			user = a
		case "com.example.system":
			system = a
		case "com.example.off":
			off = a
		}
	}
	if user.Protected || !user.Enabled || !system.Protected || off.Enabled {
		t.Fatal(apps)
	}
	ps := parseManagedProcesses("UID PID PPID RSS VSZ STAT NAME\n10101 200 1 1024 50000 S com.example.one\n2000 201 1 1000 10000 S app_process\n1000 202 1 1000 10000 S system_server\n10303 203 1 1000 10000 S unknown\n", apps)
	if len(ps) != 4 || ps[0].Protected || ps[0].Package != "com.example.one" || !ps[1].Protected || !ps[2].Protected || !ps[3].Protected {
		t.Fatal(ps)
	}
	shared := append(apps, managedApp{Package: "com.example.shared", UID: 10101})
	if !parseManagedProcesses("10101 200 1 10 20 S com.example.one", shared)[0].Protected {
		t.Fatal("ambiguous shared UID must not be stopped")
	}
}
func TestManagementAuthorizationAndInput(t *testing.T) {
	s := newTestServer(t)
	s.store.UpsertAgentDevice("phone", DeviceInfo{})
	admin, _ := s.store.Authenticate("admin", "password")
	for _, route := range []string{"management", "file-tools"} {
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/devices/phone/"+route, nil))
		if w.Code != 401 {
			t.Fatal(route, w.Code)
		}
	}
	r := httptest.NewRequest("GET", "/api/devices/phone/management?view=apps", nil)
	r = r.WithContext(context.WithValue(r.Context(), userContextKey, User{Role: RoleUser, AssignedDevices: []string{"phone"}}))
	w := httptest.NewRecorder()
	s.deviceManagement(w, r, "phone")
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	for _, body := range []string{`{"action":"launch","package":"com.test;id"}`, `{"action":"erase","package":"com.example.one"}`, `{"action":"uninstall","package":"com.example.one","pid":12}`} {
		r := httptest.NewRequest("POST", "/api/devices/phone/management", strings.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), userContextKey, admin))
		w := httptest.NewRecorder()
		s.deviceManagement(w, r, "phone")
		if w.Code != 400 {
			t.Fatal(w.Code, body)
		}
	}
}
func TestFileToolsPathsAndMetadata(t *testing.T) {
	for _, p := range []string{"relative", "/sdcard/a\x00b", "/sdcard/a\nb"} {
		if _, e := fileToolPath(p); e == nil {
			t.Fatal(p)
		}
	}
	p, e := fileToolPath("/sdcard/a/../Download/测试 'x';.txt")
	if e != nil || p != "/sdcard/Download/测试 'x';.txt" {
		t.Fatal(p, e)
	}
	if managedStoragePath("/sdcardevil/file") || managedStoragePath("/system/file") || !managedStoragePath("/sdcard/Download/a") || !storageRoot("/data/local/tmp") {
		t.Fatal("root boundaries")
	}
	f, e := parseFileToolInfo("/sdcard/test", "regular file|123|1700000000|664\n")
	if e != nil || f.Size != 123 || f.Mode != "664" {
		t.Fatal(f, e)
	}
}
