package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type GUISchedule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	Frequency string    `json:"frequency"`
	Time      string    `json:"time"`
	Weekday   int       `json:"weekday"`
	Date      string    `json:"date"`
	Plan      GUIConfig `json:"plan"`
	LastRun   string    `json:"last_run"`
	Result    string    `json:"result"`
}

var guiSchedules struct {
	sync.Mutex
	Items []GUISchedule
	Err   error
}
var guiScheduleOnce sync.Once
var guiScheduledBegin = guiBeginPlan

func schedulePath() string { return filepath.Join(guiBase(), "gui-schedules.json") }
func scheduleWrite(items []GUISchedule) error {
	b, e := json.MarshalIndent(items, "", "  ")
	if e != nil {
		return e
	}
	if e = os.WriteFile(schedulePath()+".tmp", b, 0600); e != nil {
		return e
	}
	return os.Rename(schedulePath()+".tmp", schedulePath())
}
func scheduleValid(s *GUISchedule) error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" || len(s.Name) > 120 {
		return errors.New("请填写计划名称（最多 120 字节）")
	}
	t, e := time.Parse("15:04", s.Time)
	if e != nil || t.Format("15:04") != s.Time {
		return errors.New("时间须为 HH:MM")
	}
	switch s.Frequency {
	case "daily":
	case "weekly":
		if s.Weekday < 0 || s.Weekday > 6 {
			return errors.New("星期无效")
		}
	case "once":
		if _, e = time.Parse("2006-01-02", s.Date); e != nil {
			return errors.New("请选择执行日期")
		}
	default:
		return errors.New("不支持的重复方式")
	}
	s.Plan = guiGameDefaults(s.Plan)
	s.Plan.Enabled = guiExecutionOrder(s.Plan.Enabled)
	if len(s.Plan.Enabled) == 0 {
		return errors.New("请先在一键长草中选择任务")
	}
	return guiValidate(s.Plan)
}
func scheduleDue(s GUISchedule, now time.Time) bool {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	n := now.In(loc)
	if !s.Enabled || s.Time != n.Format("15:04") || s.LastRun == n.Format("2006-01-02") {
		return false
	}
	return s.Frequency == "daily" || s.Frequency == "weekly" && s.Weekday == int(n.Weekday()) || s.Frequency == "once" && s.Date == n.Format("2006-01-02")
}
func guiSchedulesList(c *gin.Context) {
	guiSchedules.Lock()
	defer guiSchedules.Unlock()
	if guiSchedules.Err != nil {
		c.JSON(503, gin.H{"error": "定时配置读取失败，请保留文件并检查日志"})
		return
	}
	c.JSON(200, gin.H{"items": guiSchedules.Items, "timezone": "Asia/Shanghai"})
}
func guiScheduleSave(c *gin.Context) {
	var s GUISchedule
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if c.ShouldBindJSON(&s) != nil {
		c.JSON(400, gin.H{"error": "计划格式无效"})
		return
	}
	if e := scheduleValid(&s); e != nil {
		c.JSON(400, gin.H{"error": e.Error()})
		return
	}
	guiSchedules.Lock()
	defer guiSchedules.Unlock()
	if guiSchedules.Err != nil {
		c.JSON(503, gin.H{"error": "定时配置读取失败，未写入"})
		return
	}
	items := append([]GUISchedule{}, guiSchedules.Items...)
	if s.ID == "" {
		if len(items) >= 100 {
			c.JSON(400, gin.H{"error": "计划数量已达上限"})
			return
		}
		var b [12]byte
		if _, e := rand.Read(b[:]); e != nil {
			c.JSON(500, gin.H{"error": "无法创建计划"})
			return
		}
		s.ID = hex.EncodeToString(b[:])
		s.LastRun = ""
		s.Result = ""
		items = append(items, s)
	} else {
		found := false
		for i, old := range items {
			if old.ID == s.ID {
				s.LastRun = old.LastRun
				s.Result = old.Result
				items[i] = s
				found = true
				break
			}
		}
		if !found {
			c.JSON(404, gin.H{"error": "计划不存在"})
			return
		}
	}
	if e := scheduleWrite(items); e != nil {
		c.JSON(500, gin.H{"error": "保存计划失败"})
		return
	}
	guiSchedules.Items = items
	c.JSON(200, s)
}
func guiScheduleDelete(c *gin.Context) {
	var v struct {
		ID string `json:"id"`
	}
	if c.ShouldBindJSON(&v) != nil || v.ID == "" {
		c.JSON(400, gin.H{"error": "请选择计划"})
		return
	}
	guiSchedules.Lock()
	defer guiSchedules.Unlock()
	if guiSchedules.Err != nil {
		c.JSON(503, gin.H{"error": "配置不可用"})
		return
	}
	items := []GUISchedule{}
	found := false
	for _, s := range guiSchedules.Items {
		if s.ID == v.ID {
			found = true
		} else {
			items = append(items, s)
		}
	}
	if !found {
		c.JSON(404, gin.H{"error": "计划不存在"})
		return
	}
	if e := scheduleWrite(items); e != nil {
		c.JSON(500, gin.H{"error": "删除计划失败"})
		return
	}
	guiSchedules.Items = items
	c.JSON(200, gin.H{"deleted": true})
}
func guiScheduleTick(now time.Time) {
	guiSchedules.Lock()
	if guiSchedules.Err != nil {
		guiSchedules.Unlock()
		return
	}
	due := []GUISchedule{}
	items := append([]GUISchedule{}, guiSchedules.Items...)
	loc, _ := time.LoadLocation("Asia/Shanghai")
	for i, s := range items {
		if scheduleDue(s, now) {
			due = append(due, s)
			items[i].LastRun = now.In(loc).Format("2006-01-02")
			items[i].Result = "已触发"
			if s.Frequency == "once" {
				items[i].Enabled = false
			}
		}
	}
	if len(due) == 0 {
		guiSchedules.Unlock()
		return
	}
	// Persist the claim before starting, so a restart cannot repeat the same run.
	if scheduleWrite(items) != nil {
		guiSchedules.Unlock()
		return
	}
	guiSchedules.Items = items
	guiSchedules.Unlock()
	for _, s := range due {
		_, err := guiScheduledBegin(context.Background(), s.Plan, false)
		result := "已提交任务，执行结果见运行日志"
		if err != nil {
			result = "未执行：" + err.Error()
		}
		guiSchedules.Lock()
		items = append([]GUISchedule{}, guiSchedules.Items...)
		for i := range items {
			if items[i].ID == s.ID {
				items[i].Result = result
			}
		}
		if scheduleWrite(items) == nil {
			guiSchedules.Items = items
		}
		guiSchedules.Unlock()
	}
}
func guiInitSchedules() {
	guiScheduleOnce.Do(func() {
		guiSchedules.Lock()
		guiSchedules.Items = []GUISchedule{}
		b, e := os.ReadFile(schedulePath())
		if e == nil {
			guiSchedules.Err = json.Unmarshal(b, &guiSchedules.Items)
		} else if !os.IsNotExist(e) {
			guiSchedules.Err = e
		}
		guiSchedules.Unlock()
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for n := range ticker.C {
				guiScheduleTick(n)
			}
		}()
	})
}
