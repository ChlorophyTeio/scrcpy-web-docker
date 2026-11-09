package server

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func guiAwardFailure(original error, mark guiNativeLogMark) error {
	f, err := os.Open(guiNativeLogPath)
	if err == nil {
		defer f.Close()
		if info, err := f.Stat(); err == nil {
			start := int64(0)
			if mark.info != nil && os.SameFile(mark.info, info) && info.Size() >= mark.offset {
				start = mark.offset
			}
			if info.Size()-start > 256*1024 {
				start = info.Size() - 256*1024
			}
			_, _ = f.Seek(start, io.SeekStart)
		}
		b, _ := io.ReadAll(io.LimitReader(f, 256*1024))
		if reason := guiAwardLogReason(string(b)); reason != "" {
			return errors.New(reason)
		}
	}
	return fmt.Errorf("领取奖励失败，请检查游戏当前界面与未关闭的弹窗（%w）", original)
}

func guiAwardLogReason(raw string) string {
	start := strings.LastIndex(raw, "append_task Award ")
	if start < 0 {
		return ""
	}
	raw = raw[start:]
	lastRecognition := ""
	if i := strings.LastIndex(raw, `"cur_task":`); i >= 0 {
		lastRecognition = strings.SplitN(raw[i:], "\n", 2)[0]
	}
	if strings.Contains(raw, `"taskchain":"Award"`) && strings.Contains(raw, "TaskChainError") && strings.Contains(lastRecognition, `"cur_task":"AwardBegin"`) {
		return "领取奖励未能识别当前界面；请关闭干员获得、购买结果等弹窗或返回游戏首页后重试"
	}
	return ""
}
