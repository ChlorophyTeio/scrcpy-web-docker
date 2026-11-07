package server

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"maa-server/scheduler"
	"maa-server/utils"
	"net/http"
	"os"
	"path/filepath"
)

func guiPerformance(c *gin.Context) {
	h := utils.InferenceCPUInfo()
	if c.Request.Method == http.MethodPost {
		if !scheduler.ExecutionMu.TryLock() {
			c.JSON(409, gin.H{"error": "任务运行时不能切换识别模式"})
			return
		}
		defer scheduler.ExecutionMu.Unlock()
		var v struct {
			Mode string `json:"mode"`
		}
		if c.ShouldBindJSON(&v) != nil || (v.Mode != "balanced" && v.Mode != "performance") {
			c.JSON(400, gin.H{"error": "请选择默认调度或高性能核心优先"})
			return
		}
		if v.Mode == "performance" && len(h.Performance) == 0 {
			c.JSON(400, gin.H{"error": "当前环境无法可靠区分高性能核心，保留默认调度"})
			return
		}
		b, _ := json.Marshal(v)
		if os.WriteFile(utils.InferencePath()+".tmp", b, 0600) != nil || os.Rename(utils.InferencePath()+".tmp", utils.InferencePath()) != nil {
			c.JSON(500, gin.H{"error": "保存识别配置失败"})
			return
		}
	}
	var benchmark any
	b, e := os.ReadFile(filepath.Join(guiBase(), "inference-benchmark.json"))
	if e == nil {
		_ = json.Unmarshal(b, &benchmark)
	}
	effective := h.Allowed
	mode := utils.InferenceMode()
	if mode == "performance" && len(h.Performance) > 0 {
		effective = h.Performance
	}
	c.JSON(200, gin.H{"cpu_model": h.Model, "allowed_cpus": h.Allowed, "p_cores": h.Performance, "mode": mode, "effective_cpus": effective, "benchmark": benchmark})
}
