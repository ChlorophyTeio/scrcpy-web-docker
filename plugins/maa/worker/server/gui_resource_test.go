package server

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"maa-server/scheduler"
)

func TestResourceUpdateNeverUpgradesCoreOrTouchesPhone(t *testing.T) {
	original := guiExecuteCommand
	defer func() { guiExecuteCommand = original }()
	calls := []string{}
	guiExecuteCommand = func(ctx context.Context, name string, args ...string) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	scheduler.ExecutionMu.Lock()
	gui.Lock()
	gui.state = GUIState{Running: true, States: map[string]string{"resource_update": "pending"}}
	gui.Unlock()
	guiRun(ctx, cancel, GUIConfig{Enabled: []string{"resource_update"}, After: "none"}, nil)
	if !reflect.DeepEqual(calls, []string{"maa hot-update --batch"}) {
		t.Fatalf("unexpected maintenance commands: %v", calls)
	}
	if !scheduler.ExecutionMu.TryLock() {
		t.Fatal("execution lock was not released")
	}
	scheduler.ExecutionMu.Unlock()
	gui.Lock()
	defer gui.Unlock()
	if gui.state.Running || gui.state.States["resource_update"] != "completed" || !strings.Contains(gui.state.Message, "MaaCore 未升级") {
		t.Fatal("resource status incorrect")
	}
}

func TestResourceUpdateRejectsConcurrentTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/resource-update", guiResourceUpdate)
	scheduler.ExecutionMu.Lock()
	defer scheduler.ExecutionMu.Unlock()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/resource-update", strings.NewReader("{}")))
	if w.Code != 409 {
		t.Fatalf("expected busy response, got %d", w.Code)
	}
}
