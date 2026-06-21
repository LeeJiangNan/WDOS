// Package config 配置文件加载
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config 全局配置结构体，与 config/config.yaml 对应
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	CRIP     CRIPConfig     `mapstructure:"crip"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	MinIO    MinIOConfig    `mapstructure:"minio"`
	Wechat   WechatConfig   `mapstructure:"wechat"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	SLA      SLAConfig      `mapstructure:"sla"`
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Port string `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// CRIPConfig 鲲云 CRIP 平台对接配置
type CRIPConfig struct {
	CallbackPath        string `mapstructure:"callback_path"`
	OpenAPIBase         string `mapstructure:"openapi_base"`
	OpenAPIAppID        string `mapstructure:"openapi_app_id"`
	OpenAPIAppSecret    string `mapstructure:"openapi_app_secret"`
	HeartbeatInterval   int    `mapstructure:"heartbeat_interval"` // 心跳检测间隔，秒
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
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Prefix   string `mapstructure:"prefix"`
	DB       int    `mapstructure:"db"`
	Password string `mapstructure:"password"`
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

// Load 从 YAML 文件加载配置，自动替换环境变量
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &cfg, nil
}
