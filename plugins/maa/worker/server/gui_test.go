package server

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"maa-server/config"
	"maa-server/scheduler"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGUIConfigAndTaskGeneration(t *testing.T) {
	c := guiDefaults()
	if err := guiValidate(c); err != nil {
		t.Fatal(err)
	}
	original := config.D.FightDir
	config.D.FightDir = t.TempDir()
	defer func() { config.D.FightDir = original }()
	for _, id := range c.Enabled {
		if id == "launch" {
			continue
		}
		path, err := guiTaskFile(c, id, "test")
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(config.D.FightDir, path+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var v struct {
			Tasks []struct {
				Type   string         `json:"type"`
				Params map[string]any `json:"params"`
			} `json:"tasks"`
		}
		if err = json.Unmarshal(raw, &v); err != nil || len(v.Tasks) != 1 || v.Tasks[0].Type != guiTypes[id] {
			t.Fatal("invalid generated task")
		}
		if id == "fight" {
			if v.Tasks[0].Params["stone"] != float64(0) || v.Tasks[0].Params["medicine"] != float64(0) {
				t.Fatal("consumables enabled by default")
			}
		}
	}
	c.Params["infrast"]["filename"] = "../../secret.json"
	if guiValidate(c) == nil {
		t.Fatal("path traversal accepted")
	}
	delete(c.Params["infrast"], "filename")
	c.Enabled = []string{"fight", "fight"}
	if guiValidate(c) == nil {
		t.Fatal("duplicate plan accepted")
	}
	c.Enabled = []string{"shell"}
	if guiValidate(c) == nil {
		t.Fatal("unknown task accepted")
	}
}
func TestGUIStartDoesNotOverlapScheduler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/start", guiStart)
	body, _ := json.Marshal(guiDefaults())
	scheduler.ExecutionMu.Lock()
	defer scheduler.ExecutionMu.Unlock()
	w := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/start", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, request)
	if w.Code != 409 {
		t.Fatalf("overlapping execution status %d", w.Code)
	}
}
func TestGUIStopRequiresCurrentRunID(t *testing.T) {
	called := false
	gui.Lock()
	gui.state = GUIState{Running: true, ID: "current"}
	gui.cancel = func() { called = true }
	gui.Unlock()
	defer func() { gui.Lock(); gui.state = GUIState{States: map[string]string{}}; gui.cancel = nil; gui.Unlock() }()
	r := gin.New()
	r.POST("/stop", guiStop)
	for _, tc := range []struct {
		id     string
		status int
	}{{"stale", 409}, {"current", 200}} {
		w := httptest.NewRecorder()
		request := httptest.NewRequest("POST", "/stop", strings.NewReader(`{"id":"`+tc.id+`"}`))
		request.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, request)
		if w.Code != tc.status {
			t.Fatal(w.Code)
		}
		if tc.id == "stale" && called {
			t.Fatal("stale request stopped current run")
		}
	}
	if !called {
		t.Fatal("stop did not cancel")
	}
}

func TestGUIRunnerCancelsOnceAndRestoresDisplay(t *testing.T) {
	original := guiExecuteCommand
	defer func() { guiExecuteCommand = original }()
	calls := []string{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	guiExecuteCommand = func(ctx context.Context, name string, args ...string) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name == "maa" {
			cancel()
			return context.Canceled
		}
		return nil
	}
	scheduler.ExecutionMu.Lock()
	gui.Lock()
	gui.state = GUIState{Running: true, States: map[string]string{"fight": "pending"}}
	gui.Unlock()
	guiRun(ctx, cancel, GUIConfig{Enabled: []string{"fight"}, After: "none"}, map[string]string{"fight": "fake/fight"})
	if len(calls) != 3 || !strings.HasSuffix(calls[0], "prepare") || !strings.HasSuffix(calls[2], "restore") {
		t.Fatalf("unexpected execution calls: %v", calls)
	}
	if !scheduler.ExecutionMu.TryLock() {
		t.Fatal("execution lock not released after restore")
	}
	scheduler.ExecutionMu.Unlock()
	gui.Lock()
	defer gui.Unlock()
	if gui.state.Running || gui.state.States["fight"] != "stopped" {
		t.Fatal("canceled state incorrect")
	}
}

func TestGUICommandCancelsItsChildProcessGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := guiCommand(ctx, "sh", "-c", "sleep 30"); err == nil {
		t.Fatal("canceled process reported success")
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("child process kept stop operation blocked")
	}
}
