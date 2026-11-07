package controlplane

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	adbclient "github.com/cwithw/ScrcpyCat/internal/adb"
)

var pinPattern = regexp.MustCompile(`^[0-9]{4,16}$`)
var currentLockPattern = regexp.MustCompile(`(?m)^.*\(current\):[^\n]*\bdeviceLocked=([01])\b`)

func parseDeviceLocked(output string) (bool, bool) {
	match := currentLockPattern.FindStringSubmatch(output)
	if len(match) != 2 {
		return false, false
	}
	return match[1] == "1", true
}

// PIN is supplied only through non-PTY stdin, never in command arguments or logs.
// Check trust again before Enter: some phones auto-confirm their last PIN digit.
const unlockPINScript = `input keyevent 224
sleep 0.3
input keyevent 4
input keyevent 82
sleep 0.5
IFS= read -r secret || exit 1
while [ -n "$secret" ]; do
 rest=${secret#?}
 digit=${secret%"$rest"}
 secret=$rest
 input keyevent "$((digit + 7))" || exit 1
done
unset secret rest digit
sleep 0.3
state=$(dumpsys trust | grep '(current):')
case "$state" in *deviceLocked=1*) input keyevent 66 ;; esac
`

func deviceLocked(ctx context.Context, client *adbclient.Client, serial string) (bool, bool) {
	var out bytes.Buffer
	if client.Run(ctx, serial, "dumpsys trust", &out, io.Discard) != nil {
		return false, false
	}
	return parseDeviceLocked(out.String())
}

func (s *Server) deviceUnlock(w http.ResponseWriter, r *http.Request, id string) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	user := currentUser(r)
	if !user.Can("shell") || !s.store.CanAccessDevice(user, id) {
		writeError(w, http.StatusForbidden, "没有该设备的解锁权限")
		return
	}
	if _, err := s.store.Device(id); err != nil {
		writeError(w, http.StatusNotFound, "设备不存在")
		return
	}
	var body struct {
		PIN string `json:"pin"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 512)
	if decodeJSON(r, &body) != nil || !pinPattern.MatchString(body.PIN) {
		writeError(w, http.StatusBadRequest, "请输入 4–16 位数字 PIN")
		return
	}
	if _, busy := s.unlockInFlight.LoadOrStore(id, true); busy {
		writeError(w, http.StatusConflict, "该设备正在执行解锁，请稍候")
		return
	}
	defer s.unlockInFlight.Delete(id)
	if last, ok := s.unlockLastAttempt.Load(id); ok && time.Since(last.(time.Time)) < 15*time.Second {
		writeError(w, http.StatusTooManyRequests, "请稍候 15 秒再手动尝试")
		return
	}
	address := os.Getenv("SCRCPYCAT_UNLOCK_ADB_SERVER")
	if address == "" {
		address = "scrcpy-web:5037"
	}
	client, err := adbclient.New(address)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "ADB 服务未就绪")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 18*time.Second)
	defer cancel()
	if client.Online(ctx, id) != nil {
		writeError(w, http.StatusServiceUnavailable, "手机未连接或尚未授权 ADB")
		return
	}
	locked, known := deviceLocked(ctx, client, id)
	if !known {
		writeError(w, http.StatusConflict, "无法确认锁屏状态，未输入 PIN，请在手机上检查")
		return
	}
	if !locked {
		writeJSON(w, http.StatusOK, map[string]string{"status": "unlocked", "message": "手机已经解锁，未输入 PIN"})
		return
	}
	s.unlockLastAttempt.Store(id, time.Now())
	session, err := client.Start(ctx, id, unlockPINScript, false, io.Discard, io.Discard)
	if err != nil {
		writeError(w, http.StatusBadGateway, "ADB 执行失败，未自动重试")
		return
	}
	defer session.Close()
	_, err = session.Write([]byte(body.PIN + "\n"))
	body.PIN = ""
	if err == nil {
		err = session.CloseStdin()
	}
	if err == nil {
		err = session.Wait()
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "解锁命令未完成，请检查手机；不会自动重试")
		return
	}
	// Poll status only; never submit the PIN a second time.
	for i := 0; i < 5; i++ {
		select {
		case <-ctx.Done():
			writeError(w, http.StatusGatewayTimeout, "确认解锁状态超时，请检查手机")
			return
		case <-time.After(400 * time.Millisecond):
		}
		locked, known = deviceLocked(ctx, client, id)
		if known && !locked {
			writeJSON(w, http.StatusOK, map[string]string{"status": "unlocked", "message": "已确认手机解锁"})
			return
		}
	}
	status := "unknown"
	message := "已执行一次，无法确认解锁状态，请检查手机；未自动重试"
	if known && locked {
		status = "locked"
		message = "手机仍处于锁屏状态，请检查 PIN 或在手机上操作；未自动重试"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status, "message": message})
}

// Keep route parsing separate so serials are not inferred from client commands.
func unlockDeviceID(path string) string {
	return strings.TrimSuffix(strings.TrimPrefix(path, "/api/devices/"), "/unlock")
}
