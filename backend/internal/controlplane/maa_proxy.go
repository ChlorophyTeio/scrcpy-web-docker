package controlplane

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

const maaCookie = "sc_maa_session"

func maaTarget() (*url.URL, error) {
	address := os.Getenv("SCRCPYCAT_MAA_URL")
	if address == "" {
		address = "http://maa-web:8080"
	}
	return url.Parse(address)
}
func (s *Server) maaSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	target, err := maaTarget()
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		writeError(w, 503, "MAA 地址配置无效")
		return
	}
	client := http.Client{Timeout: 3 * time.Second}
	check, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target.String()+"/api/GetRunningTask", nil)
	if err != nil {
		writeError(w, 503, "MAA 地址配置无效")
		return
	}
	response, err := client.Do(check)
	if err != nil {
		writeError(w, 503, "MAA 服务暂未就绪，请稍后重新连接")
		return
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		writeError(w, 503, "MAA 服务响应异常，请稍后重新连接")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: maaCookie, Value: bearerToken(r.Header.Get("Authorization")), Path: "/maa", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: 1800})
	writeJSON(w, http.StatusOK, map[string]string{"url": "/maa-console"})
}
func (s *Server) maaProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method == http.MethodGet && (r.URL.Path == "/maa" || r.URL.Path == "/maa/") {
		http.Redirect(w, r, "/maa-console", http.StatusSeeOther)
		return
	}
	authenticated := r.Clone(r.Context())
	authenticated.Header = r.Header.Clone()
	if authenticated.Header.Get("Authorization") == "" {
		cookie, err := r.Cookie(maaCookie)
		if err != nil {
			if (r.URL.Path == "/maa" || r.URL.Path == "/maa/") && r.Method == http.MethodGet {
				http.Redirect(w, r, "/maa-console", http.StatusSeeOther)
				return
			}
			writeError(w, 401, "请先登录远控管理界面并打开 MAA")
			return
		}
		authenticated.Header.Set("Authorization", "Bearer "+cookie.Value)
		// MAA has state-changing GET endpoints; restrict cookie access to this origin.
		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			writeError(w, 403, "MAA 仅允许同源访问")
			return
		}
	}
	user, err := s.userFromRequest(authenticated)
	if err != nil {
		writeError(w, 401, "MAA 登录已过期，请重新打开 MAA 页面")
		return
	}
	if user.Role != RoleAdmin {
		writeError(w, 403, "MAA 仅管理员可使用")
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && !s.originAllowed(origin) {
		writeError(w, 403, "MAA 请求来源不允许")
		return
	}
	if r.URL.Path == "/maa" || r.URL.Path == "/maa/" {
		http.Redirect(w, r, "/maa-console", http.StatusSeeOther)
		return
	}

	target, err := maaTarget()
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		writeError(w, 503, "MAA 地址配置无效")
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.URL.Path = strings.TrimPrefix(r.URL.Path, "/maa")
		req.URL.RawPath = ""
		req.Host = target.Host
		req.Header.Del("Authorization")
		req.Header.Del("Cookie")
		req.Header.Set("X-Forwarded-Prefix", "/maa")
	}
	proxy.ModifyResponse = func(response *http.Response) error {
		response.Header.Set("Cache-Control", "no-store")
		response.Header.Set("X-Frame-Options", "SAMEORIGIN")
		response.Header.Set("Content-Security-Policy", "frame-ancestors 'self'")
		response.Header.Del("Set-Cookie")
		if location := response.Header.Get("Location"); strings.HasPrefix(location, "/") && !strings.HasPrefix(location, "/maa/") {
			response.Header.Set("Location", "/maa"+location)
		}
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeError(w, 502, "MAA 服务连接中断，请稍后重新连接")
	}
	proxy.ServeHTTP(w, r)
}
