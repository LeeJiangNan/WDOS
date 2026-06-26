package alarm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/LeeJiangNan/WDOS/internal/model"
)

// CompensateRequest 手动补偿请求
type CompensateRequest struct {
	StartTime string `json:"start_time" binding:"required"` // 2026-06-14 08:00:00
	EndTime   string `json:"end_time" binding:"required"`   // 2026-06-14 12:00:00
}

// CompensateResult 补偿结果
type CompensateResult struct {
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	TotalFetched int    `json:"total_fetched"`
	NewlyAdded   int    `json:"newly_added"`
	Skipped      int    `json:"skipped"`
	Duration     string `json:"duration"`
}

// CRIPAuthResponse CRIP 认证响应
type CRIPAuthResponse struct {
	AccessToken string `json:"accessToken"`
}

// CRIPLogSearchRequest CRIP 日志搜索请求
type CRIPLogSearchRequest struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
}

// CRIPLogSearchResponse CRIP 日志搜索响应（推测结构，以实际 API 为准）
type CRIPLogSearchResponse struct {
	Data struct {
		Total int              `json:"total"`
		List  []model.CRIPCallback `json:"list"`
	} `json:"data"`
}

// Compensate 执行手动补偿
func (s *Service) Compensate(ctx context.Context, req *CompensateRequest) (*CompensateResult, error) {
	startTime := time.Now()

	s.sugar.Infof("手动补偿开始: %s ~ %s", req.StartTime, req.EndTime)

	// 1. 获取 CRIP token
	token, err := s.getCRIPToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 CRIP token 失败: %w", err)
	}

	// 2. 分页拉取 CRIP 报警日志
	var allAlarms []model.CRIPCallback
	page := 1
	pageSize := 100

	for {
		alarms, total, err := s.fetchCRIPLogs(ctx, token, req.StartTime, req.EndTime, page, pageSize)
		if err != nil {
			s.sugar.Warnf("拉取 CRIP 日志失败(page=%d): %v", page, err)
			break
		}
		allAlarms = append(allAlarms, alarms...)

		s.sugar.Infof("补偿进度: 第%d页, 本页%d条, 累计%d/%d条", page, len(alarms), len(allAlarms), total)

		// 用 total 判断是否已拉完所有数据（避免末页恰好满 pageSize 时多发空请求）
		if len(allAlarms) >= total {
			break
		}
		page++
	}

	// 3. 逐条去重并补漏
	newCount := 0
	skippedCount := 0

	for _, alarm := range allAlarms {
		if string(alarm.SnowflakeID) == "" {
			skippedCount++
			continue
		}

		// Redis 去重检查
		dedupKey := s.prefix + "alarm:" + string(alarm.SnowflakeID)
		ok, err := s.rdb.SetNX(ctx, dedupKey, "1", 24*time.Hour).Result()
		if err != nil || !ok {
			skippedCount++
			continue
		}

		// 调用 ProcessCallback 处理，标记来源为 compensation
		_, err = s.ProcessCallbackWithSource(ctx, &alarm, "compensation")
		if err != nil {
			s.sugar.Warnf("补偿处理失败(保留去重标记，避免重复入库): snowflake=%s, err=%v", string(alarm.SnowflakeID), err)
			// 保留 dedupKey，防止失败报警被重复拉取入库
			continue
		}
		newCount++
	}

	duration := time.Since(startTime).Round(time.Millisecond).String()

	s.sugar.Infof("手动补偿完成: 拉取%d条, 新增%d条, 跳过%d条, 耗时%s",
		len(allAlarms), newCount, skippedCount, duration)

	return &CompensateResult{
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		TotalFetched: len(allAlarms),
		NewlyAdded:   newCount,
		Skipped:      skippedCount,
		Duration:     duration,
	}, nil
}

// getCRIPToken 获取 CRIP OpenAPI 访问令牌
func (s *Service) getCRIPToken(ctx context.Context) (string, error) {
	cfg := s.cripCfg // 需要注入 CRIP 配置
	if cfg.OpenAPIAppID == "" || cfg.OpenAPIAppSecret == "" {
		return "", fmt.Errorf("CRIP OpenAPI 凭证未配置")
	}

	body, _ := json.Marshal(map[string]string{
		"app_id":     cfg.OpenAPIAppID,
		"app_secret": cfg.OpenAPIAppSecret,
	})

	url := cfg.OpenAPIBase + "/mc/v1/authenticate"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("CRIP 认证失败: HTTP %d, body: %s", resp.StatusCode, string(respBody))
	}

	var authResp struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		return "", fmt.Errorf("解析 CRIP 认证响应失败: %w", err)
	}

	return authResp.AccessToken, nil
}

// fetchCRIPLogs 分页拉取 CRIP 报警日志
func (s *Service) fetchCRIPLogs(ctx context.Context, token, startTime, endTime string, page, pageSize int) ([]model.CRIPCallback, int, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"start_time": startTime,
		"end_time":   endTime,
		"page":       page,
		"page_size":  pageSize,
	})

	url := s.cripCfg.OpenAPIBase + "/cb/v1/log/search"
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, 0, fmt.Errorf("CRIP 日志查询失败: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Total int                  `json:"total"`
			List  []model.CRIPCallback `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, 0, fmt.Errorf("解析 CRIP 日志响应失败: %w", err)
	}

	return result.Data.List, result.Data.Total, nil
}
