package controlplane

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	adbclient "github.com/cwithw/ScrcpyCat/internal/adb"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var managedPackagePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)+$`)

type managedApp struct {
	Package   string `json:"package"`
	Name      string `json:"name"`
	Version   string `json:"version_code"`
	UID       int    `json:"uid"`
	System    bool   `json:"system"`
	Enabled   bool   `json:"enabled"`
	Protected bool   `json:"protected"`
	Running   bool   `json:"running"`
}
type managedProcess struct {
	PID       int    `json:"pid"`
	PPID      int    `json:"ppid"`
	UID       int    `json:"uid"`
	RSS       int64  `json:"rss_kb"`
	VSZ       int64  `json:"vsz_kb"`
	State     string `json:"state"`
	Name      string `json:"name"`
	Package   string `json:"package,omitempty"`
	AppName   string `json:"app_name,omitempty"`
	Protected bool   `json:"protected"`
}

func managedClient() (*adbclient.Client, error) {
	address := os.Getenv("SCRCPYCAT_UNLOCK_ADB_SERVER")
	if address == "" {
		address = "scrcpy-web:5037"
	}
	return adbclient.New(address)
}
func managementRun(ctx context.Context, client *adbclient.Client, id, command string) (string, error) {
	var out, errout bytes.Buffer
	e := client.Run(ctx, id, command, &out, &errout)
	if e != nil {
		message := strings.TrimSpace(errout.String() + " " + out.String())
		if message == "" {
			message = "设备未连接、授权失效或操作超时，请重试"
		}
		if len(message) > 500 {
			message = message[:500]
		}
		return "", fmt.Errorf("ADB 操作失败：%s", message)
	}
	if out.Len() > 4<<20 {
		return "", errors.New("设备返回数据过多")
	}
	return out.String(), nil
}
func managementCommand(ctx context.Context, client *adbclient.Client, id, name string, args ...string) (string, error) {
	command, e := adbclient.Command(name, args...)
	if e != nil {
		return "", e
	}
	return managementRun(ctx, client, id, command)
}
func managementUser(ctx context.Context, client *adbclient.Client, id string) (string, error) {
	out, e := managementCommand(ctx, client, id, "am", "get-current-user")
	n, f := strconv.Atoi(strings.TrimSpace(out))
	if e != nil || f != nil || n < 0 || n > 10000 {
		return "", errors.New("无法确认当前 Android 用户")
	}
	return strconv.Itoa(n), nil
}
func parseManagedApps(list, system, disabled string) []managedApp {
	systems, off := map[string]bool{}, map[string]bool{}
	for _, line := range strings.Fields(system) {
		systems[strings.TrimPrefix(line, "package:")] = true
	}
	for _, line := range strings.Fields(disabled) {
		off[strings.TrimPrefix(line, "package:")] = true
	}
	apps := []managedApp{}
	for _, line := range strings.Split(list, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "package:") {
			continue
		}
		pkg := strings.TrimPrefix(fields[0], "package:")
		if !managedPackagePattern.MatchString(pkg) {
			continue
		}
		a := managedApp{Package: pkg, Name: managedAppName(pkg), System: systems[pkg], Enabled: !off[pkg]}
		for _, f := range fields[1:] {
			if strings.HasPrefix(f, "uid:") {
				a.UID, _ = strconv.Atoi(strings.TrimPrefix(f, "uid:"))
			}
			if strings.HasPrefix(f, "versionCode:") {
				a.Version = strings.TrimPrefix(f, "versionCode:")
			}
		}
		a.Protected = a.System || a.UID < 10000 || pkg == "com.android.shell"
		apps = append(apps, a)
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Package < apps[j].Package })
	return apps
}
func managedAppName(pkg string) string {
	names := map[string]string{"com.hypergryph.arknights": "明日方舟", "com.hypergryph.arknights.bilibili": "明日方舟（B服）", "org.mozilla.firefox": "Firefox", "com.android.chrome": "Chrome", "com.tencent.mm": "微信", "com.tencent.mobileqq": "QQ", "com.eg.android.AlipayGphone": "支付宝", "tv.danmaku.bili": "哔哩哔哩", "com.miui.notes": "小米笔记", "com.android.settings": "系统设置", "com.miui.securitycenter": "手机管家", "com.termux": "Termux"}
	if name := names[pkg]; name != "" {
		return name
	}
	return pkg
}
func readManagedApps(ctx context.Context, c *adbclient.Client, id, user string) ([]managedApp, error) {
	list, e := managementCommand(ctx, c, id, "pm", "list", "packages", "-U", "--show-versioncode", "--user", user)
	if e != nil {
		return nil, e
	}
	system, e := managementCommand(ctx, c, id, "pm", "list", "packages", "-s", "--user", user)
	if e != nil {
		return nil, e
	}
	disabled, e := managementCommand(ctx, c, id, "pm", "list", "packages", "-d", "--user", user)
	if e != nil {
		return nil, e
	}
	return parseManagedApps(list, system, disabled), nil
}
func parseManagedProcesses(output string, apps []managedApp) []managedProcess {
	byUID := map[int][]managedApp{}
	for _, a := range apps {
		byUID[a.UID] = append(byUID[a.UID], a)
	}
	out := []managedProcess{}
	for _, line := range strings.Split(output, "\n") {
		f := strings.Fields(line)
		if len(f) < 7 {
			continue
		}
		uid, e := strconv.Atoi(f[0])
		if e != nil {
			continue
		}
		pid, _ := strconv.Atoi(f[1])
		ppid, _ := strconv.Atoi(f[2])
		rss, _ := strconv.ParseInt(f[3], 10, 64)
		vsz, _ := strconv.ParseInt(f[4], 10, 64)
		p := managedProcess{PID: pid, PPID: ppid, UID: uid, RSS: rss, VSZ: vsz, State: f[5], Name: strings.Join(f[6:], " "), Protected: true}
		candidates := byUID[uid]
		if len(candidates) == 1 {
			p.Package = candidates[0].Package
			p.AppName = candidates[0].Name
			p.Protected = candidates[0].Protected
		}
		if uid < 10000 || pid <= 1 || strings.Contains(p.Name, "scrcpy") || p.Name == "adbd" {
			p.Protected = true
		}
		out = append(out, p)
	}
	return out
}
func (s *Server) deviceManagement(w http.ResponseWriter, r *http.Request, id string) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	user := currentUser(r)
	if !user.Can("shell") || !s.store.CanAccessDevice(user, id) {
		writeError(w, 403, "没有该设备的应用和进程管理权限")
		return
	}
	if _, e := s.store.Device(id); e != nil {
		writeError(w, 404, "设备不存在")
		return
	}
	var v struct {
		Action  string `json:"action"`
		Package string `json:"package"`
		PID     int    `json:"pid"`
		Name    string `json:"name"`
	}
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 2048)
		if decodeJSON(r, &v) != nil || !managedPackagePattern.MatchString(v.Package) {
			writeError(w, 400, "操作或应用包名无效")
			return
		}
		switch v.Action {
		case "launch", "force_stop", "uninstall", "disable", "enable":
		default:
			writeError(w, 400, "不支持的应用操作")
			return
		}
		if v.PID != 0 && v.Action != "force_stop" {
			writeError(w, 400, "进程视图仅支持停止所属应用")
			return
		}
	}
	c, e := managedClient()
	if e != nil {
		writeError(w, 503, "ADB 服务未就绪")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	androidUser, e := managementUser(ctx, c, id)
	if e != nil {
		writeError(w, 503, e.Error())
		return
	}
	apps, e := readManagedApps(ctx, c, id, androidUser)
	if e != nil {
		writeError(w, 503, e.Error())
		return
	}
	if r.Method == http.MethodGet {
		switch r.URL.Query().Get("view") {
		case "apps":
			raw, _ := managementCommand(ctx, c, id, "ps", "-A", "-o", "UID")
			active := map[int]bool{}
			for _, line := range strings.Fields(raw) {
				if uid, e := strconv.Atoi(line); e == nil {
					active[uid] = true
				}
			}
			for i := range apps {
				apps[i].Running = active[apps[i].UID]
			}
			writeJSON(w, 200, map[string]any{"items": apps, "user": androidUser, "checked_at": time.Now().UTC()})
			return
		case "details":
			pkg := r.URL.Query().Get("package")
			if !managedPackagePattern.MatchString(pkg) {
				writeError(w, 400, "应用包名无效")
				return
			}
			found := false
			for _, a := range apps {
				if a.Package == pkg {
					found = true
				}
			}
			if !found {
				writeError(w, 404, "当前用户未安装该应用")
				return
			}
			dump, e := managementCommand(ctx, c, id, "dumpsys", "package", pkg)
			if e != nil {
				writeError(w, 502, e.Error())
				return
			}
			details := []string{}
			for _, line := range strings.Split(dump, "\n") {
				for _, key := range []string{"versionName=", "versionCode=", "firstInstallTime=", "lastUpdateTime=", "dataDir=", "targetSdk=", "minSdk=", "installerPackageName=", "requested permissions:", "android.permission."} {
					if strings.Contains(line, key) {
						details = append(details, strings.TrimSpace(line))
						break
					}
				}
				if len(details) >= 160 {
					break
				}
			}
			writeJSON(w, 200, map[string]any{"package": pkg, "details": details})
			return
		case "processes":
			raw, e := managementCommand(ctx, c, id, "ps", "-A", "-o", "UID,PID,PPID,RSS,VSZ,STAT,NAME")
			if e != nil {
				writeError(w, 502, e.Error())
				return
			}
			mem, _ := managementCommand(ctx, c, id, "cat", "/proc/meminfo")
			memory := map[string]int64{}
			for _, line := range strings.Split(mem, "\n") {
				f := strings.Fields(line)
				if len(f) > 1 && (f[0] == "MemTotal:" || f[0] == "MemAvailable:") {
					memory[strings.TrimSuffix(f[0], ":")], _ = strconv.ParseInt(f[1], 10, 64)
				}
			}
			writeJSON(w, 200, map[string]any{"items": parseManagedProcesses(raw, apps), "memory_kb": memory, "checked_at": time.Now().UTC()})
			return
		default:
			writeError(w, 400, "请选择进程或应用视图")
			return
		}
	}
	var target *managedApp
	for i := range apps {
		if apps[i].Package == v.Package {
			target = &apps[i]
			break
		}
	}
	if target == nil {
		writeError(w, 404, "当前用户未安装该应用")
		return
	}
	if v.Action != "launch" && target.Protected {
		writeError(w, 403, "此应用属于系统或受保护组件，只提供查看和打开功能")
		return
	}
	if v.PID != 0 {
		raw, e := managementCommand(ctx, c, id, "ps", "-A", "-o", "UID,PID,PPID,RSS,VSZ,STAT,NAME")
		if e != nil {
			writeError(w, 502, e.Error())
			return
		}
		valid := false
		for _, p := range parseManagedProcesses(raw, apps) {
			if p.PID == v.PID && p.Name == v.Name && p.Package == v.Package && !p.Protected {
				valid = true
			}
		}
		if !valid {
			writeError(w, 409, "进程已变化，请刷新列表后重试")
			return
		}
	}
	var output string
	switch v.Action {
	case "launch":
		var resolved string
		resolved, e = managementCommand(ctx, c, id, "cmd", "package", "resolve-activity", "--brief", "--user", androidUser, "-a", "android.intent.action.MAIN", "-c", "android.intent.category.LAUNCHER", v.Package)
		if e == nil {
			component := ""
			for _, line := range strings.Split(resolved, "\n") {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "/") && !strings.ContainsAny(line, " \t") {
					component = line
				}
			}
			if component == "" {
				e = errors.New("该应用没有可打开的主界面")
			} else {
				output, e = managementCommand(ctx, c, id, "am", "start", "--user", androidUser, "-n", component)
			}
		}
	case "force_stop":
		output, e = managementCommand(ctx, c, id, "am", "force-stop", "--user", androidUser, v.Package)
	case "uninstall":
		output, e = managementCommand(ctx, c, id, "pm", "uninstall", "--user", androidUser, v.Package)
	case "disable":
		output, e = managementCommand(ctx, c, id, "pm", "disable-user", "--user", androidUser, v.Package)
	case "enable":
		output, e = managementCommand(ctx, c, id, "pm", "enable", "--user", androidUser, v.Package)
	default:
		writeError(w, 400, "不支持的应用操作")
		return
	}
	if e != nil {
		writeError(w, 502, e.Error())
		return
	}
	if (v.Action == "uninstall" && !strings.Contains(output, "Success")) || strings.Contains(output, "Failure") || strings.Contains(output, "Error:") || strings.Contains(output, "No activities found") {
		writeError(w, 409, "应用操作未完成："+strings.TrimSpace(output))
		return
	}
	s.audit(user, "device_app."+v.Action, id, map[string]any{"package": v.Package})
	writeJSON(w, 200, map[string]any{"success": true, "package": v.Package, "message": "操作已完成"})
}
