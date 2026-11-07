package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type guiStage struct {
	Code     string `json:"code"`
	Label    string `json:"label"`
	Category string `json:"category"`
	Detail   string `json:"detail"`
	Cost     int    `json:"cost,omitempty"`
	Open     bool   `json:"open"`
	Note     string `json:"note,omitempty"`
	Chapter  int    `json:"chapter,omitempty"`
	Minimum  string `json:"minimum,omitempty"`
}
type guiStageActivity struct {
	Tip           string
	StageName     string
	UtcStartTime  string
	UtcExpireTime string
	TimeZone      int
}

func (a guiStageActivity) active(now time.Time) bool {
	loc := time.FixedZone("activity", a.TimeZone*3600)
	start, e := time.ParseInLocation("2006/01/02 15:04:05", a.UtcStartTime, loc)
	end, f := time.ParseInLocation("2006/01/02 15:04:05", a.UtcExpireTime, loc)
	return e == nil && f == nil && !now.Before(start) && now.Before(end)
}

var stageNaturalDigits = regexp.MustCompile(`\d+`)
var stageMainCode = regexp.MustCompile(`^S?(\d+)-(\d+)$`)
var guiStageDataRoots = []string{"/root/.local/share/maa/resource", "/root/.local/share/maa/MaaResource/resource", "/root/.cache/maa/resource"}
var guiActivityPath = "/root/.cache/maa/StageActivityV2.json"
var guiStageRefreshMu sync.Mutex

func guiStageRead(name string, out any) bool {
	for i := len(guiStageDataRoots) - 1; i >= 0; i-- {
		b, e := os.ReadFile(filepath.Join(guiStageDataRoots[i], name))
		if e == nil && json.Unmarshal(b, out) == nil {
			return true
		}
	}
	return false
}
func guiNormalizeStage(value string) string {
	s := strings.TrimSpace(strings.NewReplacer("－", "-", "—", "-", "–", "-", "‑", "-").Replace(value))
	upper := strings.ToUpper(s)
	switch upper {
	case "ANNIHILATION":
		return "Annihilation"
	case "CHERNOBOG@ANNIHILATION":
		return "Chernobog@Annihilation"
	case "LUNGMENOUTSKIRTS@ANNIHILATION":
		return "LungmenOutskirts@Annihilation"
	case "LUNGMENDOWNTOWN@ANNIHILATION":
		return "LungmenDowntown@Annihilation"
	}
	for _, prefix := range []string{"CE", "LS", "AP", "CA", "SK"} {
		if upper == prefix+"5" || upper == prefix+"6" {
			return prefix + "-" + upper[len(prefix):]
		}
	}
	if strings.HasPrefix(upper, "SSREOPEN-") {
		return "SSReopen-" + strings.TrimPrefix(upper, "SSREOPEN-")
	}
	if regexp.MustCompile(`^[A-Za-z0-9-]+$`).MatchString(s) {
		return upper
	}
	return s
}
func guiBuildStages(now time.Time, client string) ([]guiStage, string) {
	var items map[string]struct {
		Name string `json:"name"`
	}
	guiStageRead("item_index.json", &items)
	name := func(id string) string {
		if n := items[id].Name; n != "" {
			return n
		}
		return id
	}
	out := []guiStage{{Code: "", Label: "当前 / 上次关卡", Category: "common", Detail: "不自动导航，使用游戏当前或上次关卡", Open: true}}
	// The resource stages list includes unsupported historical event maps. Only list
	// main/sub chapters here; activities must come from the official active list.
	var all []struct {
		Code  string `json:"code"`
		ID    string `json:"stageId"`
		Cost  int    `json:"apCost"`
		Drops []struct {
			ID   string `json:"itemId"`
			Type string `json:"dropType"`
		} `json:"dropInfos"`
	}
	guiStageRead("stages.json", &all)
	seen := map[string]bool{}
	for _, s := range all {
		m := stageMainCode.FindStringSubmatch(s.Code)
		if m == nil || !strings.HasPrefix(s.ID, "main_") && !strings.HasPrefix(s.ID, "sub_") || seen[s.Code] {
			continue
		}
		seen[s.Code] = true
		chapter, _ := strconv.Atoi(m[1])
		detail := []string{}
		for _, d := range s.Drops {
			if d.Type == "NORMAL_DROP" {
				detail = append(detail, name(d.ID))
			}
		}
		entry := guiStage{Code: s.Code, Label: s.Code, Category: "main", Detail: strings.Join(detail, "、"), Cost: s.Cost, Open: true, Chapter: chapter}
		if chapter >= 10 {
			entry.Code = s.Code + "-NORMAL"
			entry.Label = s.Code + " · 标准 / 常规"
			out = append(out, entry)
			entry.Code = s.Code + "-HARD"
			entry.Label = s.Code + " · 磨难 / 险地"
			out = append(out, entry)
		} else {
			out = append(out, entry)
		}
	}
	var activities map[string]struct {
		SideStory map[string]struct {
			Minimum  string `json:"MinimumRequired"`
			Activity guiStageActivity
			Stages   []struct{ Display, Value, Drop string }
		} `json:"sideStoryStage"`
		Resources guiStageActivity `json:"resourceCollection"`
	}
	b, e := os.ReadFile(guiActivityPath)
	activityOK := e == nil && json.Unmarshal(b, &activities) == nil
	if client == "Bilibili" || client == "" {
		client = "Official"
	}
	a := activities[client]
	loc, _ := time.LoadLocation("Asia/Shanghai")
	day := int(now.In(loc).Add(-4 * time.Hour).Weekday())
	allOpen := a.Resources.active(now)
	for _, s := range []struct {
		code, label string
		days        []int
	}{{"CE-6", "龙门币", []int{2, 4, 6, 0}}, {"LS-6", "作战记录", []int{0, 1, 2, 3, 4, 5, 6}}, {"AP-5", "采购凭证", []int{1, 4, 6, 0}}, {"CA-5", "技能概要", []int{2, 3, 5, 0}}, {"SK-5", "碳素材", []int{1, 3, 5, 6}}, {"PR-A", "重装 / 医疗芯片", []int{1, 4, 5, 0}}, {"PR-B", "狙击 / 术师芯片", []int{1, 2, 5, 6}}, {"PR-C", "先锋 / 辅助芯片", []int{3, 4, 6, 0}}, {"PR-D", "近卫 / 特种芯片", []int{2, 3, 6, 0}}} {
		open := allOpen
		days := []string{}
		for _, d := range s.days {
			if d == day {
				open = true
			}
			days = append(days, []string{"日", "一", "二", "三", "四", "五", "六"}[d])
		}
		note := "周" + strings.Join(days, "、") + "开放（04:00 换日）"
		codes := []string{s.code}
		category := "supplies"
		if strings.HasPrefix(s.code, "PR-") {
			codes = []string{s.code + "-1", s.code + "-2"}
			category = "chips"
		}
		for _, code := range codes {
			out = append(out, guiStage{Code: code, Label: code, Category: category, Detail: s.label, Open: open, Note: note})
		}
	}
	for _, s := range []struct{ code, label string }{{"Annihilation", "当期剿灭"}, {"Chernobog@Annihilation", "切尔诺伯格"}, {"LungmenOutskirts@Annihilation", "龙门外环"}, {"LungmenDowntown@Annihilation", "龙门市区"}} {
		out = append(out, guiStage{Code: s.code, Label: s.label, Category: "annihilation", Detail: "需已解锁，并确认代理配置", Open: true})
	}
	for _, code := range []string{"OF-1", "OF-F3"} {
		out = append(out, guiStage{Code: code, Label: code, Category: "side", Detail: "火蓝之心 · 常驻别传", Open: true})
	}
	for _, activity := range a.SideStory {
		if !activity.Activity.active(now) {
			continue
		}
		for _, s := range activity.Stages {
			out = append(out, guiStage{Code: s.Value, Label: s.Display, Category: "activity", Detail: activity.Activity.StageName + " · " + name(s.Drop), Open: true, Note: activity.Activity.Tip + " · 截止 " + activity.Activity.UtcExpireTime, Minimum: activity.Minimum})
		}
	}
	// Main stages are naturally sorted; don't depend on JSON map iteration order.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		if out[i].Chapter != out[j].Chapter {
			return out[i].Chapter < out[j].Chapter
		}
		return stageNatural(out[i].Code) < stageNatural(out[j].Code)
	})
	note := "关卡来自已安装资源；开放日按北京时间 04:00 换日，解锁与代理状态以游戏为准。"
	if !activityOK {
		note += "活动列表暂不可用，请刷新关卡数据。"
	}
	return out, note
}
func stageNatural(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "-NORMAL", "-A"), "-HARD", "-B")
	return stageNaturalDigits.ReplaceAllStringFunc(s, func(v string) string { n, _ := strconv.Atoi(v); return fmt.Sprintf("%06d", n) })
}
func guiStageRefresh(ctx context.Context) error {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if p := coreProxy(); p != "" {
		u, e := url.Parse(p)
		if e != nil {
			return e
		}
		tr.Proxy = http.ProxyURL(u)
	}
	defer tr.CloseIdleConnections()
	c := http.Client{Transport: tr, Timeout: 20 * time.Second}
	req, e := http.NewRequestWithContext(ctx, "GET", "https://api.maa.plus/MaaAssistantArknights/api/gui/StageActivityV2.json", nil)
	if e != nil {
		return e
	}
	r, e := c.Do(req)
	if e != nil {
		return e
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", r.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if e != nil {
		return e
	}
	var v map[string]json.RawMessage
	if json.Unmarshal(b, &v) != nil || v["Official"] == nil {
		return fmt.Errorf("活动数据格式无效")
	}
	if e = os.WriteFile(guiActivityPath+".web.tmp", b, 0600); e != nil {
		return e
	}
	return os.Rename(guiActivityPath+".web.tmp", guiActivityPath)
}
func guiStages(c *gin.Context) {
	client := c.DefaultQuery("client", "Official")
	if client != "Official" && client != "Bilibili" {
		c.JSON(400, gin.H{"error": "客户端无效"})
		return
	}
	if c.Request.Method == http.MethodPost {
		guiStageRefreshMu.Lock()
		e := guiStageRefresh(c.Request.Context())
		guiStageRefreshMu.Unlock()
		if e != nil {
			c.JSON(502, gin.H{"error": "关卡数据刷新失败，已保留本地列表"})
			return
		}
	}
	items, note := guiBuildStages(time.Now(), client)
	var updated string
	if info, e := os.Stat(guiActivityPath); e == nil {
		updated = info.ModTime().Format(time.RFC3339)
	}
	c.JSON(200, gin.H{"items": items, "note": note, "updated_at": updated, "timezone": "Asia/Shanghai"})
}
