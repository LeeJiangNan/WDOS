// Package miniox MinIO 对象存储连接
package miniox

import (
	"context"
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Connect 连接 MinIO 并确保 bucket 存在
func Connect(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*minio.Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 MinIO 失败: %w", err)
	}

	// 确保 bucket 存在
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("检查 bucket 失败: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("创建 bucket 失败: %w", err)
		}
		log.Printf("[MinIO] bucket '%s' 已创建", bucket)
	}

	log.Println("[MinIO] 连接成功")
	return client, nil
}
