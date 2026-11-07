package server

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"maa-server/scheduler"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func schedulesFixture(t *testing.T) {
	coreTestFixture(t)
	old := guiSchedules.Items
	oldErr := guiSchedules.Err
	oldStart := guiScheduledBegin
	guiSchedules.Items = []GUISchedule{}
	guiSchedules.Err = nil
	t.Cleanup(func() { guiSchedules.Items = old; guiSchedules.Err = oldErr; guiScheduledBegin = oldStart })
}
func TestScheduleDueShanghaiAndNoRepeat(t *testing.T) {
	now := time.Date(2026, 11, 3, 0, 0, 30, 0, time.UTC)
	s := GUISchedule{Enabled: true, Frequency: "daily", Time: "08:00"}
	if !scheduleDue(s, now) {
		t.Fatal("Shanghai conversion")
	}
	s.LastRun = "2026-11-03"
	if scheduleDue(s, now) {
		t.Fatal("repeat")
	}
	s.LastRun = ""
	s.Frequency = "weekly"
	s.Weekday = 2
	if !scheduleDue(s, now) {
		t.Fatal("Tuesday missed")
	}
	s.Weekday = 1
	if scheduleDue(s, now) {
		t.Fatal("wrong weekday")
	}
	s.Frequency = "once"
	s.Date = "2026-11-03"
	if !scheduleDue(s, now) {
		t.Fatal("one time missed")
	}
	s.Date = "2026-11-02"
	if scheduleDue(s, now) {
		t.Fatal("late catch up")
	}
}
func TestScheduleTickClaimsBeforeStartingAndSkipsBusy(t *testing.T) {
	schedulesFixture(t)
	now := time.Date(2026, 11, 3, 0, 0, 30, 0, time.UTC)
	guiSchedules.Items = []GUISchedule{{ID: "a", Name: "Morning", Enabled: true, Frequency: "once", Date: "2026-11-03", Time: "08:00", Plan: guiDefaults()}}
	calls := 0
	guiScheduledBegin = func(ctx context.Context, p GUIConfig, persist bool) (string, error) {
		calls++
		if persist {
			t.Fatal("changed manual plan")
		}
		var saved []GUISchedule
		b, _ := os.ReadFile(schedulePath())
		if json.Unmarshal(b, &saved) != nil || saved[0].LastRun != "2026-11-03" {
			t.Fatal("run before durable claim")
		}
		return "", errors.New("busy")
	}
	guiScheduleTick(now)
	guiScheduleTick(now.Add(15 * time.Second))
	if calls != 1 || guiSchedules.Items[0].Enabled || !strings.Contains(guiSchedules.Items[0].Result, "busy") {
		t.Fatal(calls, guiSchedules.Items)
	}
}
func TestScheduleCRUDAndValidation(t *testing.T) {
	schedulesFixture(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/", guiSchedulesList)
	r.POST("/", guiScheduleSave)
	r.POST("/delete", guiScheduleDelete)
	send := func(path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("POST", path, strings.NewReader(body)))
		return w
	}
	s := GUISchedule{Name: "Morning", Frequency: "daily", Time: "08:00", Plan: guiDefaults()}
	b, _ := json.Marshal(s)
	w := send("/", string(b))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	_ = json.Unmarshal(w.Body.Bytes(), &s)
	s.Name = "Edited"
	s.LastRun = "fake"
	b, _ = json.Marshal(s)
	w = send("/", string(b))
	if w.Code != 200 || guiSchedules.Items[0].LastRun != "" {
		t.Fatal("client altered claim")
	}
	s.Time = "99:99"
	b, _ = json.Marshal(s)
	if send("/", string(b)).Code != 400 {
		t.Fatal("invalid time")
	}
	if send("/delete", `{"id":"`+s.ID+`"}`).Code != 200 || len(guiSchedules.Items) != 0 {
		t.Fatal("delete")
	}
}
func TestScheduledPlanRespectsExecutionLock(t *testing.T) {
	schedulesFixture(t)
	scheduler.ExecutionMu.Lock()
	defer scheduler.ExecutionMu.Unlock()
	if _, err := guiBeginPlan(context.Background(), guiDefaults(), false); err == nil || !strings.Contains(err.Error(), "MAA 正在运行") {
		t.Fatal(err)
	}
}
