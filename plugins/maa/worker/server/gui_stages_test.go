package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStageCatalogActivitiesAndOpenDays(t *testing.T) {
	oldRoots, oldActivity := guiStageDataRoots, guiActivityPath
	root := t.TempDir()
	guiStageDataRoots = []string{root}
	guiActivityPath = filepath.Join(root, "activities.json")
	defer func() { guiStageDataRoots = oldRoots; guiActivityPath = oldActivity }()
	write := func(name, data string) {
		if e := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); e != nil {
			t.Fatal(e)
		}
	}
	write("item_index.json", `{"30012":{"name":"固源岩"}}`)
	write("stages.json", `[{"code":"1-7","stageId":"main_01-07","apCost":6,"dropInfos":[{"itemId":"30012","dropType":"NORMAL_DROP"}]},{"code":"15-3","stageId":"main_15-02","apCost":18},{"code":"OLD-9","stageId":"act_old","apCost":18}]`)
	write("activities.json", `{"Official":{"sideStoryStage":{"new":{"MinimumRequired":"v6.21.1","Activity":{"StageName":"新活动","UtcStartTime":"2026/11/01 12:00:00","UtcExpireTime":"2026/11/22 03:59:59","TimeZone":8},"Stages":[{"Display":"TL-9","Value":"TL-9","Drop":"30012"}]},"old":{"Activity":{"StageName":"已结束","UtcStartTime":"2025/01/01 12:00:00","UtcExpireTime":"2025/01/22 03:59:59","TimeZone":8},"Stages":[{"Display":"OLD-9","Value":"OLD-9"}]}}}}`)
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	items, _ := guiBuildStages(now, "Bilibili")
	by := map[string]guiStage{}
	for _, s := range items {
		by[s.Code] = s
	}
	if by["1-7"].Detail != "固源岩" || by["1-7"].Cost != 6 || by["TL-9"].Category != "activity" {
		b, _ := json.Marshal(items)
		t.Fatal(string(b))
	}
	if _, ok := by["OLD-9"]; ok {
		t.Fatal("expired activity listed")
	}
	if by["15-3-NORMAL"].Label == "" || by["15-3-HARD"].Label == "" {
		t.Fatal("missing difficulty variants")
	}
	if !by["CE-6"].Open || by["AP-5"].Open {
		t.Fatal("wrong Tuesday resource schedule")
	}
	// Tuesday 03:00 still belongs to Monday's game day.
	items, _ = guiBuildStages(time.Date(2026, 11, 2, 19, 0, 0, 0, time.UTC), "Official")
	for _, s := range items {
		if s.Code == "CE-6" && s.Open {
			t.Fatal("04:00 day boundary ignored")
		}
	}
}
func TestStageNormalizationAndFightFailure(t *testing.T) {
	for input, want := range map[string]string{" ce6 ": "CE-6", "1‑7": "1-7", "annihilation": "Annihilation", "ssreopen-at": "SSReopen-AT", " 15-3-hard ": "15-3-HARD"} {
		if got := guiNormalizeStage(input); got != want {
			t.Fatal(input, got, want)
		}
	}
	if guiFightLogReason(`append_task Fight {} {"to_be_recognized":["Terminal-Entry@SwitchTheme@ReturnTheme"]} TaskChainError`) == "" {
		t.Fatal("missing theme diagnosis")
	}
	if got := guiFightLogReason("append_task Fight {} {\"to_be_recognized\":[\"Terminal-Entry@SwitchTheme@ReturnTheme\"]}\n{\"to_be_recognized\":[\"FightBegin\"]} TaskChainError"); got != "" {
		t.Fatal("successful theme switch misdiagnosed", got)
	}
	if got := guiFightLogReason("append_task Fight {} TaskChainError SwitchTheme@ReturnTheme Terminal-Entry\nappend_task Fight {} TaskChainCompleted"); got != "" {
		t.Fatal("stale failure reused")
	}
}
