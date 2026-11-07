package controlplane

import (
	"context"
	"errors"
	"fmt"
	adbclient "github.com/cwithw/ScrcpyCat/internal/adb"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func fileToolPath(raw string) (string, error) {
	if !strings.HasPrefix(raw, "/") || len(raw) > 4096 {
		return "", errors.New("请输入完整绝对路径")
	}
	for _, r := range raw {
		if r < 32 || r == 127 {
			return "", errors.New("路径含不支持的控制字符")
		}
	}
	return path.Clean(raw), nil
}
func managedStoragePath(p string) bool {
	for _, root := range []string{"/sdcard", "/storage/emulated/0", "/data/local/tmp"} {
		if p == root || strings.HasPrefix(p, root+"/") {
			return true
		}
	}
	return false
}
func storageRoot(p string) bool {
	return p == "/sdcard" || p == "/storage/emulated/0" || p == "/data/local/tmp"
}

type fileToolInfo struct {
	Path     string  `json:"path"`
	Type     string  `json:"type"`
	Size     int64   `json:"size"`
	Modified int64   `json:"modified"`
	Mode     string  `json:"mode"`
	Text     *string `json:"text,omitempty"`
}

func parseFileToolInfo(p, raw string) (fileToolInfo, error) {
	f := strings.Split(strings.TrimSpace(raw), "|")
	if len(f) != 4 {
		return fileToolInfo{}, errors.New("无法解析文件属性")
	}
	size, e := strconv.ParseInt(f[1], 10, 64)
	modified, ferr := strconv.ParseInt(f[2], 10, 64)
	if e != nil || ferr != nil {
		return fileToolInfo{}, errors.New("文件属性无效")
	}
	return fileToolInfo{Path: p, Type: f[0], Size: size, Modified: modified, Mode: f[3]}, nil
}
func fileToolCanonical(ctx context.Context, c *adbclient.Client, id, p string) (string, error) {
	out, e := managementCommand(ctx, c, id, "readlink", "-f", "--", p)
	if e != nil {
		return "", e
	}
	value := strings.TrimSpace(out)
	if value == "" || !managedStoragePath(value) {
		return "", errors.New("复制和移动仅支持内部存储或临时文件目录，不能通过链接访问其他位置")
	}
	return value, nil
}
func (s *Server) deviceFileTools(w http.ResponseWriter, r *http.Request, id string) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	user := currentUser(r)
	if !user.Can("file_library") || !s.store.CanAccessDevice(user, id) {
		writeError(w, 403, "没有该设备的文件管理权限")
		return
	}
	if _, e := s.store.Device(id); e != nil {
		writeError(w, 404, "设备不存在")
		return
	}
	c, e := managedClient()
	if e != nil {
		writeError(w, 503, "ADB 服务未就绪")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if r.Method == http.MethodGet {
		p, e := fileToolPath(r.URL.Query().Get("path"))
		if e != nil {
			writeError(w, 400, e.Error())
			return
		}
		raw, e := managementCommand(ctx, c, id, "stat", "-L", "-c", "%F|%s|%Y|%a", "--", p)
		if e != nil {
			writeError(w, 403, "无法读取文件属性，请检查路径或 Android 访问权限")
			return
		}
		info, e := parseFileToolInfo(p, raw)
		if e != nil {
			writeError(w, 502, e.Error())
			return
		}
		if r.URL.Query().Get("preview") == "1" {
			if !strings.HasPrefix(info.Type, "regular") || info.Size > 256<<10 {
				writeError(w, 400, "仅支持预览不超过 256 KiB 的普通文本文件")
				return
			}
			text, e := managementCommand(ctx, c, id, "head", "-c", "262144", "--", p)
			if e != nil {
				writeError(w, 403, e.Error())
				return
			}
			if !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
				writeError(w, 400, "该文件不是 UTF-8 文本，请下载后查看")
				return
			}
			info.Text = &text
		}
		writeJSON(w, 200, info)
		return
	}
	var v struct {
		Action      string `json:"action"`
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 12<<10)
	if decodeJSON(r, &v) != nil {
		writeError(w, 400, "文件操作格式无效")
		return
	}
	if v.Action != "rename" && v.Action != "move" && v.Action != "copy" {
		writeError(w, 400, "不支持的文件操作")
		return
	}
	src, e := fileToolPath(v.Source)
	if e != nil {
		writeError(w, 400, e.Error())
		return
	}
	dest, e := fileToolPath(v.Destination)
	if e != nil {
		writeError(w, 400, e.Error())
		return
	}
	if !managedStoragePath(src) || !managedStoragePath(dest) || storageRoot(src) || storageRoot(dest) || src == dest || strings.HasPrefix(dest, src+"/") {
		writeError(w, 400, "源或目标路径不适合执行此操作")
		return
	}
	if v.Action == "rename" && path.Dir(src) != path.Dir(dest) {
		writeError(w, 400, "重命名不能跨目录，请使用移动")
		return
	}
	if _, busy := s.fileToolsInFlight.LoadOrStore(id, true); busy {
		writeError(w, 409, "此设备正在处理文件，请稍后重试")
		return
	}
	defer s.fileToolsInFlight.Delete(id)
	canonicalSource, e := fileToolCanonical(ctx, c, id, src)
	if e != nil || storageRoot(canonicalSource) {
		writeError(w, 400, "源文件不可操作或指向存储根目录")
		return
	}
	parent, e := fileToolCanonical(ctx, c, id, path.Dir(dest))
	if e != nil {
		writeError(w, 400, "目标目录不存在或不可访问")
		return
	}
	canonicalDest := path.Join(parent, path.Base(dest))
	if canonicalSource == canonicalDest || strings.HasPrefix(canonicalDest, canonicalSource+"/") {
		writeError(w, 400, "不能将目录复制或移动到自身内部")
		return
	}
	_, e = managementCommand(ctx, c, id, "sh", "-c", `[ ! -e "$1" ] && [ ! -L "$1" ]`, "file-target", dest)
	if e != nil {
		writeError(w, 409, "目标已存在，未覆盖任何文件")
		return
	}
	tool, args := "mv", []string{"-n", "--", src, dest}
	if v.Action == "copy" {
		tool = "cp"
		args = []string{"-Rn", "--", src, dest}
	}
	_, e = managementCommand(ctx, c, id, tool, args...)
	if e != nil {
		writeError(w, 502, e.Error())
		return
	}
	if v.Action != "copy" {
		if _, e = managementCommand(ctx, c, id, "test", "!", "-e", src); e != nil {
			writeError(w, 409, "文件未移动，目标可能已存在，请刷新后检查")
			return
		}
	}
	s.audit(user, "device_file."+v.Action, id, map[string]any{"source": src, "destination": dest})
	writeJSON(w, 200, map[string]any{"success": true, "destination": dest, "message": fmt.Sprintf("%s 完成", v.Action)})
}
