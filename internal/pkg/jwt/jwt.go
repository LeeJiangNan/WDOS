// Package jwt JWT 令牌生成与验证
package jwt

import (
	"fmt"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// Claims JWT payload
type Claims struct {
	UserID       uint64 `json:"user_id"`
	Role         string `json:"role"`
	DepartmentID uint64 `json:"department_id"`
	GroupID      uint64 `json:"group_id"`
	jwtlib.RegisteredClaims
}

// Manager JWT 管理器
type Manager struct {
	secret        []byte
	expireSeconds int
}

// New 创建 JWT 管理器
func New(secret string, expireSeconds int) *Manager {
	return &Manager{
		secret:        []byte(secret),
		expireSeconds: expireSeconds,
	}
}

// Generate 生成 token
func (m *Manager) Generate(userID uint64, role string, departmentID, groupID uint64) (string, int, error) {
	now := time.Now()
	exp := now.Add(time.Duration(m.expireSeconds) * time.Second)

	claims := &Claims{
		UserID:       userID,
		Role:         role,
		DepartmentID: departmentID,
		GroupID:      groupID,
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(exp),
			IssuedAt:  jwtlib.NewNumericDate(now),
			Issuer:    "wdos",
		},
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, fmt.Errorf("生成 JWT 失败: %w", err)
	}

	return tokenStr, m.expireSeconds, nil
}

// Parse 解析并验证 token
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	token, err := jwtlib.ParseWithClaims(tokenStr, &Claims{}, func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名算法: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("解析 JWT 失败: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("JWT 无效")
	}

	return claims, nil
}

// Refresh 刷新 token（在旧 token 过期前调用）
func (m *Manager) Refresh(oldTokenStr string) (string, int, error) {
	claims, err := m.Parse(oldTokenStr)
	if err != nil {
		return "", 0, fmt.Errorf("旧 token 无效，无法刷新: %w", err)
	}

	// 有效期剩余超过 1 天才允许刷新
	if time.Until(claims.ExpiresAt.Time) > 24*time.Hour {
		return "", 0, fmt.Errorf("token 有效期余量充足，无需刷新")
	}

	return m.Generate(claims.UserID, claims.Role, claims.DepartmentID, claims.GroupID)
}
