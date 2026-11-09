package server

import (
	"strings"
	"testing"
)

func TestMallReportSeparatesPurchaseProofFromTaskCompletion(t *testing.T) {
	start := "append_task Mall {}\n"
	event := func(kind, task string) string {
		return `append_callback | ` + kind + ` {"taskchain":"Mall","details":{"task":"` + task + `"}}` + "\n"
	}
	failure := `append_callback | SubTaskError {"taskchain":"Mall","first":["CreditShop-BuyIt"],"details":{}}` + "\n"
	success := event("SubTaskStart", "CreditShop-BuyIt") + event("SubTaskStart", "CreditShop-Bought")
	cases := []struct {
		name, log, want string
		warning         bool
	}{
		{"missing operator template", start + "templ not found WebMall-OperatorSkip.png\n" + failure, "插件缺少干员获得画面的识别模板", true},
		{"ignored shopping errors", start + failure + event("TaskChainCompleted", ""), "未确认购买成功", true},
		{"transient retry recovered", start + failure + success, "完成 1 次", false},
		{"unresolved item followed by success", start + strings.Repeat(failure, 4) + success, "已确认 1 次购买", true},
		{"partial purchases", start + success + failure, "已确认 1 次购买", true},
		{"repeated reward animation", start + success + event("SubTaskStart", "CreditShop-Bought"), "完成 1 次", false},
		{"wrapper is not proof", start + event("SubTaskStart", "CreditShop-BuyIt") + event("SubTaskCompleted", "CreditShop-BuyIt"), "未确认新的", false},
		{"hired operator returned to store", start + event("SubTaskStart", "WebMall-HireOperator") + event("SubTaskStart", "WebMall-ReturnedToStore"), "完成 1 次", false},
		{"old failure excluded", start + failure + start + success, "完成 1 次", false},
		{"other task failure ignored", start + `append_callback | SubTaskError {"taskchain":"Fight","first":["CreditShop-BuyIt"]}`, "未确认新的", false},
		{"stale log without current append", failure, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			message, warning := guiMallLogReport(tc.log)
			if warning != tc.warning || !strings.Contains(message, tc.want) {
				t.Fatalf("%q, warning=%v", message, warning)
			}
		})
	}
}
