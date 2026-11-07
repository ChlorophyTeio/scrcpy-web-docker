package server

import (
	"errors"
	"io"
	"os"
	"strings"
)

func guiFightLogReason(text string) string {
	if i := strings.LastIndex(text, "append_task Fight "); i >= 0 {
		text = text[i:]
	}
	if strings.Contains(text, "UnsupportedLevel") || strings.Contains(text, "Not support stage") {
		return "关卡不受当前资源支持，请重新选择关卡或更新资源"
	}
	if !strings.Contains(text, "TaskChainError") {
		return ""
	}
	lastRecognition := ""
	if i := strings.LastIndex(text, `"to_be_recognized":`); i >= 0 {
		lastRecognition = strings.SplitN(text[i:], "\n", 2)[0]
	}
	if strings.Contains(lastRecognition, "SwitchTheme@ReturnTheme") && strings.Contains(lastRecognition, "Terminal-Entry") {
		return "游戏主题切换后返回失败，请在手机远控中处理登录或确认弹窗，返回游戏首页后重试"
	}
	if strings.Contains(text, `"what":"SanityNotEnough"`) {
		return "理智不足，且当前设置未允许补充理智"
	}
	return ""
}
func guiFightFailure(original error) error {
	f, e := os.Open("/root/.local/state/maa/debug/asst.log")
	if e == nil {
		defer f.Close()
		if info, e := f.Stat(); e == nil && info.Size() > 256*1024 {
			_, _ = f.Seek(-256*1024, io.SeekEnd)
		}
		b, _ := io.ReadAll(io.LimitReader(f, 256*1024))
		if reason := guiFightLogReason(string(b)); reason != "" {
			return errors.New("理智作战失败：" + reason)
		}
	}
	return errors.New("理智作战失败，请检查手机上的登录弹窗、关卡解锁和代理状态；详细原因见运行日志（" + original.Error() + "）")
}
