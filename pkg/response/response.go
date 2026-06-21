// Package response 统一 API 响应格式
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 预定义错误码
const (
	CodeSuccess       = 0     // 成功
	CodeBadRequest    = 40000 // 参数错误
	CodeOrderTaken    = 40001 // 工单已被他人接单
	CodeInvalidStatus = 40002 // 工单状态不允许此操作
	CodeNoPermission  = 40003 // 无权限操作此工单
	CodeInvalidTarget = 40004 // 转交目标用户无接单权限
	CodeUnauthorized  = 40100 // 未登录或 token 过期
	CodeTokenExpired  = 40101 // token 已过期无法刷新
	CodeForbidden     = 40300 // 无权限
	CodeNotFound      = 40400 // 资源不存在
	CodeUserNotFound  = 40401 // 用户不存在（手机号未匹配）
	CodeServerError   = 50000 // 服务器内部错误
)

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessMsg 带消息的成功响应
func SuccessMsg(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// Error 通用错误响应
func Error(c *gin.Context, httpCode int, bizCode int, message string) {
	c.JSON(httpCode, Response{
		Code:    bizCode,
		Message: message,
	})
}

// BadRequest 参数错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, CodeBadRequest, message)
}

// Unauthorized 未认证
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, CodeUnauthorized, message)
}

// Forbidden 无权限
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, CodeForbidden, message)
}

// NotFound 资源不存在
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, CodeNotFound, message)
}

// ServerError 服务器内部错误
func ServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, CodeServerError, message)
}
