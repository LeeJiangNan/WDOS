// Package redisx Redis 连接
package redisx

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Connect 连接 Redis 并返回客户端
func Connect(addr, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Println("[Redis] 连接成功")
	return rdb, nil
}
