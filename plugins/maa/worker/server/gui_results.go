package server

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"maa-server/scheduler"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type guiOutputKey struct{}
type guiRecognitionTaskKey struct{}
type guiCapturedOutput struct {
	sync.Mutex
	Lines []string
	Tasks map[string]bool
}

var guiANSI = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func guiCaptureLine(ctx context.Context, line string) {
	task, _ := ctx.Value(guiRecognitionTaskKey{}).(string)
	if task != "recruitcalc" && task != "depot" && task != "operbox" && task != "update" {
		return
	}
	v, _ := ctx.Value(guiOutputKey{}).(*guiCapturedOutput)
	if v == nil {
		return
	}
	v.Lock()
	defer v.Unlock()
	if v.Tasks == nil {
		v.Tasks = map[string]bool{}
	}
	v.Tasks[task] = true
	line = guiANSI.ReplaceAllString(line, "")
	if len(line) > 2048 {
		line = line[:2048]
	}
	v.Lines = append(v.Lines, line)
	if len(v.Lines) > 500 {
		v.Lines = v.Lines[len(v.Lines)-500:]
	}
}

type guiRecognitionResult struct {
	ID      string   `json:"id"`
	Time    string   `json:"time"`
	Tasks   []string `json:"tasks"`
	Success bool     `json:"success"`
	Output  []string `json:"output"`
}

var guiResultsMu sync.Mutex

func guiResultsPath() string { return filepath.Join(guiBase(), "recognition-results.json") }
func guiReadResults() []guiRecognitionResult {
	v := []guiRecognitionResult{}
	b, _ := os.ReadFile(guiResultsPath())
	_ = json.Unmarshal(b, &v)
	return v
}
func guiSaveRecognition(v GUIConfig, output *guiCapturedOutput, id string, success bool) {
	found := false
	for _, task := range v.Enabled {
		if task == "recruitcalc" || task == "depot" || task == "operbox" || task == "update" {
			found = true
		}
	}
	if !found {
		return
	}
	output.Lock()
	lines := append([]string{}, output.Lines...)
	recognized := []string{}
	for _, task := range v.Enabled {
		if output.Tasks[task] {
			recognized = append(recognized, task)
		}
	}
	output.Unlock()
	if len(lines) == 0 {
		return
	}
	guiResultsMu.Lock()
	defer guiResultsMu.Unlock()
	items := append([]guiRecognitionResult{{ID: id, Time: time.Now().Format(time.RFC3339), Tasks: recognized, Success: success, Output: lines}}, guiReadResults()...)
	if len(items) > 10 {
		items = items[:10]
	}
	b, _ := json.Marshal(items)
	if os.WriteFile(guiResultsPath()+".tmp", b, 0600) == nil {
		_ = os.Rename(guiResultsPath()+".tmp", guiResultsPath())
	}
}
func guiResults(c *gin.Context) {
	if c.Request.Method == "DELETE" {
		if !scheduler.ExecutionMu.TryLock() {
			c.JSON(409, gin.H{"error": "任务运行时不能清空结果"})
			return
		}
		defer scheduler.ExecutionMu.Unlock()
		guiResultsMu.Lock()
		defer guiResultsMu.Unlock()
		if e := os.Remove(guiResultsPath()); e != nil && !os.IsNotExist(e) {
			c.JSON(500, gin.H{"error": "清空失败"})
			return
		}
		c.JSON(200, gin.H{"deleted": true})
		return
	}
	guiResultsMu.Lock()
	defer guiResultsMu.Unlock()
	items := guiReadResults()
	query := strings.TrimSpace(c.Query("task"))
	if query != "" {
		selected := []guiRecognitionResult{}
		for _, r := range items {
			for _, task := range r.Tasks {
				if task == query {
					selected = append(selected, r)
					break
				}
			}
		}
		items = selected
	}
	c.JSON(200, gin.H{"items": items})
}
