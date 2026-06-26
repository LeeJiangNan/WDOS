// Package auth 认证服务
package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	jwtpkg "github.com/LeeJiangNan/WDOS/internal/pkg/jwt"
	"github.com/LeeJiangNan/WDOS/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service 认证服务
type Service struct {
	db      *gorm.DB
	jwtMgr  *jwtpkg.Manager
	appID   string // 微信 AppID
	secret  string // 微信 AppSecret
	sugar   *zap.SugaredLogger
}

// New 创建认证服务
func New(db *gorm.DB, jwtMgr *jwtpkg.Manager, appID, appSecret string, sugar *zap.SugaredLogger) *Service {
	return &Service{
		db:     db,
		jwtMgr: jwtMgr,
		appID:  appID,
		secret: appSecret,
		sugar:  sugar,
	}
}

// ========== 微信小程序登录 ==========

// WechatLoginRequest 微信登录请求
type WechatLoginRequest struct {
	Code string `json:"code" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	ExpiresIn   int          `json:"expires_in"`
	User        *model.User  `json:"user"`
}

// WechatLogin 微信小程序登录
func (s *Service) WechatLogin(code string) (*LoginResponse, error) {
	// 1. 用 code 换 openid
	openid, err := s.code2openid(code)
	if err != nil {
		return nil, fmt.Errorf("微信授权失败: %w", err)
	}

	// 2. 查用户表
	var user model.User
	err = s.db.Where("open_id = ? AND status = ?", openid, "active").First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户未注册，openid: %s", openid)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 3. 生成 JWT
	return s.generateToken(&user)
}

// code2openid 微信 code 换 openid
func (s *Service) code2openid(code string) (string, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		s.appID, s.secret, code)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取微信响应失败: %w", err)
	}
	var result struct {
		OpenID     string `json:"openid"`
		SessionKey string `json:"session_key"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("微信返回错误: %s (code=%d)", result.ErrMsg, result.ErrCode)
	}

	return result.OpenID, nil
}

// ========== Web 管理后台登录 ==========

// WebLoginRequest Web 登录请求
type WebLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// WebLogin 管理后台账号密码登录
func (s *Service) WebLogin(username, password string) (*LoginResponse, error) {
	// 查用户（管理员/经理等通过手机号或用户名登录）
	var user model.User
	err := s.db.Where("(name = ? OR phone = ? OR username = ?) AND status = ?", username, username, username, "active").First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户名或密码错误")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 验证密码（明文）
	if user.Password != password {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	return s.generateToken(&user)
}

// ========== Token 刷新 ==========

// RefreshToken 刷新 JWT
func (s *Service) RefreshToken(oldToken string) (*LoginResponse, error) {
	claims, err := s.jwtMgr.Parse(oldToken)
	if err != nil {
		return nil, fmt.Errorf("token 无效: %w", err)
	}

	// 查用户状态
	var user model.User
	if err := s.db.First(&user, claims.UserID).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}
	if user.Status != "active" {
		return nil, fmt.Errorf("用户已被禁用")
	}

	deptIDs := s.getUserDepartmentIDs(user.ID, user.DepartmentID)
	tokenStr, expiresIn, err := s.jwtMgr.Generate(user.ID, user.Role, user.DepartmentID, user.GroupID, deptIDs)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken: tokenStr,
		ExpiresIn:   expiresIn,
		User:        &user,
	}, nil
}

// ========== 内部方法 ==========

// getUserDepartmentIDs 获取用户所有部门ID
func (s *Service) getUserDepartmentIDs(userID uint64, primaryDeptID uint64) []uint64 {
	var uds []model.UserDepartment
	if err := s.db.Where("user_id = ?", userID).Find(&uds).Error; err != nil || len(uds) == 0 {
		return []uint64{primaryDeptID}
	}
	ids := make([]uint64, 0, len(uds))
	for _, ud := range uds {
		ids = append(ids, ud.DepartmentID)
	}
	return ids
}

func (s *Service) generateToken(user *model.User) (*LoginResponse, error) {
	deptIDs := s.getUserDepartmentIDs(user.ID, user.DepartmentID)
	tokenStr, expiresIn, err := s.jwtMgr.Generate(user.ID, user.Role, user.DepartmentID, user.GroupID, deptIDs)
	if err != nil {
		return nil, err
	}

	// 清除密码
	user.Password = ""

	return &LoginResponse{
		AccessToken: tokenStr,
		ExpiresIn:   expiresIn,
		User:        user,
	}, nil
}

// GetUserByID 根据 ID 查用户
func (s *Service) GetUserByID(id uint64) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ========== 密码工具 ==========

// HashPassword 明文存储（调试阶段）
func HashPassword(password string) (string, error) {
	return password, nil
}
