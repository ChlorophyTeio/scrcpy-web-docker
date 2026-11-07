package controlplane

import (
	"bytes"
	"context"
	adbclient "github.com/cwithw/ScrcpyCat/internal/adb"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// SurfaceFlinger reports hardware power, including scrcpy's panel-only off.
// PowerManager wakefulness can remain Awake while the physical display is off.
const screenStateScript = `dumpsys SurfaceFlinger | grep -E '(^Display [0-9]+|^Virtual Display [0-9]+|connectionType=|powerMode=)'`

var physicalDisplayLine = regexp.MustCompile(`^Display [0-9]+(?:\s.*)?$`)
var displayPowerLine = regexp.MustCompile(`(?i)\bpowerMode\s*=\s*(OnSuspend|DozeSuspend|Off|Doze|On|[0-4])\b`)

func parsePhysicalScreen(output string) (bool, bool) {
	type display struct {
		internal bool
		on       bool
		known    bool
	}
	var displays []display
	current := -1
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Virtual Display ") || strings.Contains(line, "(virtual,") {
			current = -1
			continue
		}
		if physicalDisplayLine.MatchString(strings.TrimSpace(line)) {
			displays = append(displays, display{})
			current = len(displays) - 1
			continue
		}
		if current < 0 {
			continue
		}
		if strings.Contains(line, "connectionType=Internal") {
			displays[current].internal = true
		}
		m := displayPowerLine.FindStringSubmatch(line)
		if len(m) == 2 {
			v := strings.ToLower(m[1])
			displays[current].known = true
			displays[current].on = v == "on" || v == "onsuspend" || v == "2" || v == "4"
		}
	}
	for _, d := range displays {
		if d.internal && d.known {
			return d.on, true
		}
	}
	// Older releases omit connectionType. Only accept an unambiguous single panel.
	var found *display
	for i := range displays {
		if displays[i].known {
			if found != nil {
				return false, false
			}
			found = &displays[i]
		}
	}
	if found != nil {
		return found.on, true
	}
	return false, false
}
func (s *Server) deviceScreenState(w http.ResponseWriter, r *http.Request, id string) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if !s.store.CanAccessDevice(currentUser(r), id) {
		writeError(w, 403, "没有该设备的访问权限")
		return
	}
	if _, e := s.store.Device(id); e != nil {
		writeError(w, 404, "设备不存在")
		return
	}
	address := os.Getenv("SCRCPYCAT_UNLOCK_ADB_SERVER")
	if address == "" {
		address = "scrcpy-web:5037"
	}
	client, e := adbclient.New(address)
	if e != nil {
		writeError(w, 503, "ADB 服务未就绪")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var out bytes.Buffer
	if e = client.Run(ctx, id, screenStateScript, &out, io.Discard); e != nil {
		writeError(w, 503, "无法读取屏幕状态，请检查设备连接与 ADB 授权")
		return
	}
	on, known := parsePhysicalScreen(out.String())
	writeJSON(w, 200, map[string]any{"known": known, "on": on, "checked_at": time.Now().UTC().Format(time.RFC3339)})
}
