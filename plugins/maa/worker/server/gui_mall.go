package server

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const guiNativeLogPath = "/root/.local/state/maa/debug/asst.log"

type guiNativeLogMark struct {
	info   os.FileInfo
	offset int64
}

func guiMarkNativeLog() guiNativeLogMark {
	info, err := os.Stat(guiNativeLogPath)
	if err != nil {
		return guiNativeLogMark{}
	}
	return guiNativeLogMark{info: info, offset: info.Size()}
}
func guiMallReport(mark guiNativeLogMark, params map[string]any) (string, bool) {
	if shopping, ok := params["shopping"].(bool); ok && !shopping {
		return "信用收取已结束；本次设置未启用购物", false
	}
	f, err := os.Open(guiNativeLogPath)
	if err != nil {
		return "", false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", false
	}
	start := int64(0)
	if mark.info != nil && os.SameFile(mark.info, info) && info.Size() >= mark.offset {
		start = mark.offset
	}
	if info.Size()-start > 2<<20 {
		start = info.Size() - (2 << 20)
	}
	if _, err = f.Seek(start, io.SeekStart); err != nil {
		return "", false
	}
	b, err := io.ReadAll(io.LimitReader(f, 2<<20))
	if err != nil {
		return "", false
	}
	return guiMallLogReport(string(b))
}

// Mall ignores some shopping subtask failures. Only a confirmed reward screen
// counts as a purchase; completing the outer task alone is not purchase proof.
func guiMallLogReport(raw string) (string, bool) {
	i := strings.LastIndex(raw, "append_task Mall ")
	if i < 0 {
		return "", false
	}
	raw = raw[i:]
	purchases, failures, retryFailures := 0, 0, 0
	pending := false
	for _, line := range strings.Split(raw, "\n") {
		_, callback, ok := strings.Cut(line, "append_callback | ")
		if !ok {
			continue
		}
		event, body, ok := strings.Cut(callback, " ")
		if !ok {
			continue
		}
		var v struct {
			Taskchain string   `json:"taskchain"`
			First     []string `json:"first"`
			Details   struct {
				Task string `json:"task"`
			} `json:"details"`
		}
		if json.Unmarshal([]byte(body), &v) != nil || v.Taskchain != "Mall" {
			continue
		}
		if event == "SubTaskError" {
			for _, first := range v.First {
				if first == "CreditShop-BuyIt" {
					retryFailures++
					if retryFailures == 4 {
						failures++
						retryFailures = 0
					}
					break
				}
			}
		}
		if event != "SubTaskStart" {
			continue
		}
		switch v.Details.Task {
		case "CreditShop-BuyIt", "WebMall-BuyItem", "WebMall-HireOperator":
			pending = true
		case "CreditShop-Bought", "WebMall-ReturnedToStore":
			if pending {
				purchases++
				retryFailures = 0
				pending = false
			}
		}
	}
	if failures > 0 || retryFailures > 0 {
		if strings.Contains(raw, "templ not found WebMall-Operator") {
			prefix := ""
			if purchases > 0 {
				prefix = fmt.Sprintf("已确认 %d 次购买；", purchases)
			}
			return prefix + "干员聘用后未能返回商店：插件缺少干员获得画面的识别模板，请更新插件资源后重试", true
		}
		if purchases > 0 {
			return fmt.Sprintf("已确认 %d 次购买，但部分商品的购买确认识别失败；请检查商店弹窗和遮挡商品的悬浮窗", purchases), true
		}
		return "购买确认识别失败，未确认购买成功；请检查商店弹窗和遮挡商品的悬浮窗后重试信用收支", true
	}
	if purchases > 0 {
		return fmt.Sprintf("已确认完成 %d 次信用购买", purchases), false
	}
	return "本次未确认新的信用购买；商品可能已售罄、信用不足或被购物条件筛选", false
}
