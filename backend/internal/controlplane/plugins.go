package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type pluginManifest struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Frontend    string         `json:"frontend"`
	APIVersion  int            `json:"api_version"`
	Target      string         `json:"target"`
	TargetEnv   string         `json:"target_env,omitempty"`
	HealthPath  string         `json:"health_path"`
	HealthMatch map[string]any `json:"health_match,omitempty"`
	Enabled     *bool          `json:"enabled,omitempty"`
}
type pluginStatus struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Frontend    string `json:"frontend"`
	APIVersion  int    `json:"api_version"`
	Available   bool   `json:"available"`
	Reason      string `json:"reason,omitempty"`
}
type pluginManager struct {
	mu          sync.Mutex
	fingerprint [32]byte
	checked     time.Time
	statuses    []pluginStatus
	client      *http.Client
}

var pluginIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
var pluginEnvPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

func newPluginManager() *pluginManager {
	return &pluginManager{client: &http.Client{Timeout: 1500 * time.Millisecond, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func pluginTarget(p pluginManifest) (*url.URL, error) {
	address := p.Target
	if p.TargetEnv != "" && os.Getenv(p.TargetEnv) != "" {
		address = os.Getenv(p.TargetEnv)
	}
	u, e := url.Parse(address)
	if e != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("invalid plugin target")
	}
	return u, nil
}
func validPlugin(p pluginManifest) bool {
	if !pluginIDPattern.MatchString(p.ID) || p.Name == "" || len(p.Name) > 80 || !pluginIDPattern.MatchString(p.Frontend) || p.APIVersion != 1 {
		return false
	}
	if p.TargetEnv != "" && !pluginEnvPattern.MatchString(p.TargetEnv) {
		return false
	}
	if !strings.HasPrefix(p.HealthPath, "/") || strings.HasPrefix(p.HealthPath, "//") || strings.ContainsAny(p.HealthPath, "\r\n?#") {
		return false
	}
	_, e := pluginTarget(p)
	return e == nil
}
func (m *pluginManager) manifests() []pluginManifest {
	merged := map[string]pluginManifest{}
	for _, dir := range []string{valueOr("SCRCPYCAT_BUNDLED_PLUGIN_DIR", "/opt/scrcpycat/plugins"), valueOr("SCRCPYCAT_PLUGIN_DIR", "/data/plugins")} {
		paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
		sort.Strings(paths)
		for _, file := range paths {
			f, e := os.Open(file)
			if e != nil {
				continue
			}
			b, e := io.ReadAll(io.LimitReader(f, 65537))
			f.Close()
			if e != nil || len(b) > 65536 {
				continue
			}
			var p pluginManifest
			if json.Unmarshal(b, &p) != nil || !validPlugin(p) {
				continue
			}
			merged[p.ID] = p
		}
	}
	ids := make([]string, 0, len(merged))
	for id := range merged {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := []pluginManifest{}
	for _, id := range ids {
		if len(out) >= 32 {
			break
		}
		p := merged[id]
		if p.Enabled == nil || *p.Enabled {
			out = append(out, p)
		}
	}
	return out
}
func (m *pluginManager) find(id string) (pluginManifest, bool) {
	for _, p := range m.manifests() {
		if p.ID == id {
			return p, true
		}
	}
	return pluginManifest{}, false
}
func (m *pluginManager) probe(ctx context.Context, p pluginManifest) pluginStatus {
	s := pluginStatus{ID: p.ID, Name: p.Name, Description: p.Description, Frontend: p.Frontend, APIVersion: p.APIVersion, Reason: "service_unavailable"}
	u, e := pluginTarget(p)
	if e != nil {
		return s
	}
	u.Path = strings.TrimRight(u.Path, "/") + p.HealthPath
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if e != nil {
		return s
	}
	res, e := m.client.Do(req)
	if e != nil {
		return s
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return s
	}
	var body map[string]any
	if json.NewDecoder(io.LimitReader(res.Body, 16384)).Decode(&body) != nil {
		return s
	}
	for k, want := range p.HealthMatch {
		got, _ := json.Marshal(body[k])
		expected, _ := json.Marshal(want)
		if string(got) != string(expected) {
			s.Reason = "incompatible_service"
			return s
		}
	}
	s.Available = true
	s.Reason = ""
	return s
}
func (m *pluginManager) list(ctx context.Context) []pluginStatus {
	manifest := m.manifests()
	b, _ := json.Marshal(manifest)
	// Environment-selected targets participate in cache invalidation as well.
	for _, p := range manifest {
		u, _ := pluginTarget(p)
		b = append(b, []byte(u.String())...)
	}
	fingerprint := sha256.Sum256(b)
	m.mu.Lock()
	defer m.mu.Unlock()
	if fingerprint == m.fingerprint && time.Since(m.checked) < 3*time.Second {
		return append([]pluginStatus{}, m.statuses...)
	}
	result := make([]pluginStatus, len(manifest))
	var wg sync.WaitGroup
	for i, p := range manifest {
		wg.Add(1)
		go func(i int, p pluginManifest) { defer wg.Done(); result[i] = m.probe(ctx, p) }(i, p)
	}
	wg.Wait()
	m.statuses = result
	m.checked = time.Now()
	m.fingerprint = fingerprint
	return append([]pluginStatus{}, result...)
}
func (s *Server) pluginsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, 200, map[string]any{"api_version": 1, "plugins": s.plugins.list(r.Context())})
}
func (s *Server) pluginRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	rest := strings.TrimPrefix(r.URL.Path, "/api/plugins/")
	id, action, _ := strings.Cut(rest, "/")
	if !pluginIDPattern.MatchString(id) {
		writeError(w, 404, "插件不存在")
		return
	}
	p, found := s.plugins.find(id)
	if !found {
		writeError(w, 404, "插件未启用")
		return
	}
	if action == "session" {
		s.requireAdmin(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}
			status := s.plugins.probe(r.Context(), p)
			if !status.Available {
				writeError(w, 503, "插件服务未启动或尚未就绪")
				return
			}
			prefix := "/api/plugins/" + p.ID + "/proxy"
			secure := strings.HasPrefix(s.config.PublicURL, "https://") || r.TLS != nil
			http.SetCookie(w, &http.Cookie{Name: "sc_plugin_" + p.ID, Value: bearerToken(r.Header.Get("Authorization")), Path: prefix, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode, MaxAge: 1800})
			writeJSON(w, 200, map[string]string{"url": "/plugins/" + p.ID, "api_base": prefix})
		})(w, r)
		return
	}
	if action != "proxy" && !strings.HasPrefix(action, "proxy/") {
		writeError(w, 404, "插件接口不存在")
		return
	}
	s.pluginProxy(w, r, p, strings.TrimPrefix(action, "proxy"))
}
func (s *Server) pluginProxy(w http.ResponseWriter, r *http.Request, p pluginManifest, suffix string) {
	authenticated := r.Clone(r.Context())
	authenticated.Header = r.Header.Clone()
	if authenticated.Header.Get("Authorization") == "" {
		cookie, e := r.Cookie("sc_plugin_" + p.ID)
		if e != nil {
			writeError(w, 401, "请先登录并打开插件")
			return
		}
		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			writeError(w, 403, "插件仅允许同源访问")
			return
		}
		authenticated.Header.Set("Authorization", "Bearer "+cookie.Value)
	}
	user, e := s.userFromRequest(authenticated)
	if e != nil {
		writeError(w, 401, "插件登录已过期")
		return
	}
	if user.Role != RoleAdmin {
		writeError(w, 403, "插件仅管理员可使用")
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && !s.originAllowed(origin) {
		writeError(w, 403, "插件请求来源不允许")
		return
	}
	target, e := pluginTarget(p)
	if e != nil {
		writeError(w, 503, "插件地址无效")
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	prefix := "/api/plugins/" + p.ID + "/proxy"
	proxy.Director = func(req *http.Request) {
		director(req)
		req.URL.Path = strings.TrimRight(target.Path, "/") + "/" + strings.TrimLeft(suffix, "/")
		req.URL.RawPath = ""
		req.Host = target.Host
		req.Header.Del("Authorization")
		req.Header.Del("Cookie")
		req.Header.Set("X-Forwarded-Prefix", prefix)
	}
	proxy.ModifyResponse = func(response *http.Response) error {
		response.Header.Set("Cache-Control", "no-store")
		response.Header.Set("X-Frame-Options", "SAMEORIGIN")
		response.Header.Del("Set-Cookie")
		if loc := response.Header.Get("Location"); strings.HasPrefix(loc, "/") && !strings.HasPrefix(loc, prefix+"/") {
			response.Header.Set("Location", prefix+loc)
		}
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		writeError(w, 503, "插件服务连接中断，请稍后重试")
	}
	proxy.ServeHTTP(w, r)
}
