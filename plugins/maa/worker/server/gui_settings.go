package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"maa-server/config"
	"maa-server/scheduler"
)

func guiNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, !math.IsNaN(n) && !math.IsInf(n, 0)
	case int:
		return float64(n), true
	}
	return 0, false
}

func guiValidateSettings(c GUIConfig) error {
	if p, exists := c.Params["launch"]; exists {
		v, ok := p["package"].(string)
		if !ok || len(v) > 200 || !guiPackagePattern.MatchString(v) {
			return errors.New("启动游戏包名无效，例如 com.hypergryph.arknights")
		}
	}
	if p, exists := c.Params["update"]; exists {
		for _, key := range []string{"operbox", "depot"} {
			if _, ok := p[key].(bool); !ok {
				return errors.New("更新数据的干员/仓库开关无效")
			}
		}
		if p["operbox"] != true && p["depot"] != true {
			return errors.New("更新数据至少选择干员或仓库")
		}
		interval, ok := p["interval"].(string)
		if !ok || (interval != "every" && interval != "daily" && interval != "weekly") {
			return errors.New("更新数据的执行间隔无效")
		}
	}

	if c.TimeoutMinutes < 0 || c.TimeoutMinutes > 1440 {
		return errors.New("运行时长上限应为 0 到 1440 分钟，0 表示不限制")
	}
	seen := map[string]bool{}
	for _, id := range c.Order {
		if _, ok := guiTypes[id]; !ok || seen[id] {
			return errors.New("任务顺序包含无效或重复项目")
		}
		seen[id] = true
	}
	boolFields := strings.Fields("start_game_enabled DrGrandet replenish dorm_notstationed_enabled dorm_trust_enabled reception_message_board reception_clue_exchange reception_send_clue continue_training fiammetta_recovery_enabled use_pinus_sylvestris use_perception_information use_worldly_plight use_abyssal_hunter refresh expedite set_time visit_friends shopping force_shopping_if_credit_full only_buy_discount reserve_max_credit credit_fight award mail recruit orundum mining specialaccess signinevent use_support use_nonfriend_support investment_enabled stop_when_investment_full investment_with_more_score stop_at_final_boss stop_at_max_level start_with_elite_two only_start_with_elite_two refresh_trader_with_dice monthly_squad_auto_iterate monthly_squad_check_comms deep_exploration_auto_iterate clear_store formation use_sanity_potion add_trust ignore_requirements report_to_penguin report_to_yituliu")
	stringFields := strings.Fields("stage account_name drones filename theme squad roles core_char support_unit_name penguin_id yituliu_id server")
	listFields := strings.Fields("facility buy_first blacklist first_tags preserve_tags fiammetta_targets tools_to_craft")
	ranges := map[string][2]float64{
		"medicine_expire_days": {0, 30}, "threshold": {0, 1}, "level3_recruitment_permit_reserve": {0, 2147483647},
		"extra_tags_mode": {0, 2}, "times": {0, 2147483647}, "formation_index": {0, 4}, "plan_index": {0, 999},
		"difficulty": {-1, 99}, "starts_count": {1, 2147483647}, "investments_count": {0, 2147483647},
		"num_craft_batches": {1, 999}, "increment_mode": {0, 1}, "loop_times": {1, 2147483647}, "support_unit_usage": {0, 3},
	}
	for id, p := range c.Params {
		bad := func(key string) error { return fmt.Errorf("%s 参数 %s 格式或范围无效", guiNames[id], key) }
		for _, key := range boolFields {
			if v, exists := p[key]; exists {
				if _, ok := v.(bool); !ok {
					return bad(key)
				}
			}
		}
		for _, key := range stringFields {
			if v, exists := p[key]; exists {
				if s, ok := v.(string); !ok || len(s) > 4096 {
					return bad(key)
				}
			}
		}
		for _, key := range listFields {
			if v, exists := p[key]; exists {
				b, _ := json.Marshal(v)
				var list []string
				if json.Unmarshal(b, &list) != nil || list == nil || len(list) > 100 {
					return bad(key)
				}
			}
		}
		for key, limits := range ranges {
			if v, exists := p[key]; exists {
				n, ok := guiNumber(v)
				if !ok || n < limits[0] || n > limits[1] || (key != "threshold" && n != math.Trunc(n)) {
					return bad(key)
				}
			}
		}
		if v, exists := p["recruitment_time"]; exists {
			b, _ := json.Marshal(v)
			var times map[string]int
			if json.Unmarshal(b, &times) != nil || times == nil {
				return bad("recruitment_time")
			}
			for star, minutes := range times {
				if !strings.Contains("3456", star) || len(star) != 1 || minutes < 60 || minutes > 540 || minutes%10 != 0 {
					return bad("recruitment_time")
				}
			}
		}
		if id == "infrast" {
			if v, exists := p["mode"]; exists {
				n, ok := guiNumber(v)
				if !ok || (n != 0 && n != 10000 && n != 20000) {
					return bad("mode")
				}
				if n == 10000 {
					f, _ := p["filename"].(string)
					if f == "" {
						return errors.New("自定义基建需要上传并选择排班文件")
					}
				}
			}
		}
		if id == "rogue" {
			if v, exists := p["mode"]; exists {
				n, ok := guiNumber(v)
				allowed := map[float64]bool{0: true, 1: true, 4: true, 5: true, 6: true, 7: true, 10001: true, 20001: true, 30001: true, 30003: true}
				if !ok || !allowed[n] {
					return bad("mode")
				}
			}
		}
	}
	return nil
}

func guiInfrast(c *gin.Context) {
	if c.Request.Method == http.MethodGet {
		entries, _ := os.ReadDir(config.D.InfrastDir)
		files := []string{}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				files = append(files, entry.Name())
			}
		}
		c.JSON(200, gin.H{"files": files})
		return
	}
	if !scheduler.ExecutionMu.TryLock() {
		c.JSON(409, gin.H{"error": "任务运行时不能替换排班文件"})
		return
	}
	defer scheduler.ExecutionMu.Unlock()
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20)
	var value struct {
		Name string          `json:"name"`
		Plan json.RawMessage `json:"plan"`
	}
	if c.ShouldBindJSON(&value) != nil || value.Name == "" || len(value.Name) > 120 || filepath.Base(value.Name) != value.Name || strings.ContainsAny(value.Name, "/\\") || !strings.HasSuffix(value.Name, ".json") {
		c.JSON(400, gin.H{"error": "排班文件名或 JSON 格式无效"})
		return
	}
	var plan struct {
		Plans []json.RawMessage `json:"plans"`
	}
	if json.Unmarshal(value.Plan, &plan) != nil || len(plan.Plans) == 0 || len(plan.Plans) > 100 {
		c.JSON(400, gin.H{"error": "排班文件需要非空 plans 数组"})
		return
	}
	for _, p := range plan.Plans {
		var entry map[string]json.RawMessage
		if json.Unmarshal(p, &entry) != nil || entry == nil {
			c.JSON(400, gin.H{"error": "排班方案必须为 JSON 对象"})
			return
		}
	}
	if os.MkdirAll(config.D.InfrastDir, 0755) != nil {
		c.JSON(500, gin.H{"error": "无法创建排班目录"})
		return
	}
	f, err := os.CreateTemp(config.D.InfrastDir, ".gui-upload-*")
	if err != nil {
		c.JSON(500, gin.H{"error": "无法保存排班"})
		return
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	_, err = f.Write(value.Plan)
	closeErr := f.Close()
	if err != nil || closeErr != nil || os.Rename(tmp, filepath.Join(config.D.InfrastDir, value.Name)) != nil {
		c.JSON(500, gin.H{"error": "无法保存排班"})
		return
	}
	c.JSON(200, gin.H{"saved": true, "filename": value.Name, "plans": len(plan.Plans)})
}
