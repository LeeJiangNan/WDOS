// WDOS Callback Buffer — 腾讯云 SCF 云函数
// 接收 CRIP 回调 → 暂存到 COS → 昭阳 poller 拉取
//
// 环境变量（在 SCF 控制台配置）：
//   COS_BUCKET    — 存储桶名称，如 wdos-callback-1234567890
//   COS_REGION    — 地域，如 ap-guangzhou
//   COS_SECRET_ID — 腾讯云 SecretId
//   COS_SECRET_KEY— 腾讯云 SecretKey
//
// SCF HTTP 触发器监听 0.0.0.0:9000

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

var bucketURL *url.URL

func main() {
	// 从环境变量构建 COS Bucket URL
	bucket := os.Getenv("COS_BUCKET")
	region := os.Getenv("COS_REGION")
	if bucket == "" || region == "" {
		log.Fatalf("COS_BUCKET 和 COS_REGION 环境变量必须设置")
	}
	var err error
	bucketURL, err = url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region))
	if err != nil {
		log.Fatalf("COS Bucket URL 无效: %v", err)
	}

	// 路由：所有路径都走同一个 handler（CRIP 可能不管路径）
	http.HandleFunc("/", handleCallback)

	log.Printf("WDOS Callback Buffer 启动，COS: %s", bucketURL)
	log.Fatal(http.ListenAndServe("0.0.0.0:9000", nil))
}

// handleCallback 处理 CRIP 的 POST 回调
func handleCallback(w http.ResponseWriter, r *http.Request) {
	// 健康检查
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "wdos-callback-buffer"})
		return
	}

	// 只接受 POST
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"action": "error",
			"reason": "仅支持 POST 请求",
		})
		return
	}

	// 读取 Body
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 最大 10MB
	if err != nil {
		log.Printf("ERROR 读取请求体: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"action": "error",
			"reason": fmt.Sprintf("读取请求体失败: %v", err),
		})
		return
	}

	// 校验 JSON 格式
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("ERROR JSON 解析失败: %v, body=%s", err, truncate(string(body), 200))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"action": "error",
			"reason": fmt.Sprintf("JSON 格式错误: %v", err),
		})
		return
	}

	// 提取 snowflake_id 用于文件名
	snowflakeID := "unknown"
	if id, ok := data["snowflake_id"]; ok {
		snowflakeID = sanitize(fmt.Sprintf("%v", id))
	}

	// 生成唯一文件名：pending/{纳秒时间戳}_{snowflake_id}.json
	// 时间戳在前保证按时间排序
	filename := fmt.Sprintf("pending/%d_%s.json", time.Now().UnixNano(), snowflakeID)

	// 写入 COS
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := uploadToCOS(ctx, filename, body); err != nil {
		log.Printf("ERROR COS 写入失败: %v, file=%s", err, filename)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"action": "error",
			"reason": fmt.Sprintf("COS 写入失败: %v", err),
		})
		return
	}

	// 成功响应（模拟 WDOS callback 响应格式）
	log.Printf("OK buffered: %s (%d bytes)", filename, len(body))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"action":   "buffered",
		"reason":   "已暂存到 COS，昭阳端将拉取处理",
		"filename": filename,
	})
}

// uploadToCOS 上传 JSON 内容到 COS
func uploadToCOS(ctx context.Context, key string, body []byte) error {
	secretID := os.Getenv("COS_SECRET_ID")
	secretKey := os.Getenv("COS_SECRET_KEY")

	// 创建带认证的 COS 客户端（每次请求创建，避免并发问题）
	client := cos.NewClient(
		&cos.BaseURL{BucketURL: bucketURL},
		&http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  secretID,
				SecretKey: secretKey,
			},
			Timeout: 10 * time.Second,
		},
	)

	_, err := client.Object.Put(ctx, key, strings.NewReader(string(body)), nil)
	return err
}

// sanitize 清理文件名中的非法字符
func sanitize(s string) string {
	if s == "" {
		return "unknown"
	}
	// 只保留字母数字和常用符号
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_",
		"|", "_", " ", "_",
	)
	return replacer.Replace(s)
}

// truncate 截断字符串用于日志
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
