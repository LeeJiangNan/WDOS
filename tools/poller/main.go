// WDOS Callback Poller — 运行在昭阳 WSL2
// 定期轮询 COS，下载暂存的 CRIP 回调 JSON → POST 到本地 WDOS 后端
//
// 环境变量（在 systemd unit 或 .env 中配置）：
//   COS_BUCKET      — 存储桶名称
//   COS_REGION      — 地域
//   COS_SECRET_ID   — 腾讯云 SecretId
//   COS_SECRET_KEY  — 腾讯云 SecretKey
//   WDOS_URL        — 本地 WDOS callback 地址（默认 http://localhost:9090/api/v1/callback/crip）
//   POLL_INTERVAL   — 轮询间隔秒数（默认 10）
//   MAX_RETRIES     — 单文件最大重试次数（默认 3）

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// ============================================================
// 配置
// ============================================================

var (
	bucketURL    *url.URL
	wdosURL      string
	pollInterval time.Duration
	maxRetries   int
)

func loadConfig() {
	bucket := os.Getenv("COS_BUCKET")
	region := os.Getenv("COS_REGION")
	if bucket == "" || region == "" {
		log.Fatalf("必须设置 COS_BUCKET 和 COS_REGION 环境变量")
	}
	var err error
	bucketURL, err = url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region))
	if err != nil {
		log.Fatalf("COS Bucket URL 无效: %v", err)
	}

	wdosURL = os.Getenv("WDOS_URL")
	if wdosURL == "" {
		wdosURL = "http://localhost:9090/api/v1/callback/crip"
	}

	pollSec := 10
	if s := os.Getenv("POLL_INTERVAL"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 300 {
			pollSec = n
		}
	}
	pollInterval = time.Duration(pollSec) * time.Second

	maxRetries = 3
	if s := os.Getenv("MAX_RETRIES"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			maxRetries = n
		}
	}
}

// ============================================================
// COS 客户端
// ============================================================

func newCOSClient() *cos.Client {
	secretID := os.Getenv("COS_SECRET_ID")
	secretKey := os.Getenv("COS_SECRET_KEY")

	return cos.NewClient(
		&cos.BaseURL{BucketURL: bucketURL},
		&http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  secretID,
				SecretKey: secretKey,
			},
			Timeout: 15 * time.Second,
		},
	)
}

// ============================================================
// 主逻辑
// ============================================================

func main() {
	loadConfig()

	log.Printf("=== WDOS Callback Poller 启动 ===")
	log.Printf("COS:    %s", bucketURL.String())
	log.Printf("WDOS:   %s", wdosURL)
	log.Printf("间隔:   %v", pollInterval)
	log.Printf("重试:   %d 次", maxRetries)

	cosClient := newCOSClient()

	// 优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("收到信号 %v，正在退出...", sig)
		cancel()
	}()

	// 首次立即执行
	poll(ctx, cosClient)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			poll(ctx, cosClient)
		case <-ctx.Done():
			log.Println("Poller 已停止")
			return
		}
	}
}

// ============================================================
// 轮询逻辑
// ============================================================

func poll(ctx context.Context, client *cos.Client) {
	// 列出 pending/ 下的所有文件
	opt := &cos.BucketGetOptions{
		Prefix:  "pending/",
		MaxKeys: 100, // 一次最多处理 100 条
	}
	result, _, err := client.Bucket.Get(ctx, opt)
	if err != nil {
		log.Printf("ERROR 列出 COS 文件失败: %v", err)
		return
	}

	if len(result.Contents) == 0 {
		return // 没有待处理文件，静默跳过
	}

	// 按文件名排序（时间戳在前，保证先入先出）
	sort.Slice(result.Contents, func(i, j int) bool {
		return result.Contents[i].Key < result.Contents[j].Key
	})

	successCount := 0
	failCount := 0

	for _, obj := range result.Contents {
		if ctx.Err() != nil {
			return // 收到退出信号
		}

		retryCount := parseRetryCount(obj.Key)
		if retryCount > maxRetries {
			// 超过重试次数，移到死信目录
			log.Printf("DEAD %s (重试 %d 次，超过上限)", obj.Key, retryCount)
			moveToDead(ctx, client, obj.Key)
			continue
		}

		if err := processOne(ctx, client, obj.Key); err != nil {
			failCount++
			log.Printf("FAIL [%d/%d] %s: %v", failCount, len(result.Contents), obj.Key, err)

			// 标记重试次数：文件名加 .retryN 后缀
			if retryCount < maxRetries {
				markRetry(ctx, client, obj.Key, retryCount)
			}
		} else {
			successCount++
		}
	}

	if successCount > 0 || failCount > 0 {
		log.Printf("本轮完成: 成功 %d, 失败 %d", successCount, failCount)
	}
}

// ============================================================
// 单文件处理
// ============================================================

func processOne(ctx context.Context, client *cos.Client, key string) error {
	// 1. 从 COS 下载
	resp, err := client.Object.Get(ctx, key, nil)
	if err != nil {
		return fmt.Errorf("COS 下载失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 最大 10MB
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 2. 校验 JSON 格式（避免把非法数据 POST 给 WDOS）
	if !json.Valid(body) {
		// 坏数据直接移到 dead/，不浪费重试
		log.Printf("WARN %s: JSON 格式无效，移到 dead/", key)
		moveToDead(ctx, client, key)
		return fmt.Errorf("JSON 格式无效，已移到 dead/")
	}

	// 3. POST 到本地 WDOS
	wdosResp, err := http.Post(wdosURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("POST WDOS 失败: %w", err)
	}
	defer wdosResp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(wdosResp.Body, 4096))

	if wdosResp.StatusCode >= 400 {
		return fmt.Errorf("WDOS 返回 HTTP %d: %s", wdosResp.StatusCode, truncate(string(respBody), 200))
	}

	// 4. 成功 → 移到 processed/ 持久保留（不删除，用于灾备恢复）
	srcURL := fmt.Sprintf("%s/%s", bucketURL.Host, key)
	processedKey := strings.Replace(key, "pending/", "processed/", 1)
	processedKey = strings.Split(processedKey, ".retry")[0]
	if _, _, err := client.Object.Copy(ctx, processedKey, srcURL, nil); err != nil {
		log.Printf("WARN 移到 processed/ 失败: %v", err)
	} else {
		client.Object.Delete(ctx, key)
	}

	log.Printf("OK   %s → WDOS %d", key, wdosResp.StatusCode)
	return nil
}

// ============================================================
// 重试标记机制
// 文件名格式：pending/timestamp_snowflake_id.json
// 重试后重命名：pending/timestamp_snowflake_id.json.retry1
// ============================================================

// parseRetryCount 从文件名解析重试次数
func parseRetryCount(key string) int {
	parts := strings.Split(key, ".retry")
	if len(parts) < 2 {
		return 0
	}
	n, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return n
}

// markRetry 标记重试（重命名文件加 .retryN 后缀）
func markRetry(ctx context.Context, client *cos.Client, key string, currentRetry int) {
	newKey := strings.Split(key, ".retry")[0] + fmt.Sprintf(".retry%d", currentRetry+1)

	// COS 没有 rename 操作，需要 copy + delete
	srcURL := fmt.Sprintf("%s/%s", bucketURL.Host, key)
	_, _, err := client.Object.Copy(ctx, newKey, srcURL, nil)
	if err != nil {
		log.Printf("WARN 标记重试失败: %v", err)
		return
	}
	client.Object.Delete(ctx, key)
}

// ============================================================
// 死信处理
// ============================================================

// moveToDead 将文件移到 dead/ 目录
func moveToDead(ctx context.Context, client *cos.Client, key string) {
	// 提取文件名（去掉 pending/ 前缀）
	baseName := strings.TrimPrefix(key, "pending/")
	// 去掉 .retryN 后缀
	baseName = strings.Split(baseName, ".retry")[0]

	deadKey := fmt.Sprintf("dead/%s_%s", time.Now().Format("20060102"), baseName)

	srcURL := fmt.Sprintf("%s/%s", bucketURL.Host, key)
	_, _, err := client.Object.Copy(ctx, deadKey, srcURL, nil)
	if err != nil {
		log.Printf("WARN 移到 dead/ 失败: %v", err)
		return
	}
	client.Object.Delete(ctx, key)
	log.Printf("DEAD %s → %s", key, deadKey)
}

// ============================================================
// 工具函数
// ============================================================

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
