package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"maa-server/scheduler"
)

const coreStableManifest = "https://api.maa.plus/MaaAssistantArknights/api/version/stable.json"

type coreUpdateState struct {
	Enabled       bool   `json:"enabled"`
	Time          string `json:"time"`
	Timezone      string `json:"timezone"`
	Current       string `json:"current"`
	Latest        string `json:"latest"`
	Available     bool   `json:"available"`
	Checking      bool   `json:"checking"`
	CheckedAt     string `json:"checked_at"`
	UpdatedAt     string `json:"updated_at"`
	LastAutoDate  string `json:"last_auto_date"`
	BackupVersion string `json:"backup_version"`
	Message       string `json:"message"`
}

var coreUpdates struct {
	sync.Mutex
	State coreUpdateState
}
var coreSchedulerOnce sync.Once
var coreVersionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

func coreVersionParts(v string) ([3]int, error) {
	var out [3]int
	m := coreVersionPattern.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return out, errors.New("版本号不是稳定版格式")
	}
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return out, err
		}
		out[i] = n
	}
	return out, nil
}
func coreNewer(latest, current string) (bool, error) {
	a, e := coreVersionParts(latest)
	if e != nil {
		return false, e
	}
	b, e := coreVersionParts(current)
	if e != nil {
		return false, e
	}
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] > b[i], nil
		}
	}
	return false, nil
}
func coreSaveLocked() error {
	b, e := json.MarshalIndent(coreUpdates.State, "", "  ")
	if e != nil {
		return e
	}
	p := filepath.Join(guiBase(), "core-update.json")
	if e = os.WriteFile(p+".tmp", b, 0600); e != nil {
		return e
	}
	return os.Rename(p+".tmp", p)
}
func coreState() coreUpdateState {
	coreUpdates.Lock()
	defer coreUpdates.Unlock()
	return coreUpdates.State
}
func coreMessage(text string) {
	coreUpdates.Lock()
	coreUpdates.State.Message = text
	err := coreSaveLocked()
	coreUpdates.Unlock()
	if err != nil {
		log.Error("[MAA 更新] 无法保存更新状态")
	}
	log.Info("[MAA 更新] " + text)
}

// Reuse the existing repository-scoped proxy without publishing it in the UI.
func coreProxy() string {
	if v := os.Getenv("MAA_UPDATE_PROXY"); v != "" {
		return v
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, _ := exec.CommandContext(ctx, "git", "-C", "/root/.local/share/maa/MaaResource", "config", "--get", "http.proxy").Output()
	return strings.TrimSpace(string(out))
}
func coreUpdateEnv() []string {
	env := os.Environ()
	if p := coreProxy(); p != "" {
		env = append(env, "HTTP_PROXY="+p, "HTTPS_PROXY="+p, "http_proxy="+p, "https_proxy="+p)
	}
	return env
}

var coreInstalledVersion = func(ctx context.Context) (string, error) {
	out, e := exec.CommandContext(ctx, "maa", "version", "maa-core").Output()
	if e != nil {
		return "", errors.New("无法读取已安装的 MaaCore 版本")
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return "", errors.New("核心版本为空")
	}
	v := fields[len(fields)-1]
	if _, e := coreVersionParts(v); e != nil {
		return "", e
	}
	return v, nil
}
var coreDataDirectory = func(ctx context.Context) (string, error) {
	out, e := exec.CommandContext(ctx, "maa", "dir", "data").Output()
	if e != nil {
		return "", e
	}
	p := strings.TrimSpace(string(out))
	if !filepath.IsAbs(p) || p == "/" {
		return "", errors.New("核心数据目录无效")
	}
	return p, nil
}
var coreGetManifest = func(ctx context.Context) (string, error) {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if p := coreProxy(); p != "" {
		u, e := url.Parse(p)
		if e != nil {
			return "", errors.New("更新代理配置无效")
		}
		tr.Proxy = http.ProxyURL(u)
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 20 * time.Second}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, coreStableManifest, nil)
	if e != nil {
		return "", e
	}
	req.Header.Set("User-Agent", "MAA-Web-Core-Updater/1")
	r, e := client.Do(req)
	if e != nil {
		return "", errors.New("无法连接官方稳定版服务，请稍后重试")
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return "", fmt.Errorf("官方版本服务返回 HTTP %d", r.StatusCode)
	}
	var m struct {
		Version string `json:"version"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&m) != nil {
		return "", errors.New("官方版本信息格式无效")
	}
	if _, e := coreVersionParts(m.Version); e != nil {
		return "", e
	}
	return m.Version, nil
}

func coreCheck(ctx context.Context) (bool, error) {
	coreUpdates.Lock()
	coreUpdates.State.Checking = true
	coreUpdates.State.Available = false
	coreUpdates.Unlock()
	defer func() {
		coreUpdates.Lock()
		coreUpdates.State.Checking = false
		_ = coreSaveLocked()
		coreUpdates.Unlock()
	}()
	current, err := coreInstalledVersion(ctx)
	if err != nil {
		coreMessage("检查失败：" + err.Error())
		return false, err
	}
	latest, err := coreGetManifest(ctx)
	if err != nil {
		coreMessage("检查失败：" + err.Error())
		return false, err
	}
	available, err := coreNewer(latest, current)
	if err != nil {
		return false, err
	}
	coreUpdates.Lock()
	s := &coreUpdates.State
	s.Current = current
	s.Latest = latest
	s.Available = available
	s.CheckedAt = time.Now().Format(time.RFC3339)
	coreUpdates.Unlock()
	if available {
		coreMessage("发现稳定版 " + latest + "，当前版本 " + current)
	} else {
		coreMessage("当前 " + current + " 已是最新稳定版")
	}
	return available, nil
}

// Install into a fresh data directory: maa update loads the installed library
// before removing it, which can leave FUSE hidden files and fail with ENOTEMPTY.
var coreInstallStage = func(ctx context.Context, stage string) error {
	env := coreUpdateEnv()
	filtered := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, "MAA_DATA_DIR=") {
			filtered = append(filtered, entry)
		}
	}
	filtered = append(filtered, "MAA_DATA_DIR="+stage)
	return guiCommandEnv(ctx, filtered, "maa", "install", "stable", "--batch", "--test-time", "0")
}

func corePerformUpgrade(ctx context.Context) (runErr error) {
	available, err := coreCheck(ctx)
	if err != nil {
		return err
	}
	if !available {
		return nil
	}
	s := coreState()
	data, err := coreDataDirectory(ctx)
	if err != nil {
		return err
	}
	coreMessage("正在备份 " + s.Current + " 核心与基础资源")
	if err := coreCreateBackup(ctx, data, s.Current); err != nil {
		coreMessage("备份失败，已取消升级")
		return errors.New("核心备份失败，未执行升级")
	}
	coreUpdates.Lock()
	coreUpdates.State.BackupVersion = s.Current
	_ = coreSaveLocked()
	coreUpdates.Unlock()
	defer func() {
		if runErr == nil {
			return
		}
		restoreCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		v, e := coreRestoreBackup(restoreCtx, data)
		if e != nil {
			coreMessage("升级失败，自动恢复失败，请保留备份并检查磁盘")
			runErr = errors.New("升级及自动恢复失败")
			return
		}
		coreUpdates.Lock()
		coreUpdates.State.Current = v
		coreUpdates.State.Available = true
		coreUpdates.Unlock()
		coreMessage("升级失败，已自动恢复 " + v)
		runErr = errors.New("升级失败，已自动恢复 " + v)
	}()
	coreMessage("正在升级 MaaCore 稳定版及配套基础资源")
	stage, err := os.MkdirTemp(data, ".web-core-install-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := coreInstallStage(ctx, stage); err != nil {
		return err
	}
	for _, name := range []string{"lib", "resource"} {
		entries, e := os.ReadDir(filepath.Join(stage, name))
		if e != nil || len(entries) == 0 {
			return errors.New("新核心或资源目录为空，已取消替换")
		}
	}
	if err := coreSwapTrees(data, stage); err != nil {
		return err
	}
	v, err := coreInstalledVersion(ctx)
	if err != nil {
		return err
	}
	behind, err := coreNewer(s.Latest, v)
	if err != nil || behind {
		return errors.New("更新后的核心版本验证失败")
	}
	coreUpdates.Lock()
	coreUpdates.State.Current = v
	coreUpdates.State.Available = false
	coreUpdates.State.UpdatedAt = time.Now().Format(time.RFC3339)
	coreUpdates.Unlock()
	coreMessage("MaaCore 已更新至 " + v + "；可回退至 " + s.Current)
	return nil
}

func corePerformRollback(ctx context.Context) error {
	data, e := coreDataDirectory(ctx)
	if e != nil {
		return e
	}
	v, e := coreRestoreBackup(ctx, data)
	if e != nil {
		return e
	}
	installed, e := coreInstalledVersion(ctx)
	if e != nil || installed != v {
		return errors.New("回退后版本验证失败")
	}
	coreUpdates.Lock()
	coreUpdates.State.Current = v
	coreUpdates.State.Available, _ = coreNewer(coreUpdates.State.Latest, v)
	coreUpdates.Unlock()
	coreMessage("已回退至 " + v)
	return nil
}

func coreBegin(kind string) (string, bool) {
	if !scheduler.ExecutionMu.TryLock() {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	id := fmt.Sprint(time.Now().UnixNano())
	gui.Lock()
	gui.state = GUIState{Running: true, ID: id, Current: kind, States: map[string]string{kind: "pending"}, Started: time.Now().Format(time.RFC3339), Message: "正在准备 MaaCore 维护"}
	gui.cancel = cancel
	gui.Unlock()
	go guiRun(ctx, cancel, GUIConfig{Enabled: []string{kind}, After: "none"}, nil)
	return id, true
}
func guiCoreState(c *gin.Context) { c.JSON(200, coreState()) }
func guiCoreSettings(c *gin.Context) {
	var v struct {
		Enabled *bool  `json:"enabled"`
		Time    string `json:"time"`
	}
	if c.ShouldBindJSON(&v) != nil || v.Enabled == nil {
		c.JSON(400, gin.H{"error": "更新设置格式无效"})
		return
	}
	t, e := time.Parse("15:04", v.Time)
	if e != nil || t.Format("15:04") != v.Time {
		c.JSON(400, gin.H{"error": "请填写 HH:MM 格式的更新时间"})
		return
	}
	coreUpdates.Lock()
	old := coreUpdates.State
	coreUpdates.State.Enabled = *v.Enabled
	coreUpdates.State.Time = v.Time
	e = coreSaveLocked()
	if e != nil {
		coreUpdates.State = old
	}
	s := coreUpdates.State
	coreUpdates.Unlock()
	if e != nil {
		c.JSON(500, gin.H{"error": "无法保存自动更新设置"})
		return
	}
	c.JSON(200, s)
}
func guiCoreCheck(c *gin.Context) {
	if !scheduler.ExecutionMu.TryLock() {
		c.JSON(409, gin.H{"error": "MAA 正在运行，请稍后检查版本"})
		return
	}
	defer scheduler.ExecutionMu.Unlock()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()
	if _, e := coreCheck(ctx); e != nil {
		c.JSON(502, gin.H{"error": e.Error()})
		return
	}
	c.JSON(200, coreState())
}
func guiCoreStart(c *gin.Context) {
	id, ok := coreBegin("core_update")
	if !ok {
		c.JSON(409, gin.H{"error": "MAA 正在运行，请稍后升级"})
		return
	}
	c.JSON(202, gin.H{"started": true, "id": id})
}
func guiCoreRollback(c *gin.Context) {
	if coreState().BackupVersion == "" {
		c.JSON(400, gin.H{"error": "暂无可回退备份"})
		return
	}
	id, ok := coreBegin("core_rollback")
	if !ok {
		c.JSON(409, gin.H{"error": "MAA 正在运行，请稍后回退"})
		return
	}
	c.JSON(202, gin.H{"started": true, "id": id})
}

func coreAutoTick(now time.Time) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now = now.In(loc)
	coreUpdates.Lock()
	s := &coreUpdates.State
	day := now.Format("2006-01-02")
	if !s.Enabled || s.Time != now.Format("15:04") || s.LastAutoDate == day {
		coreUpdates.Unlock()
		return
	}
	s.LastAutoDate = day
	if e := coreSaveLocked(); e != nil {
		coreUpdates.Unlock()
		log.Error("[MAA 更新] 无法保存执行日期，取消自动更新")
		return
	}
	coreUpdates.Unlock()
	if _, ok := coreBegin("core_update"); !ok {
		coreMessage("MAA 正在运行，今日定时更新已跳过")
	}
}
func guiInitCoreUpdates() {
	coreSchedulerOnce.Do(func() {
		s := coreUpdateState{Enabled: true, Time: "04:00", Timezone: "Asia/Shanghai", Message: "每天北京时间 04:00 检查稳定版，有新版时自动升级"}
		if b, e := os.ReadFile(filepath.Join(guiBase(), "core-update.json")); e == nil {
			if json.Unmarshal(b, &s) != nil {
				s = coreUpdateState{Enabled: false, Time: "04:00", Timezone: "Asia/Shanghai", Message: "设置读取失败，自动升级已关闭"}
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if v, e := coreInstalledVersion(ctx); e == nil {
			s.Current = v
		}
		cancel()
		s.Checking = false
		s.Timezone = "Asia/Shanghai"
		coreUpdates.Lock()
		coreUpdates.State = s
		_ = coreSaveLocked()
		coreUpdates.Unlock()
		go func() {
			coreAutoTick(time.Now())
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for now := range ticker.C {
				coreAutoTick(now)
			}
		}()
	})
}
