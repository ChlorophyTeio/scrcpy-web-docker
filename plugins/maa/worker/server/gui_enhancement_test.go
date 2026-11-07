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
)

func TestCopilotKindsAndReadOnlyRecruit(t *testing.T) {
	coreTestFixture(t)
	old := config.D.FightDir
	config.D.FightDir = t.TempDir()
	defer func() { config.D.FightDir = old }()
	for _, kind := range []string{"copilot", "ssscopilot", "paradoxcopilot"} {
		file, e := guiCopilotFile(kind)
		if e != nil {
			t.Fatal(e)
		}
		_ = os.MkdirAll(filepath.Dir(file), 0755)
		_ = os.WriteFile(file, []byte(`{}`), 0600)
		value := guiDefaults()
		value.Enabled = []string{kind}
		if e = guiValidate(value); e != nil {
			t.Fatal(e)
		}
		path, e := guiTaskFile(value, kind, "test")
		if e != nil {
			t.Fatal(e)
		}
		b, _ := os.ReadFile(filepath.Join(config.D.FightDir, path+".json"))
		var v struct {
			Tasks []struct {
				Type   string         `json:"type"`
				Params map[string]any `json:"params"`
			} `json:"tasks"`
		}
		_ = json.Unmarshal(b, &v)
		if v.Tasks[0].Type != guiTypes[kind] || v.Tasks[0].Params["filename"] != file {
			t.Fatal(string(b))
		}
	}
	value := guiDefaults()
	value.Params["recruitcalc"] = map[string]any{"times": 10, "select": []int{6}, "confirm": []int{6}, "set_time": true, "expedite": true}
	path, e := guiTaskFile(value, "recruitcalc", "test")
	if e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(filepath.Join(config.D.FightDir, path+".json"))
	var v struct {
		Tasks []struct {
			Params map[string]any `json:"params"`
		} `json:"tasks"`
	}
	_ = json.Unmarshal(b, &v)
	p := v.Tasks[0].Params
	if p["times"] != float64(0) || p["set_time"] != false || p["expedite"] != false || len(p["select"].([]any)) != 0 || len(p["confirm"].([]any)) != 0 {
		t.Fatal("calculation would modify recruitment", string(b))
	}
}
func TestCopilotLibraryValidationAndIsolation(t *testing.T) {
	coreTestFixture(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/copilot", guiCopilot)
	r.GET("/copilot", guiCopilot)
	r.DELETE("/copilot", guiCopilot)
	send := func(method, kind, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, "/copilot?kind="+kind, strings.NewReader(body)))
		return w
	}
	for _, tc := range []struct {
		kind, body string
		code       int
	}{
		{"copilot", `{"stage_name":"1-7","actions":[{"type":"SpeedUp"}]}`, 200},
		{"ssscopilot", `{"type":"SSS","stage_name":"test","tool_men":{},"stages":[{"stage_name":"one","strategies":[],"draw_as_possible":true,"actions":[]}]}`, 200},
		{"paradoxcopilot", `{"stage_name":"char_test","actions":[{"type":"SpeedUp"}]}`, 200},
		{"ssscopilot", `{"stage_name":"bad","actions":[{}]}`, 400},
		{"copilot", `{"type":"SSS","stage_name":"bad","actions":[{}]}`, 400},
		{"copilot", `{"stage_name":"bad","actions":[1]}`, 400},
		{"../outside", `{}`, 400},
	} {
		if w := send("POST", tc.kind, tc.body); w.Code != tc.code {
			t.Fatal(tc.kind, w.Code, w.Body.String())
		}
	}
	if send("DELETE", "ssscopilot", "").Code != 200 {
		t.Fatal("delete")
	}
	if !strings.Contains(send("GET", "copilot", "").Body.String(), `"present":true`) {
		t.Fatal("normal plan overwritten")
	}
	scheduler.ExecutionMu.Lock()
	w := send("POST", "copilot", `{}`)
	scheduler.ExecutionMu.Unlock()
	if w.Code != 409 {
		t.Fatal("import while busy")
	}
}
func TestRecognitionResultsPersistBoundedAndSanitized(t *testing.T) {
	coreTestFixture(t)
	capture := new(guiCapturedOutput)
	ctx := context.WithValue(context.Background(), guiOutputKey{}, capture)
	ctx = context.WithValue(ctx, guiRecognitionTaskKey{}, "recruitcalc")
	for i := 0; i < 510; i++ {
		guiCaptureLine(ctx, "\x1b[32mresult\x1b[0m")
	}
	if len(capture.Lines) != 500 || strings.Contains(capture.Lines[0], "\x1b") {
		t.Fatal("capture limits")
	}
	v := guiDefaults()
	v.Enabled = []string{"recruitcalc"}
	for i := 0; i < 12; i++ {
		guiSaveRecognition(v, capture, "id", true)
	}
	r := guiReadResults()
	if len(r) != 10 || len(r[0].Output) != 500 {
		t.Fatal("not persisted or unbounded")
	}
	info, e := os.Stat(guiResultsPath())
	if e != nil || info.Mode().Perm() != 0600 {
		t.Fatal("result permissions")
	}
}
