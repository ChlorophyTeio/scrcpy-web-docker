package server

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"maa-server/config"
	"maa-server/scheduler"
)

func coreTestFixture(t *testing.T) string {
	t.Helper()
	oldHome := config.D.HomeDir
	oldState := coreState()
	config.D.HomeDir = t.TempDir()
	if e := os.MkdirAll(guiBase(), 0700); e != nil {
		t.Fatal(e)
	}
	coreUpdates.Lock()
	coreUpdates.State = coreUpdateState{Enabled: true, Time: "04:00", Timezone: "Asia/Shanghai"}
	coreUpdates.Unlock()
	t.Cleanup(func() {
		config.D.HomeDir = oldHome
		coreUpdates.Lock()
		coreUpdates.State = oldState
		coreUpdates.Unlock()
	})
	data := t.TempDir()
	for _, name := range []string{"lib", "resource"} {
		if e := os.MkdirAll(filepath.Join(data, name), 0755); e != nil {
			t.Fatal(e)
		}
	}
	if e := os.WriteFile(filepath.Join(data, "lib", "version"), []byte("v6.20.1"), 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(data, "resource", "data"), []byte("old resource"), 0644); e != nil {
		t.Fatal(e)
	}
	return data
}
func coreStubInstall(t *testing.T, data string, fail bool) *[]string {
	t.Helper()
	oldVersion, oldManifest, oldData, oldExec := coreInstalledVersion, coreGetManifest, coreDataDirectory, coreInstallStage
	t.Cleanup(func() {
		coreInstalledVersion = oldVersion
		coreGetManifest = oldManifest
		coreDataDirectory = oldData
		coreInstallStage = oldExec
	})
	coreInstalledVersion = func(context.Context) (string, error) {
		b, e := os.ReadFile(filepath.Join(data, "lib", "version"))
		return string(b), e
	}
	coreGetManifest = func(context.Context) (string, error) { return "v6.20.2", nil }
	coreDataDirectory = func(context.Context) (string, error) { return data, nil }
	calls := []string{}
	coreInstallStage = func(ctx context.Context, stage string) error {
		calls = append(calls, "maa install stable --batch --test-time 0")
		for _, name := range []string{"lib", "resource"} {
			_ = os.MkdirAll(filepath.Join(stage, name), 0755)
		}
		data := stage
		_ = os.WriteFile(filepath.Join(data, "lib", "version"), []byte("v6.20.2"), 0644)
		_ = os.WriteFile(filepath.Join(data, "resource", "data"), []byte("new resource"), 0644)
		if fail {
			return errors.New("simulated installer failure")
		}
		return nil
	}
	return &calls
}
func TestCoreStableVersionComparison(t *testing.T) {
	for _, tc := range []struct {
		latest, current string
		newer           bool
	}{{"v6.20.2", "v6.20.1", true}, {"v6.21.0", "v6.20.9", true}, {"v6.9.0", "v6.20.1", false}, {"v6.20.1", "v6.20.1", false}} {
		got, e := coreNewer(tc.latest, tc.current)
		if e != nil || got != tc.newer {
			t.Fatal(tc, got, e)
		}
	}
	if _, e := coreNewer("v6.21.0-beta.1", "v6.20.1"); e == nil {
		t.Fatal("prerelease accepted as stable")
	}
}
func TestCoreUpgradeBacksUpAndCanRollback(t *testing.T) {
	data := coreTestFixture(t)
	calls := coreStubInstall(t, data, false)
	sentinel := filepath.Join(guiBase(), "gui-web.json")
	_ = os.WriteFile(sentinel, []byte("existing daily plan"), 0600)
	if e := corePerformUpgrade(context.Background()); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(*calls, []string{"maa install stable --batch --test-time 0"}) {
		t.Fatal(*calls)
	}
	s := coreState()
	if s.Current != "v6.20.2" || s.BackupVersion != "v6.20.1" || s.Available {
		t.Fatal(s)
	}
	if e := corePerformRollback(context.Background()); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(filepath.Join(data, "resource", "data"))
	if string(b) != "old resource" {
		t.Fatal("resources were not restored")
	}
	b, _ = os.ReadFile(sentinel)
	if string(b) != "existing daily plan" {
		t.Fatal("daily plan was overwritten")
	}
}
func TestCoreFailedUpgradeAutomaticallyRestoresBothTrees(t *testing.T) {
	data := coreTestFixture(t)
	coreStubInstall(t, data, true)
	if e := corePerformUpgrade(context.Background()); e == nil {
		t.Fatal("failed installer reported success")
	}
	v, _ := os.ReadFile(filepath.Join(data, "lib", "version"))
	r, _ := os.ReadFile(filepath.Join(data, "resource", "data"))
	if string(v) != "v6.20.1" || string(r) != "old resource" {
		t.Fatal("partial installation was not restored")
	}
	if !strings.Contains(coreState().Message, "已自动恢复") {
		t.Fatal(coreState())
	}
}
func TestCoreManualCheckDoesNotInstall(t *testing.T) {
	data := coreTestFixture(t)
	calls := coreStubInstall(t, data, false)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/check", guiCoreCheck)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/check", strings.NewReader("{}")))
	if w.Code != 200 || !coreState().Available {
		t.Fatal(w.Code, w.Body.String())
	}
	if len(*calls) != 0 {
		t.Fatal("manual check installed core")
	}
	if _, e := os.Stat(filepath.Join(data, "web-core-backup")); !os.IsNotExist(e) {
		t.Fatal("manual check wrote backup/core files")
	}
	scheduler.ExecutionMu.Lock()
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/check", strings.NewReader("{}")))
	scheduler.ExecutionMu.Unlock()
	if w.Code != 409 {
		t.Fatal("check overlapped an active task")
	}
}
func TestCoreLatestVersionSkipsInstallAndBackup(t *testing.T) {
	data := coreTestFixture(t)
	calls := coreStubInstall(t, data, false)
	coreGetManifest = func(context.Context) (string, error) { return "v6.20.1", nil }
	if err := corePerformUpgrade(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 0 || coreState().Available {
		t.Fatal("already-current core was upgraded")
	}
	if _, err := os.Stat(filepath.Join(data, "web-core-backup")); !os.IsNotExist(err) {
		t.Fatal("unnecessary backup created")
	}
}
func TestCoreAutoScheduleSkipsBusyAndRunsOnlyOncePerLocalDay(t *testing.T) {
	coreTestFixture(t)
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Date(2026, 10, 27, 4, 0, 0, 0, loc)
	scheduler.ExecutionMu.Lock()
	coreAutoTick(now)
	scheduler.ExecutionMu.Unlock()
	s := coreState()
	if s.LastAutoDate != "2026-10-27" || !strings.Contains(s.Message, "已跳过") {
		t.Fatal(s)
	}
	// Same minute after becoming idle must not run a second attempt.
	coreAutoTick(now.Add(30 * time.Second))
	if !scheduler.ExecutionMu.TryLock() {
		t.Fatal("second attempt started")
	}
	scheduler.ExecutionMu.Unlock()
	coreAutoTick(now.Add(time.Hour))
	if coreState().LastAutoDate != s.LastAutoDate {
		t.Fatal("wrong schedule time")
	}
}
func TestCoreCancelledBackupPreservesPreviousBackup(t *testing.T) {
	data := coreTestFixture(t)
	if e := coreCreateBackup(context.Background(), data, "v6.20.1"); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := coreCreateBackup(ctx, data, "v6.20.2"); e == nil {
		t.Fatal("cancelled backup succeeded")
	}
	m, e := coreBackupInfo(data)
	if e != nil || m.Version != "v6.20.1" {
		t.Fatal("previous backup lost")
	}
}
func TestCoreScheduleSettingsValidateAndPersist(t *testing.T) {
	coreTestFixture(t)
	r := gin.New()
	r.POST("/settings", guiCoreSettings)
	for _, tc := range []struct {
		body   string
		status int
	}{{`{"enabled":true,"time":"25:00"}`, 400}, {`{"time":"04:00"}`, 400}, {`{"enabled":false,"time":"03:15"}`, 200}} {
		w := httptest.NewRecorder()
		q := httptest.NewRequest("POST", "/settings", strings.NewReader(tc.body))
		q.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, q)
		if w.Code != tc.status {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if coreState().Enabled || coreState().Time != "03:15" {
		t.Fatal("settings not applied")
	}
	i, e := os.Stat(filepath.Join(guiBase(), "core-update.json"))
	if e != nil || i.Mode().Perm() != 0600 {
		t.Fatal("private settings were not persisted")
	}
}

// A failed second rename must undo the first tree, leaving the active core usable.
func TestCoreStageSwapFailureRestoresOriginalTrees(t *testing.T) {
	data := coreTestFixture(t)
	stage := t.TempDir()
	if err := os.Mkdir(filepath.Join(stage, "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(stage, "lib", "version"), []byte("new"), 0644)
	if err := coreSwapTrees(data, stage); err == nil {
		t.Fatal("missing staged resource accepted")
	}
	v, _ := os.ReadFile(filepath.Join(data, "lib", "version"))
	r, _ := os.ReadFile(filepath.Join(data, "resource", "data"))
	if string(v) != "v6.20.1" || string(r) != "old resource" {
		t.Fatal("partial swap damaged active trees")
	}
}

func TestCoreStagedInstallerDoesNotTouchActiveCore(t *testing.T) {
	data := coreTestFixture(t)
	coreStubInstall(t, data, false)
	old := coreInstallStage
	coreInstallStage = func(ctx context.Context, stage string) error {
		if stage == data {
			t.Fatal("installing over active data directory")
		}
		if err := old(ctx, stage); err != nil {
			return err
		}
		v, _ := os.ReadFile(filepath.Join(data, "lib", "version"))
		if string(v) != "v6.20.1" {
			t.Fatal("installer modified active core before swap")
		}
		return nil
	}
	if err := corePerformUpgrade(context.Background()); err != nil {
		t.Fatal(err)
	}
}
