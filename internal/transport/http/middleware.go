// Package http HTTP 中间件
package http

import (
	"strings"

	jwtpkg "github.com/LeeJiangNan/WDOS/internal/pkg/jwt"
	"github.com/LeeJiangNan/WDOS/pkg/response"
	"github.com/gin-gonic/gin"
)

// JWTAuth JWT 鉴权中间件
func JWTAuth(jwtMgr *jwtpkg.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "缺少认证信息")
			c.Abort()
			return
		}

		// Bearer xxx
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "认证格式错误，应为 Bearer <token>")
			c.Abort()
			return
		}

		claims, err := jwtMgr.Parse(parts[1])
		if err != nil {
			response.Unauthorized(c, "token 无效或已过期")
			c.Abort()
			return
		}

		// 注入用户信息到 context
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("department_id", claims.DepartmentID)
		c.Set("group_id", claims.GroupID)

		c.Next()
	}
}

// RoleRequired 角色鉴权中间件（指定角色才能访问）
func RoleRequired(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			response.Forbidden(c, "无权限")
			c.Abort()
			return
		}

		role := userRole.(string)
		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "当前角色无权限访问")
		c.Abort()
	}
}
