package server

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"maa-server/scheduler"
)

func TestMallOperatorFlowHasRealTemplatesAndBoundedAdvance(t *testing.T) {
	b, err := os.ReadFile("../../resources/tasks/web-mall.json")
	if err != nil {
		t.Fatal(err)
	}
	var tasks map[string]map[string]any
	if err := json.Unmarshal(b, &tasks); err != nil {
		t.Fatal(err)
	}
	// A derived MatchTemplate task defaults to its own name, not baseTask.png.
	if tasks["WebMall-OperatorSkip"]["template"] != "RecruitSkip.png" {
		t.Fatal("derived skip points at a nonexistent template")
	}
	finish := tasks["WebMall-OperatorFinish"]
	if finish["baseTask"] == "RecruitFinish" {
		t.Fatal("recruitment-slot button used as result-page recognition")
	}
	if finish["algorithm"] != "JustReturn" || finish["maxTimes"] != float64(10) {
		t.Fatal("result advance must be bounded")
	}
	if tasks["WebMall-ReturnedToStore"]["action"] != "Stop" {
		t.Fatal("return-to-store confirmation must not buy anything")
	}
	next := tasks["WebMall-HireOperator"]["next"].([]any)
	if next[len(next)-1] != "WebMall-OperatorFinish" {
		t.Fatal("unconditional advance must be the last fallback")
	}
}

func TestAwardFailureAndLaterTasksAreReported(t *testing.T) {
	old := guiExecuteCommand
	defer func() { guiExecuteCommand = old }()
	guiExecuteCommand = func(ctx context.Context, name string, args ...string) error {
		if name == "maa" {
			return errors.New("exit status 1")
		}
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	scheduler.ExecutionMu.Lock()
	gui.Lock()
	gui.state = GUIState{Running: true, States: map[string]string{"award": "pending", "update": "pending"}}
	gui.Unlock()
	guiRun(ctx, cancel, GUIConfig{Enabled: []string{"award", "update"}, After: "none"}, map[string]string{"award": "fake/award"})
	gui.Lock()
	defer gui.Unlock()
	if gui.state.States["award"] != "failed" || gui.state.States["update"] != "skipped" {
		t.Fatal(gui.state.States)
	}
	if !strings.Contains(gui.state.TaskResults["award"], "领取奖励") || gui.state.TaskResults["update"] != "前序任务失败，未执行" {
		t.Fatal(gui.state.TaskResults)
	}
}

func TestAwardRecognitionFailureDoesNotUseStaleTask(t *testing.T) {
	failed := `append_task Award {}\n{"cur_task":"AwardBegin","taskchain":"Award"}\nTaskChainError`
	if !strings.Contains(guiAwardLogReason(failed), "未能识别当前界面") {
		t.Fatal("missing actionable error")
	}
	if guiAwardLogReason(failed+"\n"+`{"cur_task":"DailyTask","taskchain":"Award"}`+"\nTaskChainError") != "" {
		t.Fatal("initial state masked a later failure")
	}
	if guiAwardLogReason("append_task Mall {} TaskChainError") != "" {
		t.Fatal("other task leaked")
	}
	if guiAwardLogReason(failed+"\nappend_task Award {} completed") != "" {
		t.Fatal("stale award failure leaked")
	}
}
