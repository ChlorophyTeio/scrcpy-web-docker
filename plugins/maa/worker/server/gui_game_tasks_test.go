package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"maa-server/config"
)

func TestLaunchPackageValidationAndArgumentIsolation(t *testing.T) {
	c := guiDefaults()
	for _, pkg := range []string{"com.hypergryph.arknights", "com.example.custom_game"} {
		c.Params["launch"]["package"] = pkg
		if e := guiValidate(c); e != nil {
			t.Fatal(e)
		}
	}
	for _, pkg := range []string{"com.game;reboot", "com.game foo", "$(reboot)", "", "com/game"} {
		c.Params["launch"]["package"] = pkg
		if guiValidate(c) == nil {
			t.Fatal("invalid package accepted", pkg)
		}
	}
	c.Params["launch"]["package"] = "com.example.custom_game"
	old := guiExecuteCommand
	defer func() { guiExecuteCommand = old }()
	var got []string
	guiExecuteCommand = func(ctx context.Context, name string, args ...string) error {
		got = append([]string{name}, args...)
		return nil
	}
	if e := guiLaunchGame(context.Background(), c); e != nil {
		t.Fatal(e)
	}
	expected := []string{"/opt/android/adb", "-s", config.Profiles.Connection.Device, "shell", "monkey", "-p", "com.example.custom_game", "-c", "android.intent.category.LAUNCHER", "1"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatal(got)
	}
}
func TestUserDataTaskGeneratesRecognitionInsteadOfCoreUpgrade(t *testing.T) {
	old := config.D.FightDir
	config.D.FightDir = t.TempDir()
	defer func() { config.D.FightDir = old }()
	c := guiDefaults()
	path, e := guiTaskFile(c, "update", "test")
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(config.D.FightDir, path+".json"))
	if e != nil {
		t.Fatal(e)
	}
	var file struct {
		Tasks []struct {
			Type string `json:"type"`
		} `json:"tasks"`
	}
	if json.Unmarshal(b, &file) != nil {
		t.Fatal("invalid JSON")
	}
	if len(file.Tasks) != 2 || file.Tasks[0].Type != "OperBox" || file.Tasks[1].Type != "Depot" {
		t.Fatal(string(b))
	}
	if strings.Contains(string(b), "update stable") {
		t.Fatal("recognition mapped to core upgrade")
	}
	c.Params["update"]["operbox"] = false
	path, e = guiTaskFile(c, "update", "test")
	if e != nil {
		t.Fatal(e)
	}
	b, _ = os.ReadFile(filepath.Join(config.D.FightDir, path+".json"))
	_ = json.Unmarshal(b, &file)
	if len(file.Tasks) != 1 || file.Tasks[0].Type != "Depot" {
		t.Fatal(string(b))
	}
}
func TestLaunchTaskIsFirstUserDataLastAndStartupDoesNotRelaunch(t *testing.T) {
	ids := []string{"update", "fight", "startup", "launch"}
	if !reflect.DeepEqual(guiExecutionOrder(ids), []string{"launch", "fight", "startup", "update"}) {
		t.Fatal("invalid execution boundaries")
	}
	c := guiDefaults()
	old := config.D.FightDir
	config.D.FightDir = t.TempDir()
	defer func() { config.D.FightDir = old }()
	path, e := guiTaskFile(c, "startup", "test")
	if e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(filepath.Join(config.D.FightDir, path+".json"))
	if !strings.Contains(string(b), `"start_game_enabled": false`) {
		t.Fatal("startup launched the game twice")
	}
}
func TestUserDataIntervalUsesFourAMGameDay(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	last := time.Date(2026, 10, 26, 4, 1, 0, 0, loc)
	if guiUserDataTriggerDue(last.Format(time.RFC3339), "daily", last.Add(23*time.Hour)) {
		t.Fatal("daily recognition repeated before reset")
	}
	if !guiUserDataTriggerDue(last.Format(time.RFC3339), "daily", last.Add(24*time.Hour)) {
		t.Fatal("daily reset missed")
	}
	if guiUserDataTriggerDue(last.Format(time.RFC3339), "weekly", last.Add(6*24*time.Hour)) {
		t.Fatal("weekly recognition repeated")
	}
	if !guiUserDataTriggerDue(last.Format(time.RFC3339), "weekly", last.Add(7*24*time.Hour)) {
		t.Fatal("weekly reset missed")
	}
}
func TestLegacySettingsGainLaunchWithoutChangingOtherParameters(t *testing.T) {
	c := guiDefaults()
	delete(c.Params, "launch")
	delete(c.Params, "update")
	c.Enabled = []string{"update", "startup", "fight"}
	c.Params["fight"]["stage"] = "CE-6"
	c = guiGameDefaults(c)
	if !guiHasTask(c, "launch") || c.Params["fight"]["stage"] != "CE-6" || c.Params["update"]["depot"] != true {
		t.Fatal(c)
	}
	c = guiGameDefaults(c)
	n := 0
	for _, id := range c.Enabled {
		if id == "launch" {
			n++
		}
	}
	if n != 1 {
		t.Fatal("launch inserted twice")
	}
}
