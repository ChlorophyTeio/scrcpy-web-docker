package server

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"maa-server/config"
)

var guiPackagePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)+$`)

func guiHasTask(c GUIConfig, id string) bool {
	for _, v := range c.Enabled {
		if v == id {
			return true
		}
	}
	return false
}
func guiGamePackage(c GUIConfig) string {
	p, _ := c.Params["launch"]["package"].(string)
	if p == "" {
		if c.Client == "Bilibili" {
			return "com.hypergryph.arknights.bilibili"
		}
		return "com.hypergryph.arknights"
	}
	return p
}
func guiGameDefaults(c GUIConfig) GUIConfig {
	if c.Params != nil {
		if stage, ok := c.Params["fight"]["stage"].(string); ok {
			c.Params["fight"]["stage"] = guiNormalizeStage(stage)
		}
	}
	if c.Params == nil {
		c.Params = map[string]map[string]any{}
	}
	if c.Params["launch"] == nil {
		c.Params["launch"] = map[string]any{"package": guiGamePackage(c)}
		if guiHasTask(c, "startup") && c.Params["startup"]["start_game_enabled"] == true && !guiHasTask(c, "launch") {
			c.Enabled = append([]string{"launch"}, c.Enabled...)
		}
	}
	if c.Params["update"] == nil {
		c.Params["update"] = map[string]any{}
	}
	for k, v := range map[string]any{"operbox": true, "depot": true, "interval": "every"} {
		if _, ok := c.Params["update"][k]; !ok {
			c.Params["update"][k] = v
		}
	}
	return c
}
func guiExecutionOrder(ids []string) []string {
	out := []string{}
	for _, id := range ids {
		if id == "launch" {
			out = append(out, id)
		}
	}
	for _, id := range ids {
		if id != "launch" && id != "update" {
			out = append(out, id)
		}
	}
	for _, id := range ids {
		if id == "update" {
			out = append(out, id)
		}
	}
	return out
}
func guiLaunchGame(ctx context.Context, c GUIConfig) error {
	return guiExecuteCommand(ctx, "/opt/android/adb", "-s", config.Profiles.Connection.Device, "shell", "monkey", "-p", guiGamePackage(c), "-c", "android.intent.category.LAUNCHER", "1")
}

type guiUserDataState struct {
	OperBox string `json:"operbox"`
	Depot   string `json:"depot"`
}

func guiReadUserDataState() guiUserDataState {
	var s guiUserDataState
	b, e := os.ReadFile(filepath.Join(guiBase(), "user-data-state.json"))
	if e == nil {
		_ = json.Unmarshal(b, &s)
	}
	return s
}
func guiUserDataDay(t time.Time) time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return t.In(loc).Add(-4 * time.Hour)
}
func guiUserDataTriggerDue(last, interval string, now time.Time) bool {
	if interval == "every" || last == "" {
		return true
	}
	t, e := time.Parse(time.RFC3339, last)
	if e != nil {
		return true
	}
	a, b := guiUserDataDay(now), guiUserDataDay(t)
	if interval == "daily" {
		return a.Format("2006-01-02") != b.Format("2006-01-02")
	}
	ay, aw := a.ISOWeek()
	by, bw := b.ISOWeek()
	return ay != by || aw != bw
}
func guiUserDataDue(c GUIConfig, now time.Time) (GUIConfig, bool) {
	p := map[string]any{}
	for k, v := range c.Params["update"] {
		p[k] = v
	}
	interval, _ := p["interval"].(string)
	s := guiReadUserDataState()
	p["operbox"] = p["operbox"] == true && guiUserDataTriggerDue(s.OperBox, interval, now)
	p["depot"] = p["depot"] == true && guiUserDataTriggerDue(s.Depot, interval, now)
	out := c
	out.Params = map[string]map[string]any{}
	for k, v := range c.Params {
		out.Params[k] = v
	}
	out.Params["update"] = p
	return out, p["operbox"] == true || p["depot"] == true
}
func guiUserDataCompleted(c GUIConfig, now time.Time) error {
	s := guiReadUserDataState()
	if c.Params["update"]["operbox"] == true {
		s.OperBox = now.Format(time.RFC3339)
	}
	if c.Params["update"]["depot"] == true {
		s.Depot = now.Format(time.RFC3339)
	}
	b, e := json.Marshal(s)
	if e != nil {
		return e
	}
	p := filepath.Join(guiBase(), "user-data-state.json")
	if os.WriteFile(p+".tmp", b, 0600) != nil || os.Rename(p+".tmp", p) != nil {
		return errors.New("识别完成，但无法保存数据更新时间")
	}
	return nil
}
