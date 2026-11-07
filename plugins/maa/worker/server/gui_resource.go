package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"maa-server/scheduler"
)

func guiResourceUpdate(c *gin.Context) {
	if !scheduler.ExecutionMu.TryLock() {
		c.JSON(409, gin.H{"error": "MAA 正在运行，请先结束当前任务再更新资源"})
		return
	}
	// A maintenance operation never saves/replaces the user's daily task plan.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	id := fmt.Sprint(time.Now().UnixNano())
	gui.Lock()
	gui.state = GUIState{Running: true, ID: id, Current: "resource_update", States: map[string]string{"resource_update": "pending"}, Started: time.Now().Format(time.RFC3339), Message: "正在准备资源热更新"}
	gui.cancel = cancel
	gui.Unlock()
	go guiRun(ctx, cancel, GUIConfig{Enabled: []string{"resource_update"}, After: "none"}, nil)
	c.JSON(202, gin.H{"started": true, "id": id})
}
