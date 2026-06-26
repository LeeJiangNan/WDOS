// Package config 配置文件加载
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 全局配置结构体，与 config/config.yaml 对应
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	CRIP     CRIPConfig     `mapstructure:"crip"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	COS      COSConfig      `mapstructure:"cos"`
	MinIO    MinIOConfig    `mapstructure:"minio"`
	Wechat   WechatConfig   `mapstructure:"wechat"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	SLA      SLAConfig      `mapstructure:"sla"`
	Seed     SeedConfig     `mapstructure:"seed"`
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Port        string `mapstructure:"port"`
	Mode        string `mapstructure:"mode"`
	CORSOrigins string `mapstructure:"cors_origins"` // 生产模式CORS白名单(逗号分隔)
}

// SeedConfig 初始化种子数据配置
type SeedConfig struct {
	AdminPassword string `mapstructure:"admin_password"` // 初始管理员密码
}

// CRIPConfig 鲲云 CRIP 平台对接配置
type CRIPConfig struct {
	CallbackPath        string `mapstructure:"callback_path"`
	OpenAPIBase         string `mapstructure:"openapi_base"`
	OpenAPIAppID        string `mapstructure:"openapi_app_id"`
	OpenAPIAppSecret    string `mapstructure:"openapi_app_secret"`
	HeartbeatInterval   int    `mapstructure:"heartbeat_interval"` // 心跳检测间隔，秒
	CallbackSecret      string `mapstructure:"callback_secret"`     // Callback HMAC 签名密钥
}

// DatabaseConfig MySQL 配置
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

// DSN 返回 MySQL 连接字符串
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Prefix   string `mapstructure:"prefix"`
	DB       int    `mapstructure:"db"`
	Password string `mapstructure:"password"`
}

// COSConfig 腾讯云对象存储配置（图片直存）
type COSConfig struct {
	SecretID  string `mapstructure:"secret_id"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	Region    string `mapstructure:"region"`
	PublicURL string `mapstructure:"public_url"` // https://bucket.cos.region.myqcloud.com
}

// MinIOConfig 对象存储配置
type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

// WechatConfig 微信小程序配置
type WechatConfig struct {
	AppID     string `mapstructure:"app_id"`
	AppSecret string `mapstructure:"app_secret"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret        string `mapstructure:"secret"`
	ExpireSeconds int    `mapstructure:"expire_seconds"`
}

// SLAConfig SLA 超时阈值配置
type SLAConfig struct {
	AcceptL1Seconds  int `mapstructure:"accept_l1_seconds"`
	AcceptL2Seconds  int `mapstructure:"accept_l2_seconds"`
	AcceptL3Seconds  int `mapstructure:"accept_l3_seconds"`
	ProcessL1Seconds int `mapstructure:"process_l1_seconds"`
	ProcessL2Seconds int `mapstructure:"process_l2_seconds"`
	ProcessL3Seconds int `mapstructure:"process_l3_seconds"`
}

// Load 从 YAML 文件加载配置，自动替换 ${VAR} 环境变量
func Load(path string) (*Config, error) {
	// 先读取原始内容，用 os.ExpandEnv 替换 ${VAR} 语法
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	expanded := os.ExpandEnv(string(raw))

	v := viper.New()
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	if err := v.ReadConfig(strings.NewReader(expanded)); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置结构失败: %w", err)
	}

	return &cfg, nil
}
