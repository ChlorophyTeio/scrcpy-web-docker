package server

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"maa-server/scheduler"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func guiCopilotFile(kind string) (string, error) {
	name := "gui.json"
	switch kind {
	case "copilot":
	case "ssscopilot":
		name = "sss.json"
	case "paradoxcopilot":
		name = "paradox.json"
	default:
		return "", errors.New("作业类型无效")
	}
	return filepath.Join(guiBase(), "maa", "copilot", name), nil
}
func guiCopilotDocument(kind string, value map[string]json.RawMessage) error {
	var stage string
	if json.Unmarshal(value["stage_name"], &stage) != nil || strings.TrimSpace(stage) == "" || len(stage) > 240 {
		return errors.New("作业需要有效 stage_name")
	}
	var entries []json.RawMessage
	key := "actions"
	if kind == "ssscopilot" {
		key = "stages"
		var t string
		if json.Unmarshal(value["type"], &t) != nil || t != "SSS" {
			return errors.New("保全作业的 type 必须为 SSS")
		}
	}
	if json.Unmarshal(value[key], &entries) != nil || len(entries) == 0 || len(entries) > 10000 {
		return errors.New("作业需要非空 " + key + " 数组")
	}
	for _, entry := range entries {
		var v map[string]json.RawMessage
		if json.Unmarshal(entry, &v) != nil || v == nil {
			return errors.New("作业步骤必须为对象")
		}
	}
	if kind == "ssscopilot" {
		var tools map[string]int
		if json.Unmarshal(value["tool_men"], &tools) != nil || tools == nil {
			return errors.New("保全作业缺少有效 tool_men 对象")
		}
		for _, count := range tools {
			if count < 0 {
				return errors.New("保全工具人数量不能为负数")
			}
		}
		for _, raw := range entries {
			var stage struct {
				Name       string            `json:"stage_name"`
				Strategies []json.RawMessage `json:"strategies"`
				Draw       *bool             `json:"draw_as_possible"`
			}
			if json.Unmarshal(raw, &stage) != nil || strings.TrimSpace(stage.Name) == "" || stage.Strategies == nil || stage.Draw == nil {
				return errors.New("保全每层需要 stage_name、strategies 和 draw_as_possible")
			}
		}
	}
	if kind != "ssscopilot" {
		var t string
		_ = json.Unmarshal(value["type"], &t)
		if t == "SSS" {
			return errors.New("请切换到保全派驻后导入 SSS 作业")
		}
	}
	return nil
}
func guiCopilot(c *gin.Context) {
	kind := c.DefaultQuery("kind", "copilot")
	file, e := guiCopilotFile(kind)
	if e != nil {
		c.JSON(400, gin.H{"error": e.Error()})
		return
	}
	if c.Request.Method == http.MethodGet {
		b, e := os.ReadFile(file)
		if os.IsNotExist(e) {
			c.JSON(200, gin.H{"present": false, "kind": kind})
			return
		}
		var doc map[string]json.RawMessage
		if e != nil || json.Unmarshal(b, &doc) != nil {
			c.JSON(500, gin.H{"error": "作业读取失败"})
			return
		}
		c.JSON(200, gin.H{"present": true, "kind": kind, "stage_name": doc["stage_name"], "document": doc})
		return
	}
	if !scheduler.ExecutionMu.TryLock() {
		c.JSON(409, gin.H{"error": "任务运行时不能替换或删除作业"})
		return
	}
	defer scheduler.ExecutionMu.Unlock()
	if c.Request.Method == http.MethodDelete {
		if e = os.Remove(file); e != nil && !os.IsNotExist(e) {
			c.JSON(500, gin.H{"error": "删除作业失败"})
			return
		}
		c.JSON(200, gin.H{"deleted": true})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)
	var value map[string]json.RawMessage
	if c.ShouldBindJSON(&value) != nil || value == nil {
		c.JSON(400, gin.H{"error": "作业必须是有效 JSON 对象"})
		return
	}
	if e = guiCopilotDocument(kind, value); e != nil {
		c.JSON(400, gin.H{"error": e.Error()})
		return
	}
	b, _ := json.MarshalIndent(value, "", "  ")
	if os.MkdirAll(filepath.Dir(file), 0755) != nil || os.WriteFile(file+".tmp", b, 0600) != nil || os.Rename(file+".tmp", file) != nil {
		c.JSON(500, gin.H{"error": "作业保存失败"})
		return
	}
	c.JSON(200, gin.H{"saved": true, "kind": kind, "stage_name": value["stage_name"]})
}
