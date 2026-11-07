package server

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"maa-server/config"
	"maa-server/scheduler"
)

func TestGUIWindowsSettingsReachCoreTask(t *testing.T) {
	before := config.D.FightDir
	config.D.FightDir = t.TempDir()
	defer func() { config.D.FightDir = before }()
	c := guiDefaults()
	c.Order = []string{"recruit", "infrast", "fight"}
	c.Params["fight"]["series"] = -1
	c.Params["fight"]["medicine_expire_days"] = 2
	c.Params["fight"]["DrGrandet"] = true
	c.Params["recruit"]["recruitment_time"] = map[string]int{"3": 460, "4": 540}
	c.Params["infrast"]["use_worldly_plight"] = true
	if err := guiValidate(c); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"fight", "recruit", "infrast"} {
		path, err := guiTaskFile(c, id, "settings")
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(config.D.FightDir, path+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var file struct {
			Tasks []struct {
				Params map[string]any `json:"params"`
			} `json:"tasks"`
		}
		if err := json.Unmarshal(raw, &file); err != nil {
			t.Fatal(err)
		}
		params := file.Tasks[0].Params
		if id == "fight" && (params["medicine_expire_days"] != float64(2) || params["DrGrandet"] != true || params["series"] != float64(-1)) {
			t.Fatal("advanced fight parameters lost")
		}
		if id == "recruit" && params["recruitment_time"].(map[string]any)["3"] != float64(460) {
			t.Fatal("recruitment time lost")
		}
		if id == "infrast" && params["use_worldly_plight"] != true {
			t.Fatal("base team parameter lost")
		}
	}
}

func TestGUIRejectsMalformedSettings(t *testing.T) {
	for _, tc := range []struct {
		id, key string
		value   any
	}{
		{"fight", "medicine_expire_days", -1}, {"fight", "DrGrandet", "false"},
		{"recruit", "recruitment_time", map[string]int{"3": 541}}, {"recruit", "first_tags", "不是列表"},
		{"infrast", "threshold", 30}, {"infrast", "mode", 10000},
		{"rogue", "mode", 2}, {"copilot", "formation_index", 5},
	} {
		c := guiDefaults()
		c.Params[tc.id][tc.key] = tc.value
		if guiValidate(c) == nil {
			t.Fatalf("accepted invalid %s.%s", tc.id, tc.key)
		}
	}
	c := guiDefaults()
	c.Order = []string{"fight", "fight"}
	if guiValidate(c) == nil {
		t.Fatal("duplicate order accepted")
	}
	c = guiDefaults()
	c.TimeoutMinutes = -1
	if guiValidate(c) == nil {
		t.Fatal("negative runtime limit accepted")
	}
	c.TimeoutMinutes = 1441
	if guiValidate(c) == nil {
		t.Fatal("excessive runtime limit accepted")
	}
	c.TimeoutMinutes = 90
	if err := guiValidate(c); err != nil {
		t.Fatal(err)
	}
}

func TestGUIInfrastUploadValidationAndExecutionLock(t *testing.T) {
	before := config.D.InfrastDir
	config.D.InfrastDir = t.TempDir()
	defer func() { config.D.InfrastDir = before }()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/infrast", guiInfrast)
	request := func(body string) int {
		w := httptest.NewRecorder()
		q := httptest.NewRequest("POST", "/infrast", strings.NewReader(body))
		q.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, q)
		return w.Code
	}
	for _, body := range []string{`{"name":"../secret.json","plan":{"plans":[{}]}}`, `{"name":"base.json","plan":{"plans":[]}}`, `{"name":"base.json","plan":{"plans":[null]}}`} {
		if request(body) != 400 {
			t.Fatal("invalid upload accepted")
		}
	}
	valid := `{"name":"base.json","plan":{"plans":[{"name":"day","rooms":{}}]}}`
	scheduler.ExecutionMu.Lock()
	code := request(valid)
	scheduler.ExecutionMu.Unlock()
	if code != 409 {
		t.Fatal("upload during execution accepted")
	}
	if request(valid) != 200 {
		t.Fatal("valid upload rejected")
	}
	if _, err := os.Stat(filepath.Join(config.D.InfrastDir, "base.json")); err != nil {
		t.Fatal(err)
	}
	oldTasks := config.D.FightDir
	config.D.FightDir = t.TempDir()
	defer func() { config.D.FightDir = oldTasks }()
	c := guiDefaults()
	c.Params["infrast"]["mode"] = 10000
	c.Params["infrast"]["filename"] = "missing.json"
	if _, err := guiTaskFile(c, "infrast", "upload"); err == nil {
		t.Fatal("missing base file accepted")
	}
	c.Params["infrast"]["filename"] = "base.json"
	c.Params["infrast"]["plan_index"] = 1
	if _, err := guiTaskFile(c, "infrast", "upload"); err == nil {
		t.Fatal("out-of-range plan accepted")
	}
	c.Params["infrast"]["plan_index"] = 0
	if _, err := guiTaskFile(c, "infrast", "upload"); err != nil {
		t.Fatal(err)
	}
}
