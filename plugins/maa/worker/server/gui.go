package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"maa-server/config"
	"maa-server/scheduler"
	"maa-server/utils"
)

type GUIConfig struct {
	Enabled        []string                  `json:"enabled"`
	Order          []string                  `json:"order,omitempty"`
	Params         map[string]map[string]any `json:"params"`
	Client         string                    `json:"client"`
	After          string                    `json:"after"`
	TimeoutMinutes int                       `json:"timeout_minutes,omitempty"`
}
type GUIState struct {
	Running     bool              `json:"running"`
	Stopping    bool              `json:"stopping"`
	ID          string            `json:"id"`
	Current     string            `json:"current"`
	States      map[string]string `json:"states"`
	Message     string            `json:"message"`
	Started     string            `json:"started"`
	Finished    string            `json:"finished"`
	Warnings    []string          `json:"warnings,omitempty"`
	TaskResults map[string]string `json:"task_results,omitempty"`
}

var gui struct {
	sync.Mutex
	state  GUIState
	cancel context.CancelFunc
}
var guiSettingsMu sync.Mutex
var guiTypes = map[string]string{"launch": "LaunchGame", "startup": "StartUp", "fight": "Fight", "infrast": "Infrast", "recruit": "Recruit", "mall": "Mall", "award": "Award", "rogue": "Roguelike", "reclamation": "Reclamation", "copilot": "Copilot", "ssscopilot": "SSSCopilot", "paradoxcopilot": "ParadoxCopilot", "recruitcalc": "Recruit", "update": "update", "depot": "Depot", "operbox": "OperBox"}
var guiNames = map[string]string{"core_update": "核心版本更新", "core_rollback": "核心版本回退", "resource_update": "资源热更新", "launch": "启动游戏", "startup": "开始唤醒", "fight": "理智作战", "infrast": "基建换班", "recruit": "自动公招", "mall": "信用收支", "award": "领取奖励", "rogue": "自动肉鸽", "reclamation": "生息演算", "copilot": "自动战斗", "ssscopilot": "保全派驻作业", "paradoxcopilot": "悖论模拟作业", "recruitcalc": "公招计算", "update": "更新数据", "depot": "仓库识别", "operbox": "干员识别"}

func guiBase() string { return filepath.Join(config.D.HomeDir, "config") }
func guiDefaults() GUIConfig {
	return GUIConfig{Enabled: []string{"launch", "startup", "fight", "infrast", "recruit", "mall", "award"}, Client: "Official", After: "none", Params: map[string]map[string]any{
		"launch": {"package": "com.hypergryph.arknights"}, "update": {"operbox": true, "depot": true, "interval": "every"}, "startup": {"start_game_enabled": true}, "fight": {"stage": "1-7", "medicine": 0, "stone": 0, "times": 2147483647, "series": 0},
		"infrast": {"mode": 0, "facility": []string{"Mfg", "Trade", "Power", "Control", "Reception", "Office", "Dorm"}, "drones": "_NotUse", "threshold": 0.3, "dorm_trust_enabled": true},
		"recruit": {"select": []int{4, 5}, "confirm": []int{3, 4, 5}, "times": 4, "refresh": true, "expedite": false},
		"mall":    {"visit_friends": true, "shopping": true, "buy_first": []string{}, "blacklist": []string{}}, "award": {"award": true, "mail": false},
		"rogue": {"theme": "Phantom", "mode": 0, "starts_count": 1}, "reclamation": {"theme": "Tales", "mode": 0}, "copilot": {"loop_times": 1, "formation": false, "use_sanity_potion": false},
	}}
}
func guiValidate(c GUIConfig) error {
	if err := guiValidateSettings(c); err != nil {
		return err
	}
	if c.Client != "Official" && c.Client != "Bilibili" {
		return errors.New("请选择官服或 B 服")
	}
	if c.After != "none" && c.After != "close_game" {
		return errors.New("完成后操作无效")
	}
	if len(c.Enabled) > len(guiTypes) {
		return errors.New("任务数量过多")
	}
	seen := map[string]bool{}
	for _, id := range c.Enabled {
		if _, ok := guiTypes[id]; !ok || seen[id] {
			return errors.New("任务类型无效或重复")
		}
		seen[id] = true
	}
	for id, p := range c.Params {
		if _, ok := guiTypes[id]; !ok {
			return errors.New("参数任务类型无效")
		}
		// User-controlled filenames must never escape the designated configuration directory.
		if f, ok := p["filename"].(string); ok && f != "" {
			if id != "infrast" || filepath.Base(f) != f || strings.ContainsAny(f, "/\\") || !strings.HasSuffix(f, ".json") {
				return errors.New("配置文件名无效")
			}
		}
	}
	p := c.Params["fight"]
	for _, key := range []string{"medicine", "stone", "times", "series"} {
		if v, ok := p[key]; ok {
			n, ok := v.(float64)
			if !ok {
				switch x := v.(type) {
				case int:
					n = float64(x)
					ok = true
				}
			}
			min := float64(0)
			if key == "series" {
				min = -1
			}
			if !ok || n != float64(int64(n)) || n < min || n > 2147483647 || (key == "series" && n > 10) {
				return fmt.Errorf("作战参数 %s 无效", key)
			}
		}
	}
	if mode, ok := c.Params["infrast"]["mode"].(float64); ok && mode == 10000 {
		if name, ok := c.Params["infrast"]["filename"].(string); !ok || name == "" {
			return errors.New("自定义基建需要排班文件")
		}
	}
	return nil
}
func guiReadConfig() GUIConfig {
	c := guiDefaults()
	b, err := os.ReadFile(filepath.Join(guiBase(), "gui-web.json"))
	if err == nil {
		var saved GUIConfig
		if json.Unmarshal(b, &saved) == nil {
			saved = guiGameDefaults(saved)
			if guiValidate(saved) == nil {
				return saved
			}
		}
	}
	return c
}
func guiWriteConfig(c GUIConfig) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(guiBase(), "gui-web.json")
	if err = os.WriteFile(path+".tmp", b, 0600); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}
func guiConfigHandler(c *gin.Context) {
	guiSettingsMu.Lock()
	defer guiSettingsMu.Unlock()
	if c.Request.Method == http.MethodGet {
		c.JSON(200, guiReadConfig())
		return
	}
	var value GUIConfig
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if c.ShouldBindJSON(&value) != nil {
		c.JSON(400, gin.H{"error": "参数格式无效"})
		return
	}
	value = guiGameDefaults(value)
	if err := guiValidate(value); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	gui.Lock()
	running := gui.state.Running
	gui.Unlock()
	if running {
		c.JSON(409, gin.H{"error": "运行时不能修改配置"})
		return
	}
	if err := guiWriteConfig(value); err != nil {
		c.JSON(500, gin.H{"error": "保存配置失败"})
		return
	}
	c.JSON(200, gin.H{"saved": true})
}
func guiStatus(c *gin.Context) { gui.Lock(); defer gui.Unlock(); c.JSON(200, gui.state) }
func guiCommand(ctx context.Context, name string, args ...string) error {
	var env []string
	if name == "maa" && len(args) > 0 && (args[0] == "update" || args[0] == "install") {
		env = coreUpdateEnv()
	}
	return guiCommandEnv(ctx, env, name, args...)
}
func guiCommandEnv(ctx context.Context, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if name == "maa" {
		cmd = utils.MaaCommand(ctx, args...)
	}
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 3 * time.Second
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	var readers sync.WaitGroup
	readers.Add(2)
	read := func(scanner *bufio.Scanner) {
		defer readers.Done()
		scanner.Buffer(make([]byte, 4096), 1<<20)
		for scanner.Scan() {
			guiCaptureLine(ctx, scanner.Text())
			log.Info("[GUI] " + scanner.Text())
		}
	}
	go read(bufio.NewScanner(stdout))
	go read(bufio.NewScanner(stderr))
	readers.Wait()
	return cmd.Wait()
}
func guiTaskFile(c GUIConfig, id string, folder string) (string, error) {
	if id == "launch" {
		return "", nil
	}
	p := map[string]any{}
	for k, v := range c.Params[id] {
		p[k] = v
	}
	if id == "startup" && guiHasTask(c, "launch") {
		p["start_game_enabled"] = false
	}
	if id == "startup" || id == "fight" {
		p["client_type"] = c.Client
	}
	if id == "infrast" {
		if name, ok := p["filename"].(string); ok && name != "" {
			p["filename"] = filepath.Join(config.D.InfrastDir, name)
		}
		if _, ok := p["plan_index"]; !ok {
			p["plan_index"] = 0
		}
		if mode, _ := guiNumber(p["mode"]); mode == 10000 {
			name, _ := p["filename"].(string)
			raw, err := os.ReadFile(name)
			var file struct {
				Plans []json.RawMessage `json:"plans"`
			}
			if err != nil || json.Unmarshal(raw, &file) != nil || len(file.Plans) == 0 {
				return "", errors.New("请先上传有效的基建排班 JSON")
			}
			index, ok := guiNumber(p["plan_index"])
			if !ok || index < 0 || index >= float64(len(file.Plans)) {
				return "", errors.New("排班方案编号超出文件范围（从 0 开始）")
			}
		}
	}
	if id == "copilot" || id == "ssscopilot" || id == "paradoxcopilot" {
		f, _ := guiCopilotFile(id)
		if _, err := os.Stat(f); err != nil {
			return "", errors.New("请先导入作业 JSON")
		}
		p["filename"] = f
	}
	if id == "recruitcalc" {
		p = map[string]any{"times": 0, "select": []int{}, "confirm": []int{}, "set_time": false, "refresh": false, "expedite": false, "report_to_penguin": false, "report_to_yituliu": false}
	}
	task := map[string]any{"tasks": []any{map[string]any{"name": guiNames[id], "type": guiTypes[id], "params": p}}}
	if id == "update" {
		parts := []any{}
		if p["operbox"] == true {
			parts = append(parts, map[string]any{"name": "更新干员数据", "type": "OperBox", "params": map[string]any{}})
		}
		if p["depot"] == true {
			parts = append(parts, map[string]any{"name": "更新仓库数据", "type": "Depot", "params": map[string]any{}})
		}
		if len(parts) == 0 {
			return "", errors.New("更新数据至少选择干员或仓库")
		}
		task["tasks"] = parts
	}
	b, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return "", err
	}
	dir := filepath.Join(config.D.FightDir, folder)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(dir, id+".json"), b, 0600); err != nil {
		return "", err
	}
	return folder + "/" + id, nil
}
func guiConnection(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
	defer cancel()
	serial := config.Profiles.Connection.Device
	state, err := exec.CommandContext(ctx, "/opt/android/adb", "-s", serial, "get-state").Output()
	connected := err == nil && strings.TrimSpace(string(state)) == "device"
	packageName := "com.hypergryph.arknights"
	if c.Query("client") == "Bilibili" {
		packageName = "com.hypergryph.arknights.bilibili"
	}
	game := false
	if connected {
		out, _ := exec.CommandContext(ctx, "/opt/android/adb", "-s", serial, "shell", "pm", "path", packageName).Output()
		game = strings.Contains(string(out), "package:")
	}
	version, _ := exec.CommandContext(ctx, "maa", "version").Output()
	c.JSON(200, gin.H{"device": serial, "connected": connected, "game_installed": game, "version": strings.TrimSpace(string(version)), "adb": "共享 USB ADB"})
}
func guiCatalog(c *gin.Context) {
	var items map[string]json.RawMessage
	b, err := os.ReadFile("/root/.local/share/maa/MaaResource/resource/item_index.json")
	if err != nil || json.Unmarshal(b, &items) != nil {
		c.JSON(200, []any{})
		return
	}
	out := []map[string]string{}
	for id, b := range items {
		var v struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(b, &v) == nil && v.Name != "" {
			out = append(out, map[string]string{"id": id, "name": v.Name})
		}
	}
	c.JSON(200, out)
}
func guiValidatePlan(c *gin.Context) {
	var value GUIConfig
	if c.ShouldBindJSON(&value) != nil {
		c.JSON(400, gin.H{"error": "参数格式无效"})
		return
	}
	value = guiGameDefaults(value)
	if err := guiValidate(value); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if !scheduler.ExecutionMu.TryLock() {
		c.JSON(409, gin.H{"error": "MAA 正在运行，请稍候"})
		return
	}
	defer scheduler.ExecutionMu.Unlock()
	folder := "gui-check-" + fmt.Sprint(time.Now().UnixNano())
	defer os.RemoveAll(filepath.Join(config.D.FightDir, folder))
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	for _, id := range value.Enabled {
		if id == "launch" {
			continue
		}
		path, err := guiTaskFile(value, id, folder)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		cmd := exec.CommandContext(ctx, "maa", "run", path, "--dry-run", "--batch")
		out, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(400, gin.H{"error": guiNames[id] + "配置检查失败", "details": string(out)})
			return
		}
	}
	c.JSON(200, gin.H{"valid": true, "message": "配置检查通过；未连接或操作游戏"})
}
func guiStart(c *gin.Context) {
	var value GUIConfig
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	if c.ShouldBindJSON(&value) != nil {
		c.JSON(400, gin.H{"error": "参数格式无效"})
		return
	}
	id, err := guiBeginPlan(c.Request.Context(), value, true)
	if err != nil {
		code := 400
		if strings.Contains(err.Error(), "MAA 正在运行") {
			code = 409
		}
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}
	c.JSON(202, gin.H{"started": true, "id": id})
}

func guiBeginPlan(parent context.Context, value GUIConfig, persist bool) (string, error) {
	value = guiGameDefaults(value)
	value.Enabled = guiExecutionOrder(value.Enabled)
	if err := guiValidate(value); err != nil {
		return "", err
	}
	if len(value.Enabled) == 0 {
		return "", errors.New("请至少选择一项任务")
	}
	if !scheduler.ExecutionMu.TryLock() {
		return "", errors.New("MAA 正在运行，请先停止当前任务")
	}
	fail := func(message string) (string, error) { scheduler.ExecutionMu.Unlock(); return "", errors.New(message) }
	paths := map[string]string{}
	for _, id := range value.Enabled {
		if id != "launch" {
			path, err := guiTaskFile(value, id, "gui-run")
			if err != nil {
				return fail(err.Error())
			}
			paths[id] = path
		}
	}
	if len(paths) > 0 || guiHasTask(value, "launch") {
		ctx, cancel := context.WithTimeout(parent, 8*time.Second)
		defer cancel()
		serial := config.Profiles.Connection.Device
		out, err := exec.CommandContext(ctx, "/opt/android/adb", "-s", serial, "get-state").Output()
		if err != nil || strings.TrimSpace(string(out)) != "device" {
			return fail("手机未连接，请检查 USB 和 ADB 授权")
		}
		out, _ = exec.CommandContext(ctx, "/opt/android/adb", "-s", serial, "shell", "dumpsys", "trust").Output()
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "(current):") && strings.Contains(line, "deviceLocked=1") {
				return fail("手机已锁屏，请先在远控界面解锁")
			}
		}
		if guiHasTask(value, "launch") {
			out, _ = exec.CommandContext(ctx, "/opt/android/adb", "-s", serial, "shell", "pm", "path", guiGamePackage(value)).Output()
			if !bytes.Contains(out, []byte("package:")) {
				return fail("启动游戏的包名未安装，请检查自填包名")
			}
		}
		if len(paths) > 0 {
			pkg := "com.hypergryph.arknights"
			if value.Client == "Bilibili" {
				pkg = "com.hypergryph.arknights.bilibili"
			}
			out, _ = exec.CommandContext(ctx, "/opt/android/adb", "-s", serial, "shell", "pm", "path", pkg).Output()
			if !bytes.Contains(out, []byte("package:")) {
				return fail("未检测到所选明日方舟客户端，请先安装并登录游戏")
			}
		}
	}
	guiSettingsMu.Lock()
	saved := value
	for _, task := range value.Enabled {
		if task == "copilot" || task == "ssscopilot" || task == "paradoxcopilot" || task == "recruitcalc" || task == "depot" || task == "operbox" {
			saved.Enabled = guiReadConfig().Enabled
			break
		}
	}
	var err error
	if persist {
		err = guiWriteConfig(saved)
	}
	guiSettingsMu.Unlock()
	if err != nil {
		return fail("保存配置失败")
	}
	scheduler.ScheduleData.IsStop = true
	ctx, cancel := context.WithCancel(context.Background())
	if value.TimeoutMinutes > 0 {
		cancel()
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(value.TimeoutMinutes)*time.Minute)
	}
	id := fmt.Sprint(time.Now().UnixNano())
	gui.Lock()
	gui.cancel = cancel
	gui.state = GUIState{Running: true, ID: id, States: map[string]string{}, Started: time.Now().Format(time.RFC3339), Message: "正在准备任务"}
	for _, task := range value.Enabled {
		gui.state.States[task] = "pending"
	}
	gui.Unlock()
	go guiRun(ctx, cancel, value, paths)
	return id, nil
}

var guiExecuteCommand = guiCommand

func guiRun(ctx context.Context, cancel context.CancelFunc, value GUIConfig, paths map[string]string) {
	var runErr error
	var warnings []string
	capture := new(guiCapturedOutput)
	ctx = context.WithValue(ctx, guiOutputKey{}, capture)
	prepared := false
	restoreFailed := false
	defer func() {
		if prepared {
			restoreCtx, done := context.WithTimeout(context.Background(), 15*time.Second)
			if err := guiExecuteCommand(restoreCtx, "/usr/local/bin/maa-phone-display", "restore"); err != nil {
				log.Error("[GUI] 恢复手机显示设置失败")
				restoreFailed = true
				if runErr == nil {
					runErr = errors.New("恢复手机显示设置失败")
				}
			}
			done()
		}
		gui.Lock()
		gui.state.Running = false
		gui.state.Stopping = false
		gui.state.Finished = time.Now().Format(time.RFC3339)
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			gui.state.Message = "已达到运行时长上限，任务已停止"
		} else if ctx.Err() != nil {
			gui.state.Message = "任务已停止"
		} else if runErr != nil {
			gui.state.Message = "任务失败：" + runErr.Error()
		} else if len(value.Enabled) == 1 && value.Enabled[0] == "resource_update" {
			gui.state.Message = "资源热更新完成，MaaCore 未升级"
		} else if len(value.Enabled) == 1 && (value.Enabled[0] == "core_update" || value.Enabled[0] == "core_rollback") {
			gui.state.Message = "MaaCore 检查／维护完成，请查看版本更新状态"
		} else if len(warnings) > 0 {
			gui.state.Message = "任务执行结束，有提示：" + strings.Join(warnings, "；")
		} else {
			gui.state.Message = "全部任务已完成"
		}
		if restoreFailed {
			gui.state.Message += "；请重新连接手机以恢复显示设置"
		}
		log.Info("[GUI] " + gui.state.Message)
		gui.cancel = nil
		gui.Unlock()
		guiSaveRecognition(value, capture, gui.state.ID, runErr == nil && ctx.Err() == nil)
		cancel()
		scheduler.ExecutionMu.Unlock()
	}()
	for _, id := range value.Enabled {
		if ctx.Err() != nil {
			return
		}
		gui.Lock()
		gui.state.Current = id
		gui.state.States[id] = "running"
		gui.state.Message = "正在执行：" + guiNames[id]
		gui.Unlock()
		log.Info("[GUI] 开始任务：" + guiNames[id])
		taskWarning := false
		runValue := value
		if id == "update" {
			var due bool
			runValue, due = guiUserDataDue(value, time.Now())
			if !due {
				gui.Lock()
				gui.state.States[id] = "skipped"
				gui.state.Message = "更新数据未到执行间隔，已跳过"
				gui.Unlock()
				log.Info("[GUI] 更新数据未到执行间隔，已跳过")
				continue
			}
			var err error
			paths[id], err = guiTaskFile(runValue, id, "gui-run")
			if err != nil {
				runErr = err
				return
			}
		}
		if id == "launch" {
			runErr = guiLaunchGame(ctx, value)
		} else if id == "resource_update" {
			runErr = guiExecuteCommand(ctx, "maa", "hot-update", "--batch")
		} else if id == "core_update" {
			runErr = corePerformUpgrade(ctx)
		} else if id == "core_rollback" {
			runErr = corePerformRollback(ctx)
		} else {
			if !prepared {
				prepared = true
				runErr = guiExecuteCommand(ctx, "/usr/local/bin/maa-phone-display", "prepare")
			}
			if runErr == nil {
				mark := guiMallLogMark{}
				if id == "mall" {
					mark = guiMarkMallLog()
				}
				runErr = guiExecuteCommand(context.WithValue(ctx, guiRecognitionTaskKey{}, id), "maa", "run", paths[id], "--batch")
				if runErr == nil && id == "mall" && ctx.Err() == nil {
					summary, warning := guiMallReport(mark, runValue.Params["mall"])
					if summary != "" {
						gui.Lock()
						if gui.state.TaskResults == nil {
							gui.state.TaskResults = map[string]string{}
						}
						gui.state.TaskResults[id] = summary
						gui.Unlock()
						if warning {
							taskWarning = true
							warnings = append(warnings, "信用收支："+summary)
							gui.Lock()
							gui.state.Warnings = append([]string(nil), warnings...)
							gui.Unlock()
							log.Warn("[GUI] 信用收支：" + summary)
						} else {
							log.Info("[GUI] 信用收支：" + summary)
						}
					}
				}
				if runErr != nil && id == "fight" && ctx.Err() == nil {
					runErr = guiFightFailure(runErr)
				}
				if runErr == nil && id == "update" {
					runErr = guiUserDataCompleted(runValue, time.Now())
				}
			}
		}
		gui.Lock()
		state := "completed"
		if taskWarning {
			state = "warning"
		}
		if runErr != nil {
			state = "failed"
		}
		if ctx.Err() != nil {
			state = "stopped"
		}
		gui.state.States[id] = state
		gui.Unlock()
		if runErr != nil {
			return
		}
	}
	if value.After == "close_game" && len(paths) > 0 {
		task := GUIConfig{Params: map[string]map[string]any{"startup": {"client_type": value.Client}}}
		path := filepath.Join(config.D.FightDir, "gui-run", "close.json")
		b, _ := json.Marshal(map[string]any{"tasks": []any{map[string]any{"type": "CloseDown", "params": task.Params["startup"]}}})
		if err := os.WriteFile(path, b, 0600); err == nil {
			runErr = guiExecuteCommand(ctx, "maa", "run", "gui-run/close", "--batch")
		}
	}
}
func guiStop(c *gin.Context) {
	var body struct {
		ID string `json:"id"`
	}
	if c.ShouldBindJSON(&body) != nil {
		c.JSON(400, gin.H{"error": "缺少运行编号"})
		return
	}
	gui.Lock()
	defer gui.Unlock()
	if !gui.state.Running {
		c.JSON(200, gin.H{"stopped": true})
		return
	}
	if body.ID != gui.state.ID {
		c.JSON(409, gin.H{"error": "当前任务已更换，请刷新状态"})
		return
	}
	gui.state.Stopping = true
	if gui.state.Current == "resource_update" {
		gui.state.Message = "正在停止资源热更新"
	} else {
		gui.state.Message = "正在停止并恢复手机显示设置"
	}
	if gui.cancel != nil {
		gui.cancel()
	}
	c.JSON(200, gin.H{"stopping": true})
}
func InitGUI(api *gin.RouterGroup) {
	gui.state = GUIState{States: map[string]string{}, Message: "就绪"}
	guiInitCoreUpdates()
	guiInitSchedules()
	g := api.Group("/gui")
	g.GET("/plugin",func(c *gin.Context){c.JSON(200,gin.H{"id":"maa","api_version":1,"name":"MAA"})})
 g.GET("/config", guiConfigHandler)
	g.POST("/config", guiConfigHandler)
	g.GET("/status", guiStatus)
	g.GET("/performance", guiPerformance)
	g.POST("/performance", guiPerformance)
	g.GET("/schedules", guiSchedulesList)
	g.POST("/schedules", guiScheduleSave)
	g.POST("/schedules/delete", guiScheduleDelete)
	g.GET("/connection", guiConnection)
	g.GET("/catalog", guiCatalog)
	g.GET("/stages", guiStages)
	g.POST("/stages/refresh", guiStages)
	g.GET("/infrast", guiInfrast)
	g.POST("/infrast", guiInfrast)
	g.POST("/copilot", guiCopilot)
	g.GET("/copilot", guiCopilot)
	g.DELETE("/copilot", guiCopilot)
	g.GET("/results", guiResults)
	g.DELETE("/results", guiResults)
	g.POST("/validate", guiValidatePlan)
	g.POST("/start", guiStart)
	g.POST("/stop", guiStop)
	g.POST("/resource-update", guiResourceUpdate)
	g.GET("/core-update", guiCoreState)
	g.POST("/core-update/settings", guiCoreSettings)
	g.POST("/core-update/check", guiCoreCheck)
	g.POST("/core-update/start", guiCoreStart)
	g.POST("/core-update/rollback", guiCoreRollback)
}
